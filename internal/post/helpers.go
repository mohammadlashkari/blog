package post

import (
	"blog/internal/content"
	"blog/internal/ui"
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"sort"
	"strconv"

	"github.com/a-h/templ"
	"github.com/mshafiee/jalali"
)

var components = map[string]templ.Component{
	"game-of-life": ui.GameOfLifeEmbed(),
}

func render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "failed to render page", "path", r.URL.Path, "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

type contextKey string

const IsAdminKey contextKey = "IsAdmin"

func isAdmin(ctx context.Context) bool {
	isAdmin, ok := ctx.Value(IsAdminKey).(bool)
	return ok && isAdmin
}

type uiPost struct {
	*content.Post
	displayDate string
}

type yearGroup struct {
	year  string
	posts []uiPost
}

func groupByYearAndLang(posts []*content.Post) (en, fa []yearGroup) {
	enYears := make(map[int][]uiPost)
	faYears := make(map[int][]uiPost)
	var enDrafts, faDrafts []uiPost

	for _, post := range posts {
		u := buildUIPost(post, "02 Jan", "%d %B")

		switch post.Language {
		case content.LanguageEn:
			if post.PublishedAt != nil {
				enYears[post.PublishedAt.Year()] = append(enYears[post.PublishedAt.Year()], u)
			} else {
				enDrafts = append(enDrafts, u)
			}
		case content.LanguageFa:
			if post.PublishedAt != nil {
				jt := jalali.ToJalali(*post.PublishedAt)
				faYears[jt.Year()] = append(faYears[jt.Year()], u)
			} else {
				faDrafts = append(faDrafts, u)
			}
		}
	}

	en = buildYearGroups(enYears, enDrafts, "Unpublished")
	fa = buildYearGroups(faYears, faDrafts, "منتشر نشده")

	return
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
	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	for _, y := range years {
		groups = append(groups, yearGroup{year: strconv.Itoa(y), posts: m[y]})
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
