package post

import (
	"blog/internal/config"
	"database/sql"

	"github.com/yuin/goldmark"
)

type Language string

const (
	LanguageEn Language = "en"
	LanguageFa Language = "fa"
)

type Service struct {
	cfg   *config.Config
	db    *sql.DB
	store Querier
	md    goldmark.Markdown
}

func NewService(cfg *config.Config, db *sql.DB, md goldmark.Markdown) *Service {
	return &Service{
		cfg:   cfg,
		db:    db,
		store: New(db),
		md:    md,
	}
}
