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

// FrontMatter is a post's YAML header. omitempty marks the genuinely-optional keys so
// scaffolding (blog post add) can marshal this struct straight to a clean index.md; it has
// no effect on decoding, the only path the server uses. Field order here is the canonical
// key order every scaffolded post is written in.
type FrontMatter struct {
	Title       string     `yaml:"title"`
	Slug        string     `yaml:"slug"`
	CoverImage  string     `yaml:"cover_image,omitempty"`
	Description string     `yaml:"description"`
	Language    Language   `yaml:"language"`
	IsFavorite  bool       `yaml:"is_favorite"`
	Tags        []string   `yaml:"tags"`
	PublishedAt *time.Time `yaml:"published_at,omitempty"`
	Version     int64      `yaml:"version,omitempty"`
	EmbedID     string     `yaml:"embed,omitempty"`
}

type Post struct {
	FrontMatter
	HTML        string
	ContentHash [md5.Size]byte
}
