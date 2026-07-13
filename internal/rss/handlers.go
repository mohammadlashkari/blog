package rss

import (
	"blog/internal/auth"
	"blog/internal/rss/store"
	"blog/internal/ui"
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

func (s *Service) handleReadingPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	isAdmin := auth.IsAdmin(ctx)

	activeStatus := normalizeStatus(r.URL.Query().Get("status"))

	items, err := s.store.GetItemsByStatus(ctx, activeStatus)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get rss items", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, ReadingPage(items, isAdmin, activeStatus))
}

func (s *Service) handleSetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	status := r.FormValue("status")
	if !isStatusValid(status) {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateItemStatus(ctx, store.UpdateItemStatusParams{Status: status, ID: id}); err != nil {
		slog.ErrorContext(ctx, "failed to update item status", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tab := normalizeStatus(r.FormValue("tab"))
	http.Redirect(w, r, "/reading?status="+tab, http.StatusSeeOther)
}

func (s *Service) handleRefreshReading(w http.ResponseWriter, r *http.Request) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := s.Refresh(ctx); err != nil {
			slog.ErrorContext(ctx, "failed to refresh reading list", "error", err)
		}
	}()

	http.Redirect(w, r, "/reading", http.StatusSeeOther)
}
