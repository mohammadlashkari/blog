package ui

import (
	"embed"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
)

//go:embed  static
var StaticFS embed.FS

var PostEmbeds = map[string]templ.Component{
	"game-of-life": GameOfLifeEmbed(),
}

func Render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "failed to render page", "path", r.URL.Path, "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
