package post

import (
	"blog/internal/ui"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"
	"github.com/yuin/goldmark/parser"
	"gopkg.in/yaml.v3"
)

func (s *Service) handlePostsIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var (
		posts []Post
		err   error
	)

	tagsFilter := r.URL.Query()["tag"]

	if len(tagsFilter) > 0 {
		posts, err = s.store.ListPostsByTag(ctx, ListPostsByTagParams{
			TagNames:   tagsFilter,
			IncludeAll: true,
		})
	} else {
		posts, err = s.store.ListPosts(ctx, true)
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to list posts", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tags, err := s.store.ListTags(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list tags", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	en, fa := groupPostsByLanguageAndYear(posts)
	render(w, r, PostsIndexPage(en, fa, tags))
}

func (s *Service) handlePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slug := r.PathValue("slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	post, err := s.store.PostBySlug(ctx, PostBySlugParams{
		Slug:       slug,
		IncludeAll: true,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "failed to get post", "slug", slug, "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tags, err := s.store.TagsByPostID(ctx, post.ID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.ErrorContext(ctx, "failed to get post's tags", "slug", slug, "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	postPath := filepath.Join(s.cfg.LocalContentRepo, post.Filename)
	contentMD, err := os.ReadFile(postPath)
	if err != nil {
		slog.ErrorContext(
			ctx, "failed to read post's markdown file",
			"slug", slug,
			"path", postPath,
			"error", err,
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	parserCtx := parser.NewContext()

	var buf bytes.Buffer
	if err := s.md.Convert(contentMD, &buf, parser.WithContext(parserCtx)); err != nil {
		slog.ErrorContext(ctx, "failed to convert markdown to html", "slug", slug, "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	contentHTML := buf.String()

	var component templ.Component
	if post.Slug == "why-linux-fa" {
		component = ui.GameOfLifeEmbed()
	}

	uiPost := buildUIPost(post)
	render(w, r, PostPage(uiPost, tags, contentHTML, component))
}

type GitHubPushPayload struct {
	Ref        string     `json:"ref"`
	Repository Repository `json:"repository"`
}

type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	CloneURL string `json:"clone_url"`
}

func (s *Service) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read webhook body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if !verifyGitHubSignature(body, r.Header.Get("X-Hub-Signature-256"), s.cfg.WebhookSecret) {
		slog.ErrorContext(ctx, "invalid webhook signature")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload GitHubPushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.ErrorContext(ctx, "failed to parse webhook payload", "error", err)
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	if payload.Ref != "refs/heads/master" {
		w.WriteHeader(http.StatusOK)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(ctx, "git", "pull", "origin", "master")
		cmd.Dir = s.cfg.LocalContentRepo

		output, err := cmd.CombinedOutput()
		if err != nil {
			slog.ErrorContext(
				ctx,
				"git pull failed",
				"path", cmd.Dir, "error", err, "output", string(output),
			)
			return
		}
		slog.InfoContext(ctx, "git pull succeeded")

		s.sync(ctx)
	}()
}

func (s *Service) sync(ctx context.Context) error {
	posts, err := s.store.ListPosts(ctx, true)
	if err != nil {
		return err
	}

	dbPosts := make(map[string]string, len(posts))
	for _, p := range posts {
		dbPosts[p.Slug] = p.ContentHash
	}

	fms, err := getFMs(s.cfg.LocalContentRepo)
	if err != nil {
		return err
	}

	fsPosts := make(map[string]string, len(posts))
	for _, f := range fms {
		fsPosts[f.slug] = f.contentHash
	}

	// Reconciliation
	for slug, hash := range fsPosts {
		dbHash, ok := dbPosts[slug]
		if !ok {
			// Insert
		}
		if hash != dbHash {
			// Update
		}
	}

	return nil
}

type FrontMatter struct {
	filename    string     `yaml:"filename"`
	title       string     `yaml:"title"`
	slug        string     `yaml:"slug"`
	language    Language   `yaml:"language"`
	coverImage  *string    `yaml:"cover_image"`
	publishedAt *time.Time `yaml:"published_at"`
	tags        []string   `yaml:"tags"`
	contentHash string
}

type result struct {
	fm  *FrontMatter
	err error
}

func getFMs(root string) ([]*FrontMatter, error) {
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

	fms := []*FrontMatter{}
	for r := range result {
		if r.err != nil {
			return nil, r.err
		}

		fms = append(fms, r.fm)
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

			if !d.Type().IsRegular() || filepath.Ext(path) != ".md" {
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

	fm.contentHash = hash

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
