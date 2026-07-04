package content

import "sort"

// Index is an immutable snapshot of all posts, built once and swapped atomically
type Index struct {
	posts      []*Post              // all posts, drafts first, then newest-first
	bySlug     map[string]*Post     // slug -> post, O(1) lookup
	byTag      map[string][]*Post   // tag -> posts, drafts first, then newest-first
	byLanguage map[Language][]*Post // language -> posts, drafts first, then newest-first
	tags       []string             // unique tags, sorted
}

func (c *Content) buildIndex() (*Index, error) {
	bySlug, err := c.fsPosts(c.postsPath)
	if err != nil {
		return nil, err
	}

	idx := &Index{
		posts:      make([]*Post, 0, len(bySlug)),
		bySlug:     bySlug,
		byTag:      make(map[string][]*Post),
		byLanguage: make(map[Language][]*Post),
	}

	seen := make(map[string]struct{})
	for _, p := range bySlug {
		idx.posts = append(idx.posts, p)
		idx.byLanguage[p.Language] = append(idx.byLanguage[p.Language], p)
		for _, tag := range p.Tags {
			if _, ok := seen[tag]; !ok {
				seen[tag] = struct{}{}
				idx.tags = append(idx.tags, tag)
			}
			idx.byTag[tag] = append(idx.byTag[tag], p)
		}
	}

	// Sort once at build time so reads are cheap and deterministic.
	sortByNewest(idx.posts)
	for _, ps := range idx.byTag {
		sortByNewest(ps)
	}
	for _, ps := range idx.byLanguage {
		sortByNewest(ps)
	}
	sort.Strings(idx.tags)

	return idx, nil
}

// All returns a slice of all posts.
// WARNING: The returned slice shares backing storage with the index.
// Callers MUST NOT mutate the slice
func (i *Index) All() []*Post { return i.posts }

func (i *Index) BySlug(slug string) (*Post, bool) {
	p, ok := i.bySlug[slug]
	return p, ok
}

func (i *Index) ByTag(tag string) []*Post { return i.byTag[tag] }

func (i *Index) ByLanguage(lang Language) []*Post { return i.byLanguage[lang] }

func (i *Index) Tags() []string { return i.tags }

// sortByNewest orders drafts first (unpublished = newest work, not yet
// released), then published posts by PublishedAt descending, slug as tiebreak.
func sortByNewest(posts []*Post) {
	sort.Slice(posts, func(i, j int) bool {
		postA, postB := posts[i], posts[j]

		isDraftA := postA.PublishedAt == nil
		isDraftB := postB.PublishedAt == nil

		// Drafts always sort before published posts
		if isDraftA != isDraftB {
			return isDraftA
		}

		// If both are published, sort by date (newest first)
		if !isDraftA && !postA.PublishedAt.Equal(*postB.PublishedAt) {
			return postA.PublishedAt.After(*postB.PublishedAt)
		}

		// Tie-breaker: sort by slug alphabetically
		return postA.Slug < postB.Slug
	})
}
