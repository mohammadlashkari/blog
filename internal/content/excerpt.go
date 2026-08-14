package content

import (
	"html"
	"regexp"
	"strings"
)

const SummaryLength = 150

var htmlTagRE = regexp.MustCompile(`<[^>]+>`)

// excerpt flattens rendered post HTML into a plain-text opening, for feed items
// whose front matter has no description.
func excerpt(postHTML string, maxRunes int) string {
	text := htmlTagRE.ReplaceAllString(postHTML, " ")
	text = html.UnescapeString(text)
	text = strings.Join(strings.Fields(text), " ")

	cut := firstNRunes(text, maxRunes)
	if len(cut) == len(text) { // cut is a prefix, so equal length means nothing was dropped
		return text
	}

	if i := strings.LastIndex(cut, " "); i > 0 {
		cut = cut[:i]
	}

	return cut + "…"
}

func firstNRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}

	count := 0
	for idx := range s {
		if count == n {
			return s[:idx]
		}
		count++
	}

	return s
}
