package rss

import (
	"blog/internal/ui"
	"log/slog"
	"net/http"
)

func (s *Service) handleReadingPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	items, err := s.store.GetItemsByStatus(ctx, "archived")
	if err != nil {
		slog.ErrorContext(ctx, "failed to get rss items", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ui.Render(w, r, ReadingPage(items))
}

func (s *Service) handleRefreshReading(w http.ResponseWriter, r *http.Request) {
	if err := s.Refresh(r.Context()); err != nil {
		http.Error(w, "failed to refresh reading list", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
