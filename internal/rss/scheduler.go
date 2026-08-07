package rss

import (
	"context"
	"log/slog"
	"time"
)

const (
	refreshTimeout = 10 * time.Minute
	refreshHour    = 0
	refreshMin     = 0
)

var refreshLoc = time.UTC

// StartDailyRefresh blocks until ctx is cancelled, calling Refresh once per day
// at refreshHour:refreshMin in refreshLoc. A failed or timed-out run is logged
// and the schedule carries on.
func (s *Service) StartDailyRefresh(ctx context.Context) {
	for {
		next := nextRun(time.Now(), refreshHour, refreshMin, refreshLoc)
		slog.InfoContext(ctx, "next scheduled rss refresh", "at", next)

		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			slog.InfoContext(ctx, "rss refresh scheduler stopped")
			return
		case <-timer.C:
		}

		s.runScheduledRefresh(ctx)
	}
}

func (s *Service) runScheduledRefresh(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()

	if err := s.Refresh(runCtx); err != nil {
		slog.ErrorContext(runCtx, "scheduled rss refresh failed", "error", err)
	}
}

func nextRun(now time.Time, hour, min int, loc *time.Location) time.Time {
	now = now.In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, loc)

	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
