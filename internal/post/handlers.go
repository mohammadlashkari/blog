package post

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func (s *Service) handlePostsIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	posts, err := s.store.ListPosts(ctx, false)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list all posts", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := PostsIndexPage(posts).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "failed to render post index page", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Println("posts", posts)
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
	if !errors.Is(err, sql.ErrNoRows) {
		slog.ErrorContext(ctx, "failed to get post's tags", "slug", slug, "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	_ = tags

	postPath := filepath.Join(s.cfg.LocalContentRepo, post.Filename)

	contentMD, err := os.ReadFile(postPath)
	if err != nil {
		slog.ErrorContext(
			ctx, "failed to read post's markdown file",
			"slug", slug,
			"location", postPath,
			"error", err,
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	_ = contentMD

}

func handleAbout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("about\n"))
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok\n"))
}
