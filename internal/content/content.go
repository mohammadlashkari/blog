package content

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/frontmatter"
)

type Content struct {
	localPath       string
	remotePath      string
	postsPath       string
	branch          string
	contentFilename string
	md              goldmark.Markdown
}

func New(localPath, remotePath, branch, contentFilename string) *Content {
	return &Content{
		localPath:       localPath,
		remotePath:      remotePath,
		branch:          branch,
		postsPath:       filepath.Join(localPath, "posts"),
		contentFilename: contentFilename,
		md:              mdParser(),
	}
}

func mdParser() goldmark.Markdown {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Linkify,
			&frontmatter.Extender{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(),
		),
	)

	return md
}

// Build syncs the repo and always returns a freshly built index.
func (c *Content) Build(ctx context.Context) (*Index, error) {
	if _, err := c.reconcile(ctx); err != nil {
		return nil, err
	}

	idx, err := c.buildIndex()
	if err != nil {
		slog.ErrorContext(ctx, "content build failed", "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "content build succeeded")
	return idx, nil
}

// Refresh syncs the repo and rebuilds the index only when the content changed.
// It returns a nil index and nil error when nothing changed.
func (c *Content) Refresh(ctx context.Context) (*Index, error) {
	changed, err := c.reconcile(ctx)
	if err != nil {
		return nil, err
	}
	if !changed {
		slog.InfoContext(ctx, "content unchanged, skipping refresh")
		return nil, nil
	}

	idx, err := c.buildIndex()
	if err != nil {
		slog.ErrorContext(ctx, "content refresh failed", "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "content refresh succeeded")
	return idx, nil
}
