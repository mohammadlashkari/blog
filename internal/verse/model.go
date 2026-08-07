package verse

import "blog/internal/content"

type Verse struct {
	Text     string           `yaml:"text"`
	Author   string           `yaml:"author"`
	Language content.Language `yaml:"language"`
}

type versesFile struct {
	Verses []Verse `yaml:"verses"`
}
