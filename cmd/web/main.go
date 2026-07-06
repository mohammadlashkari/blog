package main

import (
	"blog/internal/auth"
	"blog/internal/config"
	"blog/internal/content"
	"blog/internal/migration"
	"blog/internal/post"
	"blog/internal/rss"
	"blog/internal/rss/store"
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"blog/internal/ui"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	ctx := context.Background()

	cfg, err := config.New()
	if err != nil {
		log.Fatalln("failed to load config:", err)
	}

	setupLogger(cfg.Env)

	db, err := openDB(ctx, cfg.DBPath)
	if err != nil {
		log.Fatalln("failed to open db connection:", err)
	}
	defer db.Close()

	if err := migration.UP(cfg.DBPath); err != nil {
		log.Fatalln("failed to do migration up:", err)
	}

	rssSrv := rss.New(
		cfg,
		store.NewRSSStore(db),
	)

	cont := content.New(
		cfg.LocalContentPath,
		cfg.RemoteContentPath,
		cfg.MainBranchName,
		cfg.ContentFilename,
	)

	postSrv, err := post.New(ctx, cfg, cont)
	if err != nil {
		log.Fatalln("failed to boot post service:", err)
	}

	authSrv := auth.New(ctx, cfg)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.FileServer(http.FS(ui.StaticFS)))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/posts", http.StatusFound)
	})
	mux.HandleFunc("GET /about", handleAbout)
	mux.HandleFunc("GET /health", handleHealth)
	authSrv.RegisterRoutes(mux)
	postSrv.RegisterRoutes(mux)
	rssSrv.RegisterRoutes(mux)

	handler := chainMiddlewars(
		mux,
		recoverPanic,
		logger,
		isAdmin(authSrv),
		rateLimit(cfg),
	)

	srv := http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", cfg.Port),
		Handler: handler,
	}

	slog.InfoContext(ctx, "starting blog server on", "address", srv.Addr, "pid", os.Getpid())
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

func setupLogger(env string) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if env == "develop" {
		opts.Level = slog.LevelDebug
		opts.AddSource = true
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, opts))
	slog.SetDefault(logger)
}
