package main

import (
	"blog/internal/config"
	"blog/internal/migration"
	"blog/internal/post"
	"database/sql"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"blog/internal/ui"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/frontmatter"
)

func main() {
	cfg, err := config.Web()
	if err != nil {
		log.Fatalln("failed to load config:", err)
	}

	db, err := openDB(cfg.DbPath)
	if err != nil {
		log.Fatalln("failed to open db:", err)
	}
	defer db.Close()

	if err := migration.UP(cfg.DbPath); err != nil {
		log.Fatalln("failed to do migration up:", err)
	}

	md := setupMDParser()

	postSvc := post.NewService(cfg, db, md)

	mux := http.NewServeMux()

	staticFS, err := fs.Sub(ui.StaticFS, "static")
	if err != nil {
		log.Println(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/posts", http.StatusFound)
	})
	postSvc.RegisterRoutes(mux)

	srv := http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", cfg.Port),
		Handler: mux,
	}

	slog.Info("starting blog server on", "address", srv.Addr, "pid", os.Getpid())
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalln("listen and serve:", err)
	}
}

func openDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func setupMDParser() goldmark.Markdown {
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
