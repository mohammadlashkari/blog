package main

import (
	"blog/internal/post"
	"blog/internal/store"
	"log"
	"net"
	"net/http"
)

type config struct {
	port   string
	dbPath string
}

func Load() (*config, error) {
	return &config{
		port:   "2026",
		dbPath: "./blog.sqlite",
	}, nil
}

func main() {
	cfg, err := Load()
	if err != nil {
		log.Fatalln("failed to load config:", err)
	}

	db, err := store.OpenDB(cfg.dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err := store.MigrationUP(cfg.dbPath); err != nil {
		panic(err)
	}

	store := store.NewStore(db)

	_ = store

	mux := http.NewServeMux()

	post.RegisterRoutes(mux)

	srv := http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", cfg.port),
		Handler: mux,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalln("listen and serve:", err)
	}
}
