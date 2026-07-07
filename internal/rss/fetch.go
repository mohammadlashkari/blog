package rss

import (
	"blog/internal/rss/store"
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	unread   = "unread"
	read     = "read"
	archived = "archived"
)

type Feed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title         string `xml:"title"`
	Link          string `xml:"link"`
	Description   string `xml:"description"`
	Language      string `xml:"language"`
	Copyright     string `xml:"copyright"`
	LastBuildDate string `xml:"lastBuildDate"`
	Docs          string `xml:"docs"`
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

func (s *Service) fetchFeed(ctx context.Context, follow FollowedFeed) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, follow.URL, nil)
	if err != nil {
		return fmt.Errorf("create request for %s: %w", follow.URL, err)
	}
	req.Header.Set("User-Agent", "mohammadlashkari.com (RSS Reader)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http execute for %s: %w", follow.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s for %s", resp.Status, follow.URL)
	}

	var f Feed
	if err := xml.NewDecoder(resp.Body).Decode(&f); err != nil {
		return fmt.Errorf("decode rss feed %s: %w", follow.URL, err)
	}

	exist, err := s.store.CheckFeedExists(ctx, follow.URL)
	if err != nil {
		return fmt.Errorf("check feed exists: %w", err)
	}

	status := unread
	if !exist {
		status = archived
	}

	for _, item := range f.Channel.Items {
		guid := item.GUID
		if guid == "" {
			guid = item.Link
		}
		if guid == "" {
			slog.WarnContext(ctx, "skipping item with no guid or link", "feed", follow.URL)
			continue
		}

		var description *string
		if item.Description != "" {
			item.Description = html.UnescapeString(item.Description)
			descCopy := item.Description
			description = &descCopy
		}

		var pubDatePtr *time.Time
		if item.PubDate != "" {
			var (
				formats    = []string{time.RFC1123, time.RFC1123Z, time.RFC822, time.RFC822Z}
				parsedTime time.Time
				parseErr   error
			)
			for _, format := range formats {
				if parsedTime, parseErr = time.Parse(format, item.PubDate); parseErr == nil {
					pubDatePtr = &parsedTime
					break
				}
			}

			if parseErr != nil {
				slog.WarnContext(ctx, "failed to parse pub date using known formats, leaving nil",
					"date", item.PubDate,
					"guid", guid,
				)
			}
		}

		var categories *string
		if len(item.Categories) > 0 {
			joined := strings.Join(item.Categories, ",")
			categories = &joined
		}

		err = s.store.CreateRssItem(ctx, store.CreateRssItemParams{
			Guid:        guid,
			Link:        item.Link,
			FeedName:    follow.Name,
			FeedUrl:     follow.URL,
			Title:       html.UnescapeString(item.Title),
			Description: description,
			Status:      status,
			PublishedAt: pubDatePtr,
			Categories:  categories,
		})
		if err != nil {
			slog.WarnContext(ctx, "failed to create rss item", "guid", guid, "error", err)
			continue
		}
	}

	return nil
}
