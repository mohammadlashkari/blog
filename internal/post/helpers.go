package post

import (
	"blog/internal/content"
	"bytes"
	"sort"
	"strconv"

	"github.com/mshafiee/jalali"
)

type uiPost struct {
	*content.Post
	displayDate string
}

type yearGroup struct {
	year  string
	posts []uiPost
}

func groupByYearAndLang(posts []*content.Post) (en, fa []yearGroup) {
	var enPosts, faPosts []*content.Post

	for _, p := range posts {
		switch p.Language {
		case content.LanguageEn:
			enPosts = append(enPosts, p)
		case content.LanguageFa:
			faPosts = append(faPosts, p)
		}
	}

	en = groupPosts(enPosts, "Unpublished", false)
	fa = groupPosts(faPosts, "منتشر نشده", true)

	return
}

func groupPosts(posts []*content.Post, draftLabel string, isFarsi bool) []yearGroup {
	years := make(map[int][]uiPost)
	var drafts []uiPost

	for _, p := range posts {
		u := buildUIPost(p, "02 Jan", "%d %B")

		if p.PublishedAt == nil {
			drafts = append(drafts, u)
			continue
		}

		var year int
		if isFarsi {
			year = jalali.ToJalali(*p.PublishedAt).Year()
		} else {
			year = p.PublishedAt.Year()
		}

		years[year] = append(years[year], u)
	}

	return buildYearGroups(years, drafts, draftLabel)
}

// buildYearGroups sorts numeric years descending and prepends the draft
// bucket (if non-empty) so drafts always come first.
func buildYearGroups(m map[int][]uiPost, drafts []uiPost, draftLabel string) []yearGroup {
	groups := make([]yearGroup, 0, len(m)+1)
	if len(drafts) > 0 {
		groups = append(groups, yearGroup{year: draftLabel, posts: drafts})
	}

	years := make([]int, 0, len(m))
	for y := range m {
		years = append(years, y)
	}
	sort.Slice(years, func(i, j int) bool {
		return years[i] > years[j]
	})

	for _, y := range years {
		groups = append(
			groups,
			yearGroup{year: strconv.Itoa(y), posts: m[y]},
		)
	}

	return groups
}

func buildUIPost(p *content.Post, enLayout, faLayout string) uiPost {
	u := uiPost{Post: p}

	switch p.Language {
	case content.LanguageEn:
		if p.PublishedAt == nil {
			u.displayDate = "Draft"
		} else {
			u.displayDate = p.PublishedAt.Format(enLayout)
		}
	case content.LanguageFa:
		if p.PublishedAt == nil {
			u.displayDate = "پیش‌نویس"
		} else {
			jt := jalali.ToJalali(*p.PublishedAt)
			u.displayDate = farsiNum(jt.Format(faLayout))
		}
	}

	return u
}

func farsiNum(s string) string {
	var buf bytes.Buffer

	for _, r := range s {
		if r >= '0' && r <= '9' {
			r = '۰' + (r - '0')
		}
		buf.WriteRune(r)
	}

	return buf.String()
}

func dirOf(l content.Language) string {
	if l == content.LanguageFa {
		return "rtl"
	}
	return "ltr"
}

func langOf(l content.Language) string {
	return string(l)
}
