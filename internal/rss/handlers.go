package rss

import (
	"blog/internal/rss/store"
	"blog/internal/ui"
	"fmt"
	"log/slog"
	"net/http"
)

func (rs *RSSService) handleReadingListPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	items, err := rs.store.GetItemsByStatus(ctx, store.GetItemsByStatusParams{
		Status: "archived",
		Limit:  10,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to get rss items", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	for _, l := range items {
		fmt.Println("----", l)
	}

	ui.Render(w, r, ReadingListPage())
}
