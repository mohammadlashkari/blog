package post

import (
	"blog/internal/ui"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/yuin/goldmark/parser"
)

func (s *Service) handlePostsIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var (
		posts []Post
		err   error
	)

	tagsFilter := r.URL.Query()["tag"]

	if len(tagsFilter) > 0 {
		posts, err = s.store.ListPostsByTag(ctx, ListPostsByTagParams{
			TagNames:   tagsFilter,
			IncludeAll: true,
		})
	} else {
		posts, err = s.store.ListPosts(ctx, true)
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to list posts", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tags, err := s.store.ListTags(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list tags", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	en, fa := groupPostsByLanguageAndYear(posts)
	render(w, r, PostsIndexPage(en, fa, tags))
}

func (s *Service) handlePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slug := r.PathValue("slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	post, err := s.store.PostBySlug(ctx, PostBySlugParams{
		Slug:       slug,
		IncludeAll: true,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		slog.ErrorContext(ctx, "failed to get post", "slug", slug, "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tags, err := s.store.TagsByPostID(ctx, post.ID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.ErrorContext(ctx, "failed to get post's tags", "slug", slug, "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	postPath := filepath.Join(s.cfg.LocalContentRepo, post.Filename)
	contentMD, err := os.ReadFile(postPath)
	if err != nil {
		slog.ErrorContext(
			ctx, "failed to read post's markdown file",
			"slug", slug,
			"path", postPath,
			"error", err,
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	parserCtx := parser.NewContext()

	var buf bytes.Buffer
	if err := s.md.Convert(contentMD, &buf, parser.WithContext(parserCtx)); err != nil {
		slog.ErrorContext(ctx, "failed to convert markdown to html", "slug", slug, "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	contentHTML := buf.String()

	var component templ.Component
	if post.Slug == "game-of-life" {
		component = ui.GameOfLifeEmbed()
	}

	render(w, r, PostPage(post, tags, contentHTML, component))
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

	if payload.Ref != "refs/heads/master" {
		w.WriteHeader(http.StatusOK)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(ctx, "git", "pull", "origin", "master")
		cmd.Dir = s.cfg.LocalContentRepo

		output, err := cmd.CombinedOutput()
		if err != nil {
			slog.ErrorContext(
				ctx,
				"git pull failed",
				"path", cmd.Dir, "error", err, "output", string(output),
			)
			return
		}

		slog.InfoContext(ctx, "git pull succeeded")
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
