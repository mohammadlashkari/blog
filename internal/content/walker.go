package content

import (
	"bytes"
	"crypto/sha256"
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
			c.getFm(done, paths, result)
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

func (c *Content) getFm(done <-chan struct{}, paths <-chan string, ch chan<- result) {
	for path := range paths {
		post, err := c.decodeFM(path)
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

func (c *Content) decodeFM(path string) (*Post, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	mdCtx := parser.NewContext()

	var html bytes.Buffer
	if err := c.md.Convert(content, &html, parser.WithContext(mdCtx)); err != nil {
		slog.Error("failed to convert markdown to html", "error", err)
		return nil, err
	}

	var fm FrontMatter
	if err := frontmatter.Get(mdCtx).Decode(&fm); err != nil {
		return nil, err
	}

	post := &Post{
		HTML:        html.String(),
		ContentHash: sha256.Sum256(content),
		FrontMatter: fm,
	}

	if err := ValidatePost(post); err != nil {
		return nil, err
	}

	return post, nil
}
