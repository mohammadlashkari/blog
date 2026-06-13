package post

import (
	"blog/internal/config"
	"database/sql"
)

type Language string

const (
	LanguageEn Language = "en"
	LanguageFa Language = "fa"
)

type Service struct {
	cfg   *config.Config
	store Querier
}

func NewService(cfg *config.Config, db *sql.DB) *Service {
	return &Service{
		cfg:   cfg,
		store: New(db),
	}
}
