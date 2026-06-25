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

type uiPost struct {
	Post
	DisplayDate string
}

type yearGroup struct {
	isUnpublished bool
	year          int
	posts         []uiPost
}

// Posts are already sorted by PublishedAt in descending order
func groupPostsByLanguageAndYear(posts []Post) (en, fa []yearGroup) {
	enGroups := make(map[int][]uiPost)
	faGroups := make(map[int][]uiPost)

	var enUnpublished []uiPost
	var faUnpublished []uiPost

	for _, p := range posts {

		uiPost := uiPost{Post: p}

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

func buildYearGroups(groups map[int][]uiPost) []yearGroup {
	yg := make([]yearGroup, 0, len(groups))
	for y, ps := range groups {
		yg = append(yg, yearGroup{year: y, posts: ps})
	}

	slices.SortFunc(yg, func(a, b yearGroup) int {
		return cmp.Compare(b.year, a.year)
	})

	return yg
}

func buildUIPost(p Post) uiPost {
	uiPost := uiPost{Post: p}

	switch p.Language {
	case LanguageEn:
		if p.PublishedAt == nil {
			uiPost.DisplayDate = "Draft"
		} else {
			uiPost.DisplayDate = p.PublishedAt.Format("02 January 2006")
		}

	case LanguageFa:
		if p.PublishedAt == nil {
			uiPost.DisplayDate = "پیش‌نویس"
		} else {
			jt := jalali.ToJalali(*p.PublishedAt)
			uiPost.DisplayDate = jt.Format("%d %B %y")
		}
	}

	return uiPost
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
