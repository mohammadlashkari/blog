package content

import (
	"blog/internal/ui"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ValidatePost validates a post's front matter against the blog's validation rules.
func ValidatePost(p *Post) error {
	if p == nil {
		return errors.New("post is nil")
	}

	var errs []error
	if strings.TrimSpace(p.Title) == "" {
		errs = append(errs, errors.New("title is required"))
	}
	if !slugRegex.MatchString(p.Slug) {
		errs = append(errs, fmt.Errorf("slug %q must be lowercase kebab-case matching %s", p.Slug, slugRegex))
	}
	switch p.Language {
	case LanguageEn, LanguageFa:
	default:
		errs = append(errs, fmt.Errorf("language %q must be one of %q, %q", p.Language, LanguageEn, LanguageFa))
	}
	if embed := strings.TrimSpace(p.EmbedID); embed != "" {
		if _, ok := ui.PostEmbeds[embed]; !ok {
			errs = append(errs, fmt.Errorf("invalid %q embed", embed))
		}
	}

	return errors.Join(errs...)
}

func (c *Content) ValidatePosts(root string) error {
	_, err := c.fsPosts(root)
	return err
}
