package main

import (
	"blog/internal/auth"
	"blog/internal/config"
	"blog/internal/content"
	"blog/internal/db"
	"blog/internal/post"
	"blog/internal/rss"
	"blog/internal/rss/store"
	"context"
	"expvar"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"blog/internal/ui"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Dev()
	if err != nil {
		log.Fatalln("failed to load config:", err)
	}

	setupLogger(cfg.Env)

	db, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		log.Fatalln("failed to open db connection:", err)
	}
	defer db.Close()

	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))
	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))
	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	rssSvc := rss.New(
		cfg,
		store.NewRSSStore(db),
	)

	cont := content.New(
		cfg.LocalContentPath,
		cfg.RemoteContentPath,
		cfg.MainBranchName,
		cfg.ContentFilename,
	)

	postSvc, err := post.New(ctx, cfg, cont, rssSvc)
	if err != nil {
		log.Fatalln("failed to boot post service:", err)
	}

	authSvc := auth.New(ctx, cfg)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.FileServer(http.FS(ui.StaticFS)))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/posts", http.StatusFound)
	})
	mux.HandleFunc("GET /about", handleAbout)
	mux.HandleFunc("GET /health", handleHealth)
	mux.Handle("GET /debug/vars", expvar.Handler())
	authSvc.RegisterRoutes(mux)
	postSvc.RegisterRoutes(mux)
	rssSvc.RegisterRoutes(mux)

	handler := chainMiddlewars(
		mux,
		recoverPanic,
		logger,
		isAdmin(authSvc),
		rateLimit(cfg),
		secureHeaders,
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
