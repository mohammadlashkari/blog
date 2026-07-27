package rss

import (
	"testing"
	"time"
)

func TestNextRun(t *testing.T) {
	tehran, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		now       time.Time
		hour, min int
		loc       *time.Location
		want      time.Time
	}{
		{
			name: "before target today",
			now:  time.Date(2026, 7, 27, 13, 5, 0, 0, time.UTC),
			loc:  time.UTC,
			want: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "after midnight rolls to tomorrow",
			now:  time.Date(2026, 7, 27, 0, 0, 1, 0, time.UTC),
			loc:  time.UTC,
			want: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "exactly at target rolls to tomorrow",
			now:  time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
			loc:  time.UTC,
			want: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "later target same day",
			now:  time.Date(2026, 7, 27, 9, 0, 0, 0, time.UTC),
			hour: 23, min: 30,
			loc:  time.UTC,
			want: time.Date(2026, 7, 27, 23, 30, 0, 0, time.UTC),
		},
		{
			name: "month rollover",
			now:  time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC),
			loc:  time.UTC,
			want: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "year rollover",
			now:  time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
			loc:  time.UTC,
			want: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			// 20:30 UTC is already past midnight in Tehran (+03:30).
			name: "target resolved in loc, not in now's zone",
			now:  time.Date(2026, 7, 27, 20, 30, 0, 0, time.UTC),
			loc:  tehran,
			want: time.Date(2026, 7, 29, 0, 0, 0, 0, tehran),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nextRun(tc.now, tc.hour, tc.min, tc.loc)
			if !got.Equal(tc.want) {
				t.Errorf("nextRun(%v) = %v, want %v", tc.now, got, tc.want)
			}
			if !got.After(tc.now) {
				t.Errorf("nextRun(%v) = %v, not in the future", tc.now, got)
			}
		})
	}
}
