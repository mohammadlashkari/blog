package content

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	img64 "github.com/tenkoh/goldmark-img64"
	"github.com/yuin/goldmark"

	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/mermaid"
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
		md:              newMDParser(),
	}
}

// postDirKey carries a post's on-disk directory through the parser context (set per
// Convert) so absPathResolver can turn relative image paths into absolute filesystem
// paths without any per-post state on the shared goldmark instance.
var postDirKey = parser.NewContextKey()

// absPathResolver rewrites relative image destinations to absolute filesystem paths at
// parse time, using the post directory from the parser context. This moves the only
// per-post state (which img64's ParentLocalPathResolver used to hold at construction)
// into a per-Convert context, letting a single parser be shared across workers. img64's
// renderer then reads the already-absolute path and base64-inlines the image.
type absPathResolver struct{}

func (absPathResolver) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	dir, _ := pc.Get(postDirKey).(string)
	if dir == "" {
		return
	}

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		img, ok := n.(*ast.Image)
		if !ok {
			return ast.WalkContinue, nil
		}
		src := string(img.Destination)
		if src != "" && !isRemote(src) && !strings.HasPrefix(src, "data:") && !filepath.IsAbs(src) {
			img.Destination = []byte(filepath.Join(dir, src))
		}
		return ast.WalkSkipChildren, nil
	})
}

func isRemote(src string) bool {
	return strings.HasPrefix(src, "http://") ||
		strings.HasPrefix(src, "https://") ||
		strings.HasPrefix(src, "//")
}

func newMDParser() goldmark.Markdown {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Linkify,
			extension.Footnote,
			extension.Typographer,
			img64.Img64, // default (identity) path resolver; paths resolved by absPathResolver
			highlighting.Highlighting,
			&mermaid.Extender{},
			&frontmatter.Extender{},
		),
		goldmark.WithParserOptions(
			parser.WithAttribute(),
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(util.Prioritized(absPathResolver{}, 100)),
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
