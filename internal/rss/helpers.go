package rss

import (
	"html"
	"regexp"
	"slices"
	"strings"
)

var htmlTagRE = regexp.MustCompile(`<[^>]+>`)

func cleanDescription(s string) string {
	s = html.UnescapeString(s)
	s = htmlTagRE.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func normalizeStatus(s string) string {
	switch s {
	case read, archived, saved:
		return s
	default:
		return unread
	}
}

func isStatusValid(s string) bool {
	return slices.Contains([]string{unread, read, saved, archived}, s)
}
