package post

import (
	"blog/internal/auth"
	"blog/internal/content"
	"blog/internal/rss"
	"blog/internal/ui"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func (s *PostService) handlePostsIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tags := r.URL.Query()["tag"]
	if len(tags) == 0 {
		tags = s.GetTags()
	}

	posts := s.GetPosts(
		withIncludeUnpublished(auth.IsAdmin(ctx)),
		withTags(tags...),
	)

	en, fa := groupByYearAndLang(posts)

	ui.Render(w, r, PostsIndexPage(en, fa, tags))
}

func (s *PostService) handlePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slug := r.PathValue("slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	post, ok := s.GetPostBySlug(
		slug,
		withIncludeUnpublished(auth.IsAdmin(ctx)),
	)
	if !ok {
		http.NotFound(w, r)
		return
	}
	uiPost := buildUIPost(post, "02 January 2006", "%d %B %Y")

	ui.Render(w, r, PostPage(uiPost, post.Tags, post.HTML, ui.PostEmbeds[post.EmbedID]))
}

type GitHubPushPayload struct {
	Ref        string     `json:"ref"`
	Repository Repository `json:"repository"`
}

type Repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	CloneURL string `json:"clone_url"`
}

const maxWebhookBodyBytes = 5 << 20 // 5MB

func (s *PostService) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read webhook body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if !verifyGitHubSignature(body, r.Header.Get("X-Hub-Signature-256"), s.cfg.WebhookSecret) {
		slog.ErrorContext(ctx, "invalid webhook signature")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload GitHubPushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.ErrorContext(ctx, "failed to parse webhook payload", "error", err)
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	if payload.Ref != fmt.Sprintf("refs/heads/%s", s.cfg.MainBranchName) {
		w.WriteHeader(http.StatusOK)
		return
	}

	if !s.refreshing.CompareAndSwap(false, true) {
		slog.InfoContext(ctx, "content refresh already in progress, skipping")
		w.WriteHeader(http.StatusAccepted)
		return
	}

	go func() {
		defer s.refreshing.Store(false)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		idx, err := s.content.Refresh(ctx)
		if err != nil || idx == nil { // Content unchanged, keep serving the current index
			return
		}
		s.index.Store(idx)

		if err := s.rssSvc.Sync(ctx); err != nil {
			slog.ErrorContext(ctx, "rss synchronization failed", "error", err)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

func (s *PostService) handleFeed(lang *content.Language) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		opts := []QueryOption{}

		langStr := ""
		if lang != nil {
			langStr = string(*lang)
			opts = append(opts, withLanguage(*lang))
		}
		posts := s.GetPosts(opts...)

		feed, err := buildFeed(langStr, s.cfg.SiteURL, posts)
		if err != nil {
			slog.ErrorContext(r.Context(), "failed to build feed", "erro", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(feed)
	}
}

func buildFeed(lang string, siteURL string, posts []*content.Post) ([]byte, error) {
	ch := rss.Channel{
		Title:       "Mohammad's blog",
		Link:        siteURL,
		Description: "Thoughts, notes, and experiments",
		Copyright:   fmt.Sprintf("Copyright © 2025–%d Mohammad", time.Now().Year()),
		Language:    lang,
		Docs:        "https://www.rssboard.org/rss-specification",
	}

	if len(posts) > 0 && posts[0].PublishedAt != nil {
		ch.LastBuildDate = posts[0].PublishedAt.Format(time.RFC1123)
	}

	for _, p := range posts {
		link := siteURL + "/post/" + p.Slug
		item := rss.Item{
			GUID:        link,
			Title:       p.Title,
			Link:        link,
			Author:      "Mohammad Lashkari",
			Description: p.Description,
			Categories:  p.Tags,
			PubDate:     p.PublishedAt.Format(time.RFC1123),
		}
		ch.Items = append(ch.Items, item)
	}

	feed := rss.Feed{Version: "2.0", Channel: ch}

	body, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), body...), nil
}

func verifyGitHubSignature(payload []byte, signature, secret string) bool {
	if signature == "" || secret == "" {
		return false
	}

	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}
