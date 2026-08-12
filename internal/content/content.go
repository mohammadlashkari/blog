package content

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
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

// mediaBaseKey carries the per-post media URL base (e.g. "/media/<dir>") through the
// parser context so mediaResolver can rewrite relative asset paths.
var mediaBaseKey = parser.NewContextKey()

// mediaResolver rewrites relative image destinations to absolute /media URLs at parse
// time. The base comes from the parser context (set per Convert).
type mediaResolver struct{}

func (mediaResolver) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	base, _ := pc.Get(mediaBaseKey).(string)

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		img, ok := n.(*ast.Image)
		if !ok {
			return ast.WalkContinue, nil
		}
		// Defer offscreen images so long, image-heavy posts load fast.
		img.SetAttributeString("loading", []byte("lazy"))
		img.SetAttributeString("decoding", []byte("async"))
		if base != "" {
			if dst := rewriteAsset(string(img.Destination), base); dst != "" {
				img.Destination = []byte(dst)
			}
		}
		return ast.WalkSkipChildren, nil
	})
}

// rewriteAsset maps a relative asset reference to an absolute base-prefixed URL. It
// returns "" (meaning "leave unchanged") for empty, remote, data:, or already-absolute
// sources.
func rewriteAsset(src, base string) string {
	if src == "" ||
		strings.HasPrefix(src, "http://") ||
		strings.HasPrefix(src, "https://") ||
		strings.HasPrefix(src, "//") ||
		strings.HasPrefix(src, "data:") ||
		strings.HasPrefix(src, "/") {
		return ""
	}
	return base + "/" + strings.TrimPrefix(src, "./")
}

func newMDParser() goldmark.Markdown {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Linkify,
			extension.Footnote,
			extension.Typographer,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.DefinitionList,
			// Class-based output, not inline styles: the CSP sets
			// style-src 'self', which strips style attributes. Token colors
			// live in internal/ui/styles/tailwind.css (chroma "github" style).
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
			),
			&mermaid.Extender{},
			&frontmatter.Extender{},
		),
		goldmark.WithParserOptions(
			parser.WithAttribute(),
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(
				util.Prioritized(mediaResolver{}, 100),
			),
		),
		goldmark.WithRendererOptions(
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
