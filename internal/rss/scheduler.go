package rss

import (
	"context"
	"log/slog"
	"time"
)

// refreshTimeout caps a single scheduled run so a hung feed can't block the
// next day's refresh.
const refreshTimeout = 10 * time.Minute

// StartDailyRefresh blocks until ctx is cancelled, calling Refresh once per day
// at hour:min in loc. A failed run is logged and the schedule continues.
func (s *Service) StartDailyRefresh(ctx context.Context, hour, min int, loc *time.Location) {
	for {
		next := nextRun(time.Now(), hour, min, loc)
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

// nextRun returns the next occurrence of hour:min in loc strictly after now.
// It is computed from the current time on every tick rather than by adding 24h
// to the previous target, so clock jumps don't accumulate drift.
func nextRun(now time.Time, hour, min int, loc *time.Location) time.Time {
	n := now.In(loc)
	next := time.Date(n.Year(), n.Month(), n.Day(), hour, min, 0, 0, loc)
	if !next.After(n) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
