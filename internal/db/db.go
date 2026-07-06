package db

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func Open(ctx context.Context, dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	if _, err := db.ExecContext(ctx, `PRAGMA journal_mode = wal; PRAGMA foreign_keys = ON;`); err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	if err := migrateUP(db); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "db setup succeed")
	return db, nil
}
