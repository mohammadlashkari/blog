package rss

import (
	"blog/internal/config"
	"blog/internal/rss/store"
	"context"
	"crypto/md5"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

type Service struct {
	cfg        *config.Config
	store      store.Storer
	httpClient *http.Client
	refreshing atomic.Bool
	lastHash   [md5.Size]byte
}

func New(cfg *config.Config, store store.Storer) *Service {
	return &Service{
		cfg:   cfg,
		store: store,
		httpClient: &http.Client{
			Timeout: 1 * time.Minute,
		},
	}
}

type FollowedFeed struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name"`
}

type Reading struct {
	Feeds []FollowedFeed `yaml:"feeds"`
}

// SyncFeeds is triggered by your web-push hook.
// It skips fetching if the configuration file hasn't changed.
func (s *Service) SyncFeeds(ctx context.Context) error {
	slog.InfoContext(ctx, "starting rss sync via webhook")
	return s.refresh(ctx, true)
}

// FetchAll is triggered by your cron job.
// It bypasses the file hash check to fetch fresh content from all feeds.
func (s *Service) FetchAll(ctx context.Context) error {
	slog.InfoContext(ctx, "starting periodic cron rss fetch")
	return s.refresh(ctx, false)
}

func (s *Service) refresh(ctx context.Context, checkHash bool) error {
	if !s.refreshing.CompareAndSwap(false, true) {
		slog.InfoContext(ctx, "rss refresh already in progress, skipping")
		return nil
	}
	defer s.refreshing.Store(false)

	path := s.feedsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read feeds file", "path", path, "error", err)
		return fmt.Errorf("failed to read feeds file: %w", err)
	}

	var r Reading
	if err := yaml.Unmarshal(data, &r); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal feeds yaml", "error", err)
		return fmt.Errorf("failed to unmarshal feeds yaml: %w", err)
	}

	newHash := md5.Sum(data)
	if checkHash && s.lastHash == newHash {
		slog.InfoContext(ctx, "feeds file unchanged, skipping remote fetch")
		return nil
	}

	var errs []error
	for _, feed := range r.Feeds {
		if err := s.fetchFeed(ctx, feed); err != nil {
			slog.ErrorContext(ctx, "fetch feed failed", "feed", feed, "error", err)
			errs = append(errs, fmt.Errorf("feed %s: %w", feed, err))
			continue
		}
	}

	s.lastHash = newHash
	slog.InfoContext(ctx, "rss refresh process completed")

	if len(errs) > 0 {
		return fmt.Errorf("completed with %d feed errors", len(errs))
	}

	return nil
}

func (s *Service) feedsPath() string {
	return filepath.Join(s.cfg.LocalContentPath, "reading", "feeds.md")
}
