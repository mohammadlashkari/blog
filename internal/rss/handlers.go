package rss

import (
	"blog/internal/auth"
	"blog/internal/rss/store"
	"blog/internal/ui"
	"log/slog"
	"net/http"
	"strconv"
)

func (s *Service) handleReadingPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	isAdmin := auth.IsAdmin(ctx)

	// Visitors only ever see the curated (saved) list.
	activeStatus := "saved"
	if isAdmin {
		activeStatus = normalizeStatus(r.URL.Query().Get("status"))
	}

	var (
		items []store.RssItem
		err   error
	)
	if activeStatus == "saved" {
		items, err = s.store.GetSavedItems(ctx)
	} else {
		items, err = s.store.GetItemsByStatus(ctx, activeStatus)
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to get rss items", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, ReadingPage(items, isAdmin, activeStatus))
}

// normalizeStatus clamps a status query param to a known tab, defaulting to unread.
func normalizeStatus(s string) string {
	switch s {
	case read, archived, "saved":
		return s
	default:
		return unread
	}
}

func (s *Service) handleSetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	status := r.FormValue("status")
	if status != unread && status != read && status != archived {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateItemStatus(ctx, store.UpdateItemStatusParams{Status: status, ID: id}); err != nil {
		slog.ErrorContext(ctx, "failed to update item status", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	redirectToTab(w, r)
}

func (s *Service) handleToggleSaved(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := s.store.ToggleSavedStatus(ctx, id); err != nil {
		slog.ErrorContext(ctx, "failed to toggle saved", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	redirectToTab(w, r)
}

// redirectToTab sends the owner back to the reading tab they acted from.
func redirectToTab(w http.ResponseWriter, r *http.Request) {
	tab := normalizeStatus(r.FormValue("tab"))
	http.Redirect(w, r, "/reading?status="+tab, http.StatusSeeOther)
}

func (s *Service) handleRefreshReading(w http.ResponseWriter, r *http.Request) {
	if err := s.Refresh(r.Context()); err != nil {
		http.Error(w, "failed to refresh reading list", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
