package rss

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/a-h/templ"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()

	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestDetectFormat(t *testing.T) {
	cases := []struct {
		name    string
		fixture string
		want    feedFormat
		wantErr bool
	}{
		{name: "rss", fixture: "rss_pretty.xml", want: formatRSS},
		{name: "rss with namespaces", fixture: "rss_wordpress.xml", want: formatRSS},
		{name: "atom", fixture: "atom.xml", want: formatAtom},
		{name: "html directory index", fixture: "notafeed.html", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := detectFormat(readFixture(t, tc.fixture))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("detectFormat() = %v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("detectFormat() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("detectFormat() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDetectFormatEmpty(t *testing.T) {
	if _, err := detectFormat(nil); err == nil {
		t.Fatal("detectFormat(nil) = nil error, want error")
	}
}

func TestParseFeedRSSPrettyPrinted(t *testing.T) {
	const feedURL = "https://antirez.com/rss"

	items, err := parseFeed(t.Context(), readFixture(t, "rss_pretty.xml"), feedURL)
	if err != nil {
		t.Fatalf("parseFeed() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	// The feed wraps element bodies in newlines. An untrimmed link fails
	// templ.URL's scheme check and renders as about:invalid.
	first := items[0]
	if want := "http://antirez.com/news/172"; first.Link != want {
		t.Errorf("Link = %q, want %q", first.Link, want)
	}
	if want := "The real AI risk is inside the labs"; first.Title != want {
		t.Errorf("Title = %q, want %q", first.Title, want)
	}
	if want := "http://antirez.com/news/172"; first.GUID != want {
		t.Errorf("GUID = %q, want %q", first.GUID, want)
	}
	if first.Published == nil {
		t.Fatal("Published = nil, want a time")
	}
	if want := time.Date(2026, 7, 28, 11, 0, 11, 0, time.FixedZone("", 2*60*60)); !first.Published.Equal(want) {
		t.Errorf("Published = %v, want %v", first.Published, want)
	}

	// Query parameters named after semicolon-less legacy HTML entities must
	// survive: html.UnescapeString would rewrite them to "<" and "©".
	if want := "http://antirez.com/news/171?a=1&lt=2&copy=3"; items[1].Link != want {
		t.Errorf("Link = %q, want %q", items[1].Link, want)
	}
}

func TestParseFeedRSSNamespacedLink(t *testing.T) {
	// This feed carries <atom:link rel="self"> alongside <link>. encoding/xml
	// matches on local name, so the namespaced element must not clobber the
	// item's real link.
	items, err := parseFeed(t.Context(), readFixture(t, "rss_wordpress.xml"), "https://dave.cheney.net/feed")
	if err != nil {
		t.Fatalf("parseFeed() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}

	got := items[0]
	if want := "https://dave.cheney.net/2025/12/18/pop-quiz-what-time-was-it"; got.Link != want {
		t.Errorf("Link = %q, want %q", got.Link, want)
	}
	// <guid isPermaLink="false"> is an identifier, not the link.
	if want := "https://dave.cheney.net/?p=4428"; got.GUID != want {
		t.Errorf("GUID = %q, want %q", got.GUID, want)
	}
	// CDATA hides entities from the XML decoder, so cleanDescription has to
	// unescape them itself; tags are stripped.
	if want := "Here’s a small quiz derived from some incorrect advice."; got.Description != want {
		t.Errorf("Description = %q, want %q", got.Description, want)
	}
	if want := []string{"Go", "Testing"}; !slices.Equal(got.Categories, want) {
		t.Errorf("Categories = %v, want %v", got.Categories, want)
	}
}

func TestParseFeedAtom(t *testing.T) {
	const feedURL = "https://eli.thegreenplace.net/feeds/all.atom.xml"

	items, err := parseFeed(t.Context(), readFixture(t, "atom.xml"), feedURL)
	if err != nil {
		t.Fatalf("parseFeed() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}

	t.Run("prefers rel=alternate over rel=self", func(t *testing.T) {
		want := "https://eli.thegreenplace.net/2026/relative-velocity-and-closing-speed/"
		if items[0].Link != want {
			t.Errorf("Link = %q, want %q", items[0].Link, want)
		}
	})

	t.Run("id becomes the guid", func(t *testing.T) {
		want := "tag:eli.thegreenplace.net,2026-08-03:/2026/relative-velocity-and-closing-speed/"
		if items[0].GUID != want {
			t.Errorf("GUID = %q, want %q", items[0].GUID, want)
		}
	})

	t.Run("prefers published over updated", func(t *testing.T) {
		if items[0].Published == nil {
			t.Fatal("Published = nil, want a time")
		}
		want := time.Date(2026, 8, 3, 20, 1, 0, 0, time.FixedZone("", -7*60*60))
		if !items[0].Published.Equal(want) {
			t.Errorf("Published = %v, want %v", items[0].Published, want)
		}
	})

	t.Run("prefers summary over content", func(t *testing.T) {
		want := "In Physics simulations it’s sometimes useful to determine the speed with which two objects are approaching."
		if items[0].Description != want {
			t.Errorf("Description = %q, want %q", items[0].Description, want)
		}
	})

	t.Run("term attribute becomes the category", func(t *testing.T) {
		if want := []string{"misc", "Math"}; !slices.Equal(items[0].Categories, want) {
			t.Errorf("Categories = %v, want %v", items[0].Categories, want)
		}
	})

	t.Run("relative href resolves against the feed url", func(t *testing.T) {
		want := "https://eli.thegreenplace.net/2026/no-published-date/"
		if items[1].Link != want {
			t.Errorf("Link = %q, want %q", items[1].Link, want)
		}
	})

	t.Run("falls back to updated and to content", func(t *testing.T) {
		if items[1].Published == nil {
			t.Fatal("Published = nil, want a time")
		}
		want := time.Date(2026, 7, 1, 10, 0, 0, 0, time.FixedZone("", -7*60*60))
		if !items[1].Published.Equal(want) {
			t.Errorf("Published = %v, want %v", items[1].Published, want)
		}
		if items[1].Description == "" {
			t.Error("Description is empty, want the content body")
		}
	})

	t.Run("rel=self only yields no link", func(t *testing.T) {
		// A link back to the feed document is worse than none; fetchFeed skips
		// these rather than storing an entry that opens the XML.
		if items[2].Link != "" {
			t.Errorf("Link = %q, want empty", items[2].Link)
		}
	})
}

func TestParseFeedRejectsHTML(t *testing.T) {
	_, err := parseFeed(t.Context(), readFixture(t, "notafeed.html"), "https://eli.thegreenplace.net/feeds")
	if err == nil {
		t.Fatal("parseFeed() = nil error, want error")
	}
}

// TestParsedLinksSurviveSanitization guards the whole point of the trimming and
// resolution above: templ.URL replaces anything it cannot recognize as http,
// https, mailto or tel, and the replacement renders as a dead link.
func TestParsedLinksSurviveSanitization(t *testing.T) {
	fixtures := []string{"rss_pretty.xml", "rss_wordpress.xml", "atom.xml"}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			items, err := parseFeed(t.Context(), readFixture(t, fixture), "https://example.com/feed.xml")
			if err != nil {
				t.Fatalf("parseFeed() error = %v", err)
			}

			for _, item := range items {
				if item.Link == "" {
					continue
				}
				if got := string(templ.URL(item.Link)); got != item.Link {
					t.Errorf("templ.URL(%q) = %q, want it unchanged", item.Link, got)
				}
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	cases := []struct {
		name       string
		candidates []string
		want       time.Time // zero means "expect nil"
	}{
		{
			name:       "rfc1123z",
			candidates: []string{"Tue, 28 Jul 2026 11:00:11 +0200"},
			want:       time.Date(2026, 7, 28, 11, 0, 11, 0, time.FixedZone("", 2*60*60)),
		},
		{
			name:       "rfc3339",
			candidates: []string{"2026-08-03T20:01:00-07:00"},
			want:       time.Date(2026, 8, 3, 20, 1, 0, 0, time.FixedZone("", -7*60*60)),
		},
		{
			// time.RFC1123 requires a zero-padded day and rejects this.
			name:       "single digit day",
			candidates: []string{"Mon, 1 Jun 2026 08:01:00 GMT"},
			want:       time.Date(2026, 6, 1, 8, 1, 0, 0, time.UTC),
		},
		{
			name:       "date only",
			candidates: []string{"2026-08-03"},
			want:       time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			name:       "skips empty candidates",
			candidates: []string{"", "  ", "2026-08-03"},
			want:       time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			name:       "falls through to the next candidate",
			candidates: []string{"not a date", "2026-08-03"},
			want:       time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
		},
		{name: "no candidates"},
		{name: "all unparseable", candidates: []string{"whenever"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTime(t.Context(), "https://example.com/feed.xml", tc.candidates...)

			if tc.want.IsZero() {
				if got != nil {
					t.Fatalf("parseTime() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("parseTime() = nil, want a time")
			}
			if !got.Equal(tc.want) {
				t.Errorf("parseTime() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAbsoluteURL(t *testing.T) {
	const feedURL = "https://eli.thegreenplace.net/feeds/all.atom.xml"

	cases := []struct {
		name string
		href string
		want string
	}{
		{name: "empty stays empty", href: "", want: ""},
		{name: "absolute is untouched", href: "https://other.example/post", want: "https://other.example/post"},
		{name: "root relative", href: "/2026/post/", want: "https://eli.thegreenplace.net/2026/post/"},
		{name: "path relative", href: "post/", want: "https://eli.thegreenplace.net/feeds/post/"},
		{name: "protocol relative", href: "//cdn.example/post", want: "https://cdn.example/post"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := absoluteURL(tc.href, feedURL); got != tc.want {
				t.Errorf("absoluteURL(%q) = %q, want %q", tc.href, got, tc.want)
			}
		})
	}
}
