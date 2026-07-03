package post

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func (s *Service) handlePostsIndex(w http.ResponseWriter, r *http.Request) {
	var (
		tags []string
		en   []yearGroup
		fa   []yearGroup
	)

	tags = r.URL.Query()["tag"]

	if len(tags) > 0 {
		posts := s.GetPostsWithTags(tags)
		en, fa = groupByYearAndLang(posts)

	} else {
		posts := s.GetPosts()
		en, fa = groupByYearAndLang(posts)
		tags = s.GetTags()
	}

	render(w, r, PostsIndexPage(en, fa, tags))
}

func (s *Service) handlePost(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	post, ok := s.GetPostBySlug(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	uiPost := buildUIPost(post, "02 January 2006", "%d %B %Y")

	render(w, r, PostPage(uiPost, post.Tags, post.HTML, components[post.Embed]))
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

func (s *Service) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		idx, err := s.content.Build(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "failed to rebuild the blog content", "error", err)
			return
		}
		s.index.Store(idx)
	}()
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
