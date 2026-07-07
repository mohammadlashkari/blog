package rss

import (
	"html"
	"regexp"
	"strings"
)

var htmlTagRE = regexp.MustCompile(`<[^>]+>`)

func cleanDescription(s string) string {
	s = html.UnescapeString(s)
	s = htmlTagRE.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), " ")
	return s
}
