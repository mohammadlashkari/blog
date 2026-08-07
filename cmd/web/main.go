package main

import (
	"blog/internal/auth"
	"blog/internal/config"
	"blog/internal/content"
	"blog/internal/db"
	"blog/internal/post"
	"blog/internal/rss"
	"blog/internal/rss/store"
	"blog/internal/verse"
	"context"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"blog/internal/ui"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server terminated", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	setupLogger(cfg.Env)

	db, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open db connection: %w", err)
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

	go rssSvc.StartDailyRefresh(ctx)

	cont := content.New(
		cfg.LocalContentPath,
		cfg.RemoteContentPath,
		cfg.MainBranchName,
		cfg.ContentFilename,
	)

	verseSvc := verse.New(cfg)

	postSvc, err := post.New(ctx, cfg, cont, rssSvc, verseSvc)
	if err != nil {
		return fmt.Errorf("failed to boot post service: %w", err)
	}

	// Must run after post.New: that's what puts the content repo on disk.
	// An empty verses page is degraded, not fatal, so don't abort the boot.
	if err := verseSvc.Load(ctx); err != nil {
		slog.WarnContext(ctx, "failed to load verses", "error", err)
	}

	authSvc := auth.New(ctx, cfg)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.FileServer(http.FS(ui.StaticFS)))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/posts", http.StatusFound)
	})
	mux.HandleFunc("GET /about", handleAbout)
	mux.HandleFunc("GET /health", handleHealth)
	mux.Handle("GET /debug/vars", auth.RequireAdmin(expvar.Handler()))
	authSvc.RegisterRoutes(mux)
	postSvc.RegisterRoutes(mux)
	rssSvc.RegisterRoutes(mux)
	verseSvc.RegisterRoutes(mux)

	handler := chainMiddlewars(
		mux,
		recoverPanic,
		logger,
		secureHeaders,
		rateLimit(cfg),
		isAdmin(authSvc),
	)

	srv := &http.Server{
		Addr:              net.JoinHostPort("0.0.0.0", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	srvErr := make(chan error, 1)
	go func() {
		slog.InfoContext(ctx, "starting blog server on", "address", srv.Addr, "pid", os.Getpid())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
		}
	}()

	select {
	case err := <-srvErr:
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
		stop()
	}

	slog.Info("shutting down, draining in-flight requests")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	slog.Info("server stopped")
	return nil
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
