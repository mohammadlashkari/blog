package rss

import (
	"blog/internal/config"
	"blog/internal/rss/store"
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type RSSService struct {
	cfg         *config.Config
	store       store.Storer
	httpClient  *http.Client
	readingList []string
}

func New(cfg *config.Config, store store.Storer) *RSSService {
	return &RSSService{
		cfg:   cfg,
		store: store,
		httpClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
	}
}

// reading-list.md
var feed_list = []string{
	"https://techcrunch.com/feed/",
	"https://www.wagslane.dev/index.xml",
	"https://news.ycombinator.com/rss",
}

type Feed struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title         string `xml:"title"`
	Link          string `xml:"link"`
	Description   string `xml:"description"`
	Language      string `xml:"language"`
	LastBuildDate string `xml:"lastBuildDate"`
	Items         []Item `xml:"item"`
}

type Item struct {
	GUID        string   `xml:"guid"`
	Title       string   `xml:"title"`
	Author      string   `xml:"author"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	Categories  []string `xml:"category"`
	PubDate     string   `xml:"pubDate"`
}

// cron job
func (rs *RSSService) Run() {

}

func (rs *RSSService) FetchFeed(ctx context.Context, feedURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "mohammadlashkari.com (RSS Reader)")

	resp, err := rs.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s for %s", resp.Status, feedURL)
	}

	var f Feed
	if err := xml.NewDecoder(resp.Body).Decode(&f); err != nil {
		return fmt.Errorf("decode rss feed %s: %w", feedURL, err)
	}

	exist, err := rs.store.CheckFeedExists(ctx, feedURL)
	if err != nil {
		slog.ErrorContext(ctx, "check feed exists", "error", err)
		return fmt.Errorf("check feed exists: %w", err)
	}

	status := "unread"
	if !exist {
		status = "archived"
	}

	for _, item := range f.Channel.Items {
		// Fallback for missing GUID
		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}
		if guid == "" {
			slog.WarnContext(ctx, "skipping item with no guid or link", "feed", feedURL)
			continue
		}

		var description *string
		if item.Description != "" {
			description = &item.Description
		}

		var pubDatePtr *time.Time
		if item.PubDate != "" {
			pubDate, err := time.Parse(time.RFC1123, item.PubDate)
			if err == nil {
				pubDatePtr = &pubDate
			} else {
				slog.WarnContext(ctx, "failed to parse pub date, leaving nil", "date", item.PubDate)
			}
		}

		err = rs.store.CreateRssItem(ctx, store.CreateRssItemParams{
			Guid:        guid,
			Link:        item.Link,
			Title:       item.Title,
			Description: description,
			FeedUrl:     feedURL,
			Status:      status,
			PublishedAt: pubDatePtr,
		})
		if err != nil {
			slog.WarnContext(ctx, "failed to create rss item", "guid", guid, "error", err)
			continue
		}

	}

	return nil
}
