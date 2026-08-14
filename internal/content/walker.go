package content

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
)

const walkWorkersNum = 20

type result struct {
	post *Post
	err  error
}

func (c *Content) fsPosts(root string) (map[string]*Post, error) {
	done := make(chan struct{})
	defer close(done)

	paths, errc := c.walkRepo(done, root)

	result := make(chan result)

	var wg sync.WaitGroup
	for range walkWorkersNum {
		wg.Go(func() {
			c.getPost(done, paths, result)
		})
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	posts := map[string]*Post{}
	for r := range result {
		if r.err != nil {
			return nil, r.err // TODO: should return?
		}

		if _, dup := posts[r.post.Slug]; dup {
			return nil, fmt.Errorf("duplicate slug %q", r.post.Slug)
		}
		posts[r.post.Slug] = r.post
	}

	if err := <-errc; err != nil {
		return nil, err
	}

	return posts, nil
}

func (c *Content) walkRepo(done <-chan struct{}, root string) (<-chan string, <-chan error) {
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

			if !d.Type().IsRegular() || filepath.Base(path) != c.contentFilename {
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

func (c *Content) getPost(done <-chan struct{}, paths <-chan string, ch chan<- result) {
	for path := range paths {
		post, err := c.decodePost(path)
		if err != nil {
			err = fmt.Errorf("%s: %w", path, err)
		}
		select {
		case ch <- result{post, err}:
		case <-done:
			return
		}
	}
}

func (c *Content) decodePost(path string) (*Post, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var (
		dir       = filepath.Dir(path)
		dirName   = filepath.Base(dir)
		mediaBase = "/media/" + dirName
	)

	mdCtx := parser.NewContext()
	mdCtx.Set(mediaBaseKey, mediaBase)

	var buf bytes.Buffer
	if err := c.md.Convert(content, &buf, parser.WithContext(mdCtx)); err != nil {
		slog.Error("failed to convert markdown to html", "error", err)
		return nil, err
	}

	fmData := frontmatter.Get(mdCtx)
	if fmData == nil {
		return nil, errors.New("missing front matter")
	}

	var fm FrontMatter
	if err := fmData.Decode(&fm); err != nil {
		return nil, err
	}

	if cover := rewriteAsset(fm.CoverImage, mediaBase); cover != "" {
		fm.CoverImage = cover
	}

	post := &Post{
		HTML:        buf.String(),
		Summary:     excerpt(buf.String(), SummaryLength),
		FrontMatter: fm,
		// ContentHash: md5.Sum(content),
	}

	if err := ValidatePost(post); err != nil {
		return nil, err
	}

	// The post's on-disk folder name must equal its slug so that /media/<dir> URLs and
	// /post/<slug> pages stay in sync.
	if dirName != fm.Slug {
		return nil, fmt.Errorf("post directory %q must match slug %q", dirName, fm.Slug)
	}

	return post, nil
}
