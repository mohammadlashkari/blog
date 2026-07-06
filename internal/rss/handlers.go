package rss

import (
	"blog/internal/rss/store"
	"blog/internal/ui"
	"fmt"
	"log/slog"
	"net/http"
)

func (s *Service) handleReadingPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	items, err := s.store.GetItemsByStatus(ctx, store.GetItemsByStatusParams{
		Status: "archived",
		Limit:  10,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to get rss items", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, l := range items {
		fmt.Println("---->", l.FeedName, l.FeedUrl, l.Link)
	}

	ui.Render(w, r, ReadingPage(items))
}

func (s *Service) handleRefreshReading(w http.ResponseWriter, r *http.Request) {
	if err := s.FetchAll(r.Context()); err != nil {
		http.Error(w, "failed to refresh", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
