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

	postSvc := post.NewService(cfg, db)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./cmd/web/static"))))
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
