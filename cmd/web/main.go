package main

import (
	"blog/internal/config"
	"blog/internal/content"
	"blog/internal/post"
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"blog/internal/ui"
)

func main() {
	ctx := context.Background()

	cfg, err := config.New()
	if err != nil {
		log.Fatalln("failed to load config:", err)
	}

	setupLogger(cfg.Env)

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

	mux := http.NewServeMux()
	mux.Handle("/static/", http.FileServer(http.FS(ui.StaticFS)))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/posts", http.StatusFound)
	})
	mux.HandleFunc("GET /about", handleAbout)
	mux.HandleFunc("GET /health", handleHealth)
	postSrv.RegisterRoutes(mux)

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
