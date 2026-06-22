package main

import (
	"blog/internal/config"
	"blog/internal/migration"
	"blog/internal/post"
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"blog/internal/ui"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/frontmatter"
)

func main() {
	ctx := context.Background()

	cfg, err := config.New()
	if err != nil {
		log.Fatalln("failed to load config:", err)
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if _, err := os.ReadDir(cfg.LocalContentRepo); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				slog.ErrorContext(ctx, "failed to read content repo dir", "error", err)
				return
			}

			cmd := exec.CommandContext(ctx, "git", "clone", cfg.RemoteContentRepo, cfg.LocalContentRepo)
			if output, err := cmd.CombinedOutput(); err != nil {
				slog.ErrorContext(ctx, "git clone failed", "error", err, "output", string(output))
				return
			}
			slog.InfoContext(ctx, "git clone succeeded")
			return
		}

		cmd := exec.CommandContext(ctx, "git", "pull", "origin", "master")
		cmd.Dir = cfg.LocalContentRepo
		if output, err := cmd.CombinedOutput(); err != nil {
			slog.ErrorContext(ctx, "git pull failed", "error", err, "output", string(output))
			return
		}
		slog.InfoContext(ctx, "git pull succeeded")
	}()

	db, err := openDB(ctx, cfg.DbPath)
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
	mux.HandleFunc("GET /admin/login", handleAdminLogin)
	mux.HandleFunc("GET /health", handleHealth)
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

func openDB(ctx context.Context, dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
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
