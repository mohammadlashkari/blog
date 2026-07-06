package store

import (
	"context"
	"database/sql"
	"fmt"
)

type Storer interface {
	Querier
	ExecTx(ctx context.Context, fn func(Querier) error) error
}

type RSSStore struct {
	db *sql.DB
	*Queries
}

func NewRSSStore(db *sql.DB) *RSSStore {
	return &RSSStore{
		db:      db,
		Queries: New(db),
	}
}

// ExecTx executes a function within a database transaction.
func (s *RSSStore) ExecTx(ctx context.Context, fn func(Querier) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)

	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
