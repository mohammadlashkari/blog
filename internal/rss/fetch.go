package rss

import (
	"blog/internal/rss/store"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const (
	unread   = "unread"
	read     = "read"
	archived = "archived"
	saved    = "saved"
)

const maxFeedBytes = 10 << 20 // 10MB

func (s *Service) fetchFeed(ctx context.Context, follow FollowedFeed) error {
	body, err := s.downloadFeed(ctx, follow.URL)
	if err != nil {
		return err
	}

	items, err := parseFeed(ctx, body, follow.URL)
	if err != nil {
		return fmt.Errorf("parse feed %s: %w", follow.URL, err)
	}

	exists, err := s.store.CheckFeedExists(ctx, follow.URL)
	if err != nil {
		return fmt.Errorf("failed to check whether feed %q exists: %w", follow.URL, err)
	}

	var status string
	if exists {
		// New items land in the unread inbox
		status = unread
	} else {
		status = archived
	}

	for _, item := range items {
		if item.GUID == "" {
			slog.WarnContext(ctx, "skipping item with no guid or link", "feed", follow.URL)
			continue
		}
		// An entry with no link is dead weight in a reading list: the card
		// renders, but clicking it goes nowhere.
		if item.Link == "" {
			slog.WarnContext(ctx, "skipping item with no link", "feed", follow.URL, "guid", item.GUID)
			continue
		}

		var description *string
		if item.Description != "" {
			description = &item.Description
		}

		var categories *string
		if len(item.Categories) > 0 {
			joined := strings.Join(item.Categories, ",")
			categories = &joined
		}

		err = s.store.CreateRssItem(ctx, store.CreateRssItemParams{
			Guid:        item.GUID,
			Link:        item.Link,
			FeedName:    follow.Name,
			FeedUrl:     follow.URL,
			Title:       item.Title,
			Description: description,
			Status:      status,
			PublishedAt: item.Published,
			Categories:  categories,
		})
		if err != nil {
			slog.WarnContext(ctx, "failed to create rss item", "guid", item.GUID, "error", err)
			continue
		}
	}

	return nil
}

func (s *Service) downloadFeed(ctx context.Context, feedURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request for %s: %w", feedURL, err)
	}
	req.Header.Set("User-Agent", "mohammadlashkari.com (RSS Reader)")
	req.Header.Set("Accept", "application/atom+xml, application/rss+xml, application/xml;q=0.9, */*;q=0.8")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http execute for %s: %w", feedURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s for %s", resp.Status, feedURL)
	}

	// Buffered rather than streamed: detecting the dialect and decoding it both
	// need to read from the start.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes))
	if err != nil {
		return nil, fmt.Errorf("read feed body for %s: %w", feedURL, err)
	}

	return body, nil
}
