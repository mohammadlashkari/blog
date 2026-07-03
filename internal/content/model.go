package content

import (
	"crypto/sha256"
	"time"
)

type Language string

const (
	LanguageEn Language = "en"
	LanguageFa Language = "fa"
)

type FrontMatter struct {
	Title       string     `yaml:"title"`
	Slug        string     `yaml:"slug"`
	CoverImage  string     `yaml:"cover_image"`
	Language    Language   `yaml:"language"`
	IsFavorite  bool       `yaml:"is_favorite"`
	Tags        []string   `yaml:"tags"`
	PublishedAt *time.Time `yaml:"published_at"`
	Version     int64      `yaml:"version"`
	Embed       string     `yaml:"embed"`
}

type Post struct {
	FrontMatter
	HTML        string
	ContentHash [sha256.Size]byte
}
