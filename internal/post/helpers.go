package post

import (
	"cmp"
	"slices"
)

type yearGroup struct {
	year  int
	posts []Post
}

// Posts are already sorted by PublishedAt in descending order
func groupPostsByLanguageAndYear(posts []Post) (en, fa []yearGroup) {
	enGroups := make(map[int][]Post)
	faGroups := make(map[int][]Post)

	for _, p := range posts {
		if p.PublishedAt == nil {
			continue
		}
		year := p.PublishedAt.Year()
		switch p.Language {
		case LanguageEn:
			enGroups[year] = append(enGroups[year], p)
		case LanguageFa:
			faGroups[year] = append(faGroups[year], p)
		}
	}

	return buildYearGroups(enGroups), buildYearGroups(faGroups)
}

func buildYearGroups(groups map[int][]Post) []yearGroup {
	yg := make([]yearGroup, 0, len(groups))
	for y, ps := range groups {
		yg = append(yg, yearGroup{year: y, posts: ps})
	}
	slices.SortFunc(yg, func(a, b yearGroup) int {
		return cmp.Compare(b.year, a.year)
	})
	return yg
}
