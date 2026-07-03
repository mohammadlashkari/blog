package content

import (
	"context"
	"path/filepath"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/frontmatter"
)

type Content struct {
	localPath  string
	remotePath string
	postsPath  string
	branch     string
	md         goldmark.Markdown
}

func New(localPath, remotePath, branch string) *Content {
	c := &Content{
		localPath:  localPath,
		remotePath: remotePath,
		branch:     branch,
		postsPath:  filepath.Join(localPath, "posts"),
		md:         mdParser(),
	}

	return c
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

func (c *Content) Build(ctx context.Context) (*Index, error) {
	if err := c.reconcile(ctx); err != nil {
		return nil, err
	}

	return c.buildIndex()
}
