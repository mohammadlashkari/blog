package content

import (
	"crypto/md5"
	"time"
)

type Language string

const (
	LanguageEn Language = "en"
	LanguageFa Language = "fa"
)

func (l Language) Dir() string {
	if l == LanguageFa {
		return "rtl"
	}
	return "ltr"
}

func (l Language) Lang() string {
	return string(l)
}

type FrontMatter struct {
	Title       string     `yaml:"title"`
	Slug        string     `yaml:"slug"`
	Description string     `yaml:"description"`
	Language    Language   `yaml:"language"`
	IsFavorite  bool       `yaml:"is_favorite"`
	Tags        []string   `yaml:"tags"`
	CoverImage  string     `yaml:"cover_image"`
	EmbedID     string     `yaml:"embed"`
	PublishedAt *time.Time `yaml:"published_at"`
	Version     int64      `yaml:"version"`
}

type Post struct {
	FrontMatter
	HTML        string
	ContentHash [md5.Size]byte
}
