package rss

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

var pubDateFormats = []string{
	time.RFC1123Z,
	time.RFC822Z,
	time.RFC3339,
	time.RFC1123,
	time.RFC822,
	// RFC1123's "02" demands a zero-padded day, which several feeds omit.
	"Mon, 2 Jan 2006 15:04:05 MST",
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"2006-01-02",
}

// feedFormat is the syndication dialect a document is written in. RSS 1.0
// (<rdf:RDF>) is deliberately unsupported.
type feedFormat int

const (
	formatRSS feedFormat = iota + 1
	formatAtom
)

// parsedItem is the format-neutral shape both dialects normalize into, so the
// storage path never has to know which one it came from.
type parsedItem struct {
	GUID        string
	Link        string
	Title       string
	Description string
	Categories  []string
	Published   *time.Time
}

// RSS 2.0.

type Feed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title string `xml:"title"`
	// Link is written when generating this site's own feed. On the parse side
	// it is unused and unreliable: encoding/xml matches on local name, so a
	// feed's <atom:link rel="self"> also lands here.
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

// Atom (RFC 4287).

type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Entries []AtomEntry `xml:"entry"`
}

type AtomEntry struct {
	ID         string         `xml:"id"`
	Title      string         `xml:"title"`
	Links      []AtomLink     `xml:"link"`
	Published  string         `xml:"published"`
	Updated    string         `xml:"updated"`
	Summary    string         `xml:"summary"`
	Content    string         `xml:"content"`
	Categories []AtomCategory `xml:"category"`
	Author     AtomAuthor     `xml:"author"`
}

// AtomLink carries the target in an href attribute rather than in the element
// body, which is the main structural difference from RSS.
type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type AtomCategory struct {
	Term string `xml:"term,attr"`
}

type AtomAuthor struct {
	Name string `xml:"name"`
}

// newDecoder builds a decoder tolerant of what feeds actually ship: HTML named
// entities XML leaves undefined (&nbsp;, &mdash;) and the occasional
// non-UTF-8 encoding.
func newDecoder(body []byte) *xml.Decoder {
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.Entity = xml.HTMLEntity
	dec.CharsetReader = charset.NewReaderLabel
	return dec
}

// detectFormat reports the dialect from the root element. It exists so a
// misconfigured feed URL that serves HTML fails with a message naming what was
// found instead of an opaque unmarshalling error.
func detectFormat(body []byte) (feedFormat, error) {
	dec := newDecoder(body)
	for {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0, errors.New("no xml elements found")
			}
			return 0, fmt.Errorf("scan for root element: %w", err)
		}

		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch start.Name.Local {
		case "rss":
			return formatRSS, nil
		case "feed":
			return formatAtom, nil
		default:
			return 0, fmt.Errorf("unsupported feed root element <%s>", start.Name.Local)
		}
	}
}

// parseFeed normalizes a feed document into items ready for storage. feedURL is
// the address the body was fetched from; it resolves relative links.
func parseFeed(ctx context.Context, body []byte, feedURL string) ([]parsedItem, error) {
	format, err := detectFormat(body)
	if err != nil {
		return nil, err
	}

	switch format {
	case formatRSS:
		return parseRSS(ctx, body, feedURL)
	case formatAtom:
		return parseAtom(ctx, body, feedURL)
	default:
		return nil, fmt.Errorf("unhandled feed format %d", format)
	}
}

func parseRSS(ctx context.Context, body []byte, feedURL string) ([]parsedItem, error) {
	var f Feed
	if err := newDecoder(body).Decode(&f); err != nil {
		return nil, fmt.Errorf("decode rss feed: %w", err)
	}

	items := make([]parsedItem, 0, len(f.Channel.Items))
	for _, item := range f.Channel.Items {
		p := parsedItem{
			GUID:        strings.TrimSpace(item.GUID),
			Link:        cleanURL(item.Link),
			Title:       cleanText(item.Title),
			Description: cleanDescription(item.Description),
			Categories:  cleanCategories(item.Categories),
			Published:   parseTime(ctx, feedURL, item.PubDate),
		}
		p.normalize(feedURL)
		items = append(items, p)
	}

	return items, nil
}

func parseAtom(ctx context.Context, body []byte, feedURL string) ([]parsedItem, error) {
	var f AtomFeed
	if err := newDecoder(body).Decode(&f); err != nil {
		return nil, fmt.Errorf("decode atom feed: %w", err)
	}

	items := make([]parsedItem, 0, len(f.Entries))
	for _, entry := range f.Entries {
		// <summary> is the abstract and <content> the full body; the reading
		// list only shows two clamped lines, so prefer the shorter one.
		body := entry.Summary
		if strings.TrimSpace(body) == "" {
			body = entry.Content
		}

		terms := make([]string, 0, len(entry.Categories))
		for _, c := range entry.Categories {
			terms = append(terms, c.Term)
		}

		p := parsedItem{
			GUID:        strings.TrimSpace(entry.ID),
			Link:        atomLink(entry.Links),
			Title:       cleanText(entry.Title),
			Description: cleanDescription(body),
			Categories:  cleanCategories(terms),
			// <updated> is required, <published> is not; prefer the original
			// publication date and fall back to the last edit.
			Published: parseTime(ctx, feedURL, entry.Published, entry.Updated),
		}
		p.normalize(feedURL)
		items = append(items, p)
	}

	return items, nil
}

// atomLink picks the entry's human-readable page. Entries carry several <link>
// elements distinguished by rel; "alternate" is that page, and is also the
// default when rel is omitted. Anything else (rel="self", rel="replies") is
// skipped rather than used as a fallback — a link to the feed itself is worse
// than no link at all, which the caller can detect and skip.
func atomLink(links []AtomLink) string {
	for _, l := range links {
		if l.Rel != "alternate" && l.Rel != "" {
			continue
		}
		if href := cleanURL(l.Href); href != "" {
			return href
		}
	}
	return ""
}

// normalize fills in the cross-references between an item's identifier and its
// link, and makes the link absolute.
func (p *parsedItem) normalize(feedURL string) {
	p.Link = absoluteURL(p.Link, feedURL)

	// RSS 2.0 permits <guid isPermaLink="true"> to stand in for a missing
	// <link>, and Atom ids are sometimes plain URLs.
	if p.Link == "" && isHTTPURL(p.GUID) {
		p.Link = p.GUID
	}
	if p.GUID == "" {
		p.GUID = p.Link
	}
}

// cleanURL trims the whitespace pretty-printed feeds leave inside elements. It
// deliberately does not decode entities: encoding/xml has already done that for
// element text, and html.UnescapeString would additionally eat query parameters
// named after semicolon-less legacy entities (?lt=, ?copy=, ?amp=).
func cleanURL(s string) string {
	return strings.TrimSpace(s)
}

// cleanText decodes entities the XML parser leaves alone inside CDATA sections,
// which is how most feeds wrap titles.
func cleanText(s string) string {
	return strings.TrimSpace(html.UnescapeString(strings.TrimSpace(s)))
}

func cleanCategories(in []string) []string {
	out := make([]string, 0, len(in))
	for _, c := range in {
		if c = cleanText(c); c != "" {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseTime returns the first candidate that parses under any known layout.
func parseTime(ctx context.Context, feedURL string, candidates ...string) *time.Time {
	var tried []string
	for _, raw := range candidates {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		tried = append(tried, raw)

		for _, format := range pubDateFormats {
			if t, err := time.Parse(format, raw); err == nil {
				return &t
			}
		}
	}

	if len(tried) > 0 {
		slog.WarnContext(ctx, "failed to parse feed date using known formats, leaving nil",
			"dates", tried,
			"feed", feedURL,
		)
	}
	return nil
}

// absoluteURL resolves a relative href against the feed's own address. Atom
// permits relative links, which would otherwise render as links into this site.
func absoluteURL(href, feedURL string) string {
	if href == "" {
		return ""
	}

	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	if u.IsAbs() {
		return href
	}

	base, err := url.Parse(feedURL)
	if err != nil {
		return href
	}
	return base.ResolveReference(u).String()
}

func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
