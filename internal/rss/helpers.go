package rss

import (
	"blog/internal/rss/store"
	"html"
	"regexp"
	"slices"
	"strings"
)

const (
	unread    = "unread"
	read      = "read"
	archived  = "archived"
	saved     = "saved"
	groupFeed = "feed"
)

var (
	htmlTagRE = regexp.MustCompile(`<[^>]+>`)

	// The classes are Unicode-aware on purpose: feed names come from feeds.yaml and
	// can be non-Latin, so narrowing this to [^a-z0-9]+ would slug every Farsi feed
	// down to an empty string.
	nonAlnumRE = regexp.MustCompile(`[^\p{L}\p{Nd}]+`)
)

func isGroupedByFeed(value string) bool {
	return value == groupFeed
}

func cleanDescription(s string) string {
	s = html.UnescapeString(s)
	s = htmlTagRE.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func normalizeStatus(s string) string {
	switch s {
	case read, archived, saved:
		return s
	default:
		return unread
	}
}

func isStatusValid(s string) bool {
	return slices.Contains([]string{unread, read, saved, archived}, s)
}

// readingView is everything the reading page needs. It replaces the loose
// argument list ReadingPage used to take, which stopped scaling once grouping
// and paging arrived.
type readingView struct {
	Items      []store.RssItem // current page, flat newest-first
	Groups     []feedGroup     // populated only when Grouped
	Feeds      []feedCount     // index over the whole status bucket, not just this page
	Status     string
	Grouped    bool
	IsAdmin    bool
	Page       int
	TotalPages int
}

type feedGroup struct {
	Name  string
	Slug  string
	Items []store.RssItem
}

// feedCount is one entry of the feed index strip above the list.
type feedCount struct {
	Name  string
	Slug  string
	Total int
	// Page is where this feed's newest item falls. Grouping happens per page,
	// so a feed only has a heading to jump to on the pages it appears on —
	// without this the index would link to headings that aren't rendered.
	Page int
}

// groupByFeed buckets items by feed name, keeping first-appearance order. The
// caller passes a slice already sorted newest-first, so the feed holding the
// newest item leads and each group stays newest-first internally.
func groupByFeed(items []store.RssItem) []feedGroup {
	var groups []feedGroup
	positions := make(map[string]int) // feed name -> its index in groups

	for _, item := range items {
		pos, seen := positions[item.FeedName]
		if !seen {
			pos = len(groups)
			positions[item.FeedName] = pos
			groups = append(groups, feedGroup{
				Name: item.FeedName,
				Slug: feedSlug(item.FeedName),
			})
		}
		groups[pos].Items = append(groups[pos].Items, item)
	}

	return groups
}

// feedCounts tallies items per feed over the whole status bucket, biggest
// first, name-ascending on ties. size is the page size, used to work out which
// page each feed's heading will be on.
func feedCounts(items []store.RssItem, size int) []feedCount {
	counts := make(map[string]int)
	first := make(map[string]int)

	for i, item := range items {
		if _, seen := counts[item.FeedName]; !seen {
			first[item.FeedName] = i
		}
		counts[item.FeedName]++
	}

	feeds := make([]feedCount, 0, len(counts))
	for name, n := range counts {
		feeds = append(feeds, feedCount{
			Name:  name,
			Slug:  feedSlug(name),
			Total: n,
			Page:  first[name]/size + 1,
		})
	}

	slices.SortFunc(feeds, func(a, b feedCount) int {
		if a.Total != b.Total {
			return b.Total - a.Total
		}
		return strings.Compare(a.Name, b.Name)
	})

	return feeds
}

// feedSlug turns a feed name into a stable id for the heading it links to — the
// same idea as a post's slug.
func feedSlug(name string) string {
	s := nonAlnumRE.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(s, "-")
}

// paginate cuts one page out of items. An out-of-range or unparsed page (0) is
// pulled back into range rather than 404ing, so the caller should use the
// returned page number, not the one it asked for.
func paginate(items []store.RssItem, requested, size int) (pageItems []store.RssItem, page, totalPages int) {
	totalPages = max((len(items)+size-1)/size, 1)
	page = min(max(requested, 1), totalPages)

	start := (page - 1) * size
	end := min(start+size, len(items))
	if start >= end {
		return nil, page, totalPages
	}

	return items[start:end], page, totalPages
}
