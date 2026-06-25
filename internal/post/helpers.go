package post

import (
	"cmp"
	"log/slog"
	"net/http"
	"slices"

	"github.com/a-h/templ"
	"github.com/mshafiee/jalali"
)

func render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "failed to render page", "path", r.URL.Path, "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

type PostUI struct {
	Post
	DisplayDate string
}

type yearGroup struct {
	isUnpublished bool
	year          int
	posts         []PostUI
}

// Posts are already sorted by PublishedAt in descending order
func groupPostsByLanguageAndYear(posts []Post) (en, fa []yearGroup) {
	enGroups := make(map[int][]PostUI)
	faGroups := make(map[int][]PostUI)

	var enUnpublished []PostUI
	var faUnpublished []PostUI

	for _, p := range posts {

		uiPost := PostUI{Post: p}

		switch p.Language {
		case LanguageEn:

			if p.PublishedAt == nil {
				uiPost.DisplayDate = "Draft"
				enUnpublished = append(enUnpublished, uiPost)
			} else {
				uiPost.DisplayDate = p.PublishedAt.Format("02 Jan")
				year := p.PublishedAt.Year()
				enGroups[year] = append(enGroups[year], uiPost)
			}

		case LanguageFa:

			if p.PublishedAt == nil {
				uiPost.DisplayDate = "پیش‌نویس"
				faUnpublished = append(faUnpublished, uiPost)
			} else {
				jt := jalali.ToJalali(*p.PublishedAt)
				uiPost.DisplayDate = jt.Format("%d %B")
				year := jt.Year()
				faGroups[year] = append(faGroups[year], uiPost)
			}
		}
	}

	// Build standard groups
	en = buildYearGroups(enGroups)
	fa = buildYearGroups(faGroups)

	if len(enUnpublished) > 0 {
		en = append([]yearGroup{{isUnpublished: true, posts: enUnpublished}}, en...)
	}
	if len(faUnpublished) > 0 {
		fa = append([]yearGroup{{isUnpublished: true, posts: faUnpublished}}, fa...)
	}

	return en, fa
}

func buildYearGroups(groups map[int][]PostUI) []yearGroup {
	yg := make([]yearGroup, 0, len(groups))
	for y, ps := range groups {
		yg = append(yg, yearGroup{year: y, posts: ps})
	}

	slices.SortFunc(yg, func(a, b yearGroup) int {
		return cmp.Compare(b.year, a.year)
	})

	return yg
}

func dirOf(l Language) string {
	if l == LanguageFa {
		return "rtl"
	}
	return "ltr"
}

func langOf(l Language) string {
	return string(l)
}
