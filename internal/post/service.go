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
	cfg *config.Config
	db  *sql.DB
	q   Querier
	md  goldmark.Markdown
}

func NewService(cfg *config.Config, db *sql.DB, md goldmark.Markdown) *Service {
	return &Service{
		cfg: cfg,
		db:  db,
		q:   New(db),
		md:  md,
	}
}
