package ui

import (
	"bytes"
	"embed"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
)

//go:embed  static
var StaticFS embed.FS

var PostEmbeds = map[string]templ.Component{
	"game-of-life": GameOfLifeEmbed(),
}

func Render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	ctx := r.Context()

	// Render into a buffer first. Rendering straight to w commits 200 OK plus a
	// partial body, so a mid-render failure can no longer be turned into a clean
	// 500 and clients (and caching proxies) get a truncated page instead.
	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		slog.ErrorContext(ctx, "failed to render page", "path", r.URL.Path, "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	if _, err := buf.WriteTo(w); err != nil {
		slog.ErrorContext(ctx, "failed to write page", "path", r.URL.Path, "error", err)
	}
}
