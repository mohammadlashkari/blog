package main

import (
	"blog/internal/config"
	"blog/internal/migration"
	"blog/internal/post"
	"database/sql"
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
	mux.Handle("/static/", http.FileServer(http.FS(ui.StaticFS)))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/posts", http.StatusFound)
	})
	mux.HandleFunc("GET /about", handleAbout)
	mux.HandleFunc("GET /health", handleHealth)
	postSvc.RegisterRoutes(mux)

	handler := chainMiddlewars(
		mux,
		recoverPanic,
		logger,
		rateLimit(cfg),
	)

	srv := http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", cfg.Port),
		Handler: handler,
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
