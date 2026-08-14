package rss

import (
	"blog/internal/auth"
	"blog/internal/rss/store"
	"blog/internal/ui"
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const readingPageSize = 50

func (s *Service) handleReadingPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := r.URL.Query()
	activeStatus := normalizeStatus(query.Get("status"))
	grouped := isGroupedByFeed(query.Get("group"))
	// a missing or malformed page is 0, which paginate clamps to 1
	requestedPage, _ := strconv.Atoi(query.Get("page"))

	items, err := s.store.GetItemsByStatus(ctx, activeStatus)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get rss items", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Count over the full bucket so the index reflects the whole status, then
	// page, then group — grouping the page rather than the bucket keeps the
	// page size fixed and the item order identical in both views.
	feeds := feedCounts(items, readingPageSize)
	pageItems, page, totalPages := paginate(items, requestedPage, readingPageSize)

	view := readingView{
		Items:      pageItems,
		Feeds:      feeds,
		Status:     activeStatus,
		Grouped:    grouped,
		IsAdmin:    auth.IsAdmin(ctx),
		Page:       page,
		TotalPages: totalPages,
	}
	if grouped {
		view.Groups = groupByFeed(pageItems)
	}

	ui.Render(w, r, ReadingPage(view))
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

	err = s.store.UpdateItemStatus(
		ctx, store.UpdateItemStatusParams{Status: status, ID: id},
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update item status", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Carry the view back with the redirect, otherwise marking one item read
	// throws the admin back to page 1 of the flat list.
	tab := normalizeStatus(r.FormValue("tab"))
	grouped := isGroupedByFeed(r.FormValue("group"))
	page, _ := strconv.Atoi(r.FormValue("page"))

	http.Redirect(w, r, readingURL(tab, grouped, page), http.StatusSeeOther)
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

func readingURL(status string, grouped bool, page int) string {
	query := url.Values{"status": {status}}
	if grouped {
		query.Set("group", groupFeed)
	}
	if page > 1 {
		query.Set("page", strconv.Itoa(page))
	}
	return "/reading?" + query.Encode()
}
