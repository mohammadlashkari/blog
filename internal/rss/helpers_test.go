package rss

import (
	"blog/internal/rss/store"
	"slices"
	"testing"
)

// items builds a slice of RssItem carrying only the fields the grouping and
// paging helpers look at: the feed name (and a title, to tell rows apart).
func items(feedNames ...string) []store.RssItem {
	out := make([]store.RssItem, len(feedNames))
	for i, name := range feedNames {
		out[i] = store.RssItem{ID: int64(i), FeedName: name, Title: name}
	}
	return out
}

func feedNames(items []store.RssItem) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.FeedName
	}
	return out
}

func TestGroupByFeed(t *testing.T) {
	cases := []struct {
		name  string
		in    []store.RssItem
		want  []string // group names, in order
		sizes []int
	}{
		{
			name: "empty",
		},
		{
			name:  "single feed",
			in:    items("A", "A", "A"),
			want:  []string{"A"},
			sizes: []int{3},
		},
		{
			name:  "first appearance order wins, not count",
			in:    items("A", "B", "B", "B"),
			want:  []string{"A", "B"},
			sizes: []int{1, 3},
		},
		{
			name:  "interleaved feeds collapse into one group each",
			in:    items("A", "B", "A", "C", "B"),
			want:  []string{"A", "B", "C"},
			sizes: []int{2, 2, 1},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			groups := groupByFeed(c.in)

			if len(groups) != len(c.want) {
				t.Fatalf("got %d groups, want %d", len(groups), len(c.want))
			}

			var total int
			for i, group := range groups {
				if group.Name != c.want[i] {
					t.Errorf("group %d: got name %q, want %q", i, group.Name, c.want[i])
				}
				if len(group.Items) != c.sizes[i] {
					t.Errorf("group %q: got %d items, want %d", group.Name, len(group.Items), c.sizes[i])
				}
				if group.Slug != feedSlug(group.Name) {
					t.Errorf("group %q: slug %q does not match feedSlug", group.Name, group.Slug)
				}
				total += len(group.Items)
			}

			if total != len(c.in) {
				t.Errorf("grouping dropped items: got %d, want %d", total, len(c.in))
			}
		})
	}
}

func TestGroupByFeedPreservesItemOrder(t *testing.T) {
	in := []store.RssItem{
		{FeedName: "A", Title: "a1"},
		{FeedName: "B", Title: "b1"},
		{FeedName: "A", Title: "a2"},
	}

	groups := groupByFeed(in)

	if got := []string{groups[0].Items[0].Title, groups[0].Items[1].Title}; !slices.Equal(got, []string{"a1", "a2"}) {
		t.Errorf("got %v, want [a1 a2]", got)
	}
}

func TestFeedCounts(t *testing.T) {
	cases := []struct {
		name string
		in   []store.RssItem
		size int
		want []feedCount
	}{
		{
			name: "empty",
			size: 2,
			want: []feedCount{},
		},
		{
			name: "sorted by count descending",
			in:   items("A", "B", "B", "C", "B", "C"),
			size: 100,
			want: []feedCount{
				{Name: "B", Slug: "b", Total: 3, Page: 1},
				{Name: "C", Slug: "c", Total: 2, Page: 1},
				{Name: "A", Slug: "a", Total: 1, Page: 1},
			},
		},
		{
			name: "ties break on name ascending",
			in:   items("Zed", "Ada", "Mux"),
			size: 100,
			want: []feedCount{
				{Name: "Ada", Slug: "ada", Total: 1, Page: 1},
				{Name: "Mux", Slug: "mux", Total: 1, Page: 1},
				{Name: "Zed", Slug: "zed", Total: 1, Page: 1},
			},
		},
		{
			// Page must point at the feed's *first* item, since that is the only
			// page where grouping gives it a heading to jump to.
			name: "page tracks where each feed first appears",
			in:   items("A", "A", "B", "B", "C", "C"),
			size: 2,
			want: []feedCount{
				{Name: "A", Slug: "a", Total: 2, Page: 1},
				{Name: "B", Slug: "b", Total: 2, Page: 2},
				{Name: "C", Slug: "c", Total: 2, Page: 3},
			},
		},
		{
			name: "a feed straddling a page boundary points at the earlier page",
			in:   items("A", "B", "B", "B"),
			size: 2,
			want: []feedCount{
				{Name: "B", Slug: "b", Total: 3, Page: 1},
				{Name: "A", Slug: "a", Total: 1, Page: 1},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := feedCounts(c.in, c.size); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestFeedSlug(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "lowercased", in: "Hacker News", want: "hacker-news"},
		{name: "punctuation collapses", in: "Simon Willison's Weblog", want: "simon-willison-s-weblog"},
		{name: "runs of separators collapse to one dash", in: "Go   —  Blog", want: "go-blog"},
		{name: "leading separators dropped", in: "  ...Ars Technica", want: "ars-technica"},
		{name: "trailing separators dropped", in: "Golang Weekly!!", want: "golang-weekly"},
		{name: "digits kept", in: "Web 2.0 Roundup", want: "web-2-0-roundup"},
		{name: "non-latin letters kept", in: "وبلاگ فارسی", want: "وبلاگ-فارسی"},
		{name: "separators only", in: "--- !!! ---", want: ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := feedSlug(c.in); got != c.want {
				t.Errorf("feedSlug(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestPaginate(t *testing.T) {
	all := items("A", "B", "C", "D", "E")

	cases := []struct {
		name           string
		in             []store.RssItem
		page, size     int
		want           []string
		wantPage       int
		wantTotalPages int
	}{
		{
			name: "empty list is one empty page",
			// paginate must not report 0 pages: the pager and the clamp both
			// read TotalPages as a lower bound of 1.
			page: 1, size: 2,
			wantPage: 1, wantTotalPages: 1,
		},
		{
			name: "first page", in: all, page: 1, size: 2,
			want: []string{"A", "B"}, wantPage: 1, wantTotalPages: 3,
		},
		{
			name: "last page is short", in: all, page: 3, size: 2,
			want: []string{"E"}, wantPage: 3, wantTotalPages: 3,
		},
		{
			name: "page 0 (missing or unparsed param) clamps up to 1", in: all, page: 0, size: 2,
			want: []string{"A", "B"}, wantPage: 1, wantTotalPages: 3,
		},
		{
			name: "negative page clamps up to 1", in: all, page: -7, size: 2,
			want: []string{"A", "B"}, wantPage: 1, wantTotalPages: 3,
		},
		{
			name: "page past the end clamps down to the last", in: all, page: 999, size: 2,
			want: []string{"E"}, wantPage: 3, wantTotalPages: 3,
		},
		{
			name: "exact multiple leaves no trailing empty page", in: all[:4], page: 2, size: 2,
			want: []string{"C", "D"}, wantPage: 2, wantTotalPages: 2,
		},
		{
			name: "page larger than the list", in: all, page: 1, size: 50,
			want: []string{"A", "B", "C", "D", "E"}, wantPage: 1, wantTotalPages: 1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, page, totalPages := paginate(c.in, c.page, c.size)

			if names := feedNames(got); !slices.Equal(names, c.want) && !(len(names) == 0 && len(c.want) == 0) {
				t.Errorf("items: got %v, want %v", names, c.want)
			}
			if page != c.wantPage {
				t.Errorf("page: got %d, want %d", page, c.wantPage)
			}
			if totalPages != c.wantTotalPages {
				t.Errorf("totalPages: got %d, want %d", totalPages, c.wantTotalPages)
			}
		})
	}
}
