package post

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

func (s *Service) syncBlog(ctx context.Context) error {
	posts, err := s.store.ListPosts(ctx, true)
	if err != nil {
		return fmt.Errorf("list posts: %w", err)
	}
	dbPosts := make(map[string]Post, len(posts))
	for _, p := range posts {
		dbPosts[p.Slug] = p
	}

	root := filepath.Join(s.cfg.LocalContentRepo, "posts")
	fsPosts, err := fsPosts(root)
	if err != nil {
		return fmt.Errorf("walk fs posts: %w", err)
	}

	var errs []error
	for slug, fm := range fsPosts {
		dbHash, ok := dbPosts[slug]
		delete(dbPosts, slug)

		if ok && fm.ContentHash == dbHash.ContentHash {
			slog.InfoContext(ctx, "no change", "slug", fm.Slug)
			continue
		}

		slog.InfoContext(ctx, "changed", "slug", fm.Slug)

		if err := s.syncPost(ctx, slug, fm); err != nil {
			errs = append(errs, err)
		}
	}

	if len(dbPosts) > 0 {
		deleteSlugs := make([]string, 0, len(dbPosts))
		for slug := range dbPosts {
			deleteSlugs = append(deleteSlugs, slug)
		}
		if err := s.store.DeletePosts(ctx, deleteSlugs); err != nil {
			slog.ErrorContext(ctx, "failed to delete posts", "slugs", deleteSlugs, "error", err)
			errs = append(errs, fmt.Errorf("delete posts %v: %w", deleteSlugs, err))
		}
	}

	return errors.Join(errs...)
}

// syncPost upserts a single post and reconciles its tags inside one
// transaction, so a partial failure never leaves a post with an updated
// content hash but stale tags (which the hash-skip check would hide forever).
func (s *Service) syncPost(ctx context.Context, slug string, fm *FrontMatter) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for post %q: %w", slug, err)
	}
	defer tx.Rollback()

	qtx := New(tx)

	var coverImage *string
	if fm.CoverImage != "" {
		coverImage = &fm.CoverImage
	}

	postID, err := qtx.UpsertPost(ctx, UpsertPostParams{
		Slug:        slug,
		Title:       fm.Title,
		CoverImage:  coverImage,
		Language:    fm.Language,
		ContentHash: fm.ContentHash,
		PublishedAt: fm.PublishedAt,
	})
	if err != nil {
		return fmt.Errorf("upsert post %q: %w", slug, err)
	}

	if err := s.syncTags(ctx, qtx, postID, slug, fm.Tags); err != nil {
		return err // already wrapped in syncTags
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx for post %q: %w", slug, err)
	}

	return nil
}

func (s *Service) syncTags(ctx context.Context, qtx *Queries, postID int64, slug string, fsTags []string) error {
	dbTags, err := qtx.ListTagsForPost(ctx, postID)
	if err != nil {
		return fmt.Errorf("list tags for post %q: %w", slug, err)
	}

	toAdd, toRemove := diffTags(dbTags, fsTags)

	fmt.Println("----", dbTags, "-----", fsTags)

	if len(toAdd) > 0 {
		tagIDs, err := s.upsertTags(ctx, qtx, toAdd)
		if err != nil {
			return fmt.Errorf("upsert tags %v for post %q: %w", toAdd, slug, err)
		}
		if err := s.addPostTags(ctx, qtx, postID, tagIDs); err != nil {
			return fmt.Errorf("add post tags for post %q: %w", slug, err)
		}
		slog.InfoContext(ctx, "create tags", "tags", toAdd)
	}

	if len(toRemove) > 0 {
		// if err := qtx.RemovePostTagsByName(ctx, RemovePostTagsByNameParams{
		// 	PostID: postID,
		// 	Names:  toRemove,
		// }); err != nil {
		// 	return fmt.Errorf("remove post tags %v for post %q: %w", toRemove, slug, err)
		// }
		//
		// slog.InfoContext(ctx, "remove tags", "tags", toRemove)
	}

	return nil
}

func (s *Service) upsertTags(ctx context.Context, qtx *Queries, names []string) ([]int64, error) {
	ids := make([]int64, 0, len(names))
	for _, name := range names {
		id, err := qtx.UpsertTag(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("upsert tag %q: %w", name, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Service) addPostTags(ctx context.Context, qtx *Queries, postID int64, tagIDs []int64) error {
	for _, tagID := range tagIDs {
		if err := qtx.AddPostTag(ctx, AddPostTagParams{
			PostID: postID,
			TagID:  tagID,
		}); err != nil {
			return fmt.Errorf("add post_tag (post=%d, tag=%d): %w", postID, tagID, err)
		}
	}
	return nil
}

func diffTags(dbTags []Tag, fsTags []string) (toAdd, toRemove []string) {
	have := make(map[string]struct{}, len(dbTags))
	for _, tag := range dbTags {
		have[tag.Name] = struct{}{}
	}

	want := make(map[string]struct{}, len(fsTags))
	for _, name := range fsTags {
		want[name] = struct{}{}
	}

	for t := range want {
		if _, ok := have[t]; !ok {
			toAdd = append(toAdd, t)
		}
	}

	for t := range have {
		if _, ok := want[t]; !ok {
			toRemove = append(toRemove, t)
		}
	}

	return
}

type FrontMatter struct {
	Title       string     `yaml:"title"`
	Slug        string     `yaml:"slug"`
	CoverImage  string     `yaml:"cover_image"`
	Language    Language   `yaml:"language"`
	PublishedAt *time.Time `yaml:"published_at"`
	Tags        []string   `yaml:"tags"`
	ContentHash string
}

type result struct {
	fm  *FrontMatter
	err error
}

func fsPosts(root string) (map[string]*FrontMatter, error) {
	done := make(chan struct{})
	defer close(done)

	paths, errc := walkRepo(done, root)

	const numWorkers = 20
	result := make(chan result)

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Go(func() {
			getFm(done, paths, result)
		})
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	fms := map[string]*FrontMatter{}
	for r := range result {
		if r.err != nil {
			return nil, r.err // TODO: should return?
		}

		fms[r.fm.Slug] = r.fm
	}

	if err := <-errc; err != nil {
		return nil, err
	}

	return fms, nil
}

func walkRepo(done <-chan struct{}, root string) (<-chan string, <-chan error) {
	var (
		paths = make(chan string)
		errc  = make(chan error, 1)
	)

	go func() {
		defer close(paths)

		errc <- filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.Type().IsRegular() || filepath.Base(path) != "index.md" {
				return nil
			}

			select {
			case paths <- path:
				return nil
			case <-done:
				return errors.New("walked canceled")
			}

		})
	}()

	return paths, errc
}

func getFm(done <-chan struct{}, paths <-chan string, c chan<- result) {
	for path := range paths {
		fm, err := decodeFM(path)
		select {
		case c <- result{fm, err}:
		case <-done:
			return
		}
	}
}

func decodeFM(path string) (*FrontMatter, error) {
	data, err := os.ReadFile(path) // TODO: should i read chunk by chunk?
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])

	parts := bytes.SplitN(data, []byte("---\n"), 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid %q post format", path)
	}

	fmBytes := parts[1]

	var fm FrontMatter
	if err := yaml.Unmarshal(fmBytes, &fm); err != nil {
		return nil, err
	}

	fm.ContentHash = hash

	return &fm, nil
}

func verifyGitHubSignature(payload []byte, signature, secret string) bool {
	if signature == "" || secret == "" {
		return false
	}

	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}
