package post

import (
	"blog/internal/config"
	"blog/internal/content"
	"blog/internal/rss"
	"blog/internal/verse"
	"context"
	"sync/atomic"
	"time"
)

type PostService struct {
	cfg        *config.Config
	content    *content.Content
	index      atomic.Pointer[content.Index]
	refreshing atomic.Bool
	rssSvc     *rss.Service
	verseSvc   *verse.Service
}

func New(
	ctx context.Context,
	cfg *config.Config,
	cont *content.Content,
	rssSvc *rss.Service,
	verseSvc *verse.Service,
) (*PostService, error) {
	idx, err := cont.Build(ctx)
	if err != nil {
		return nil, err
	}
	s := &PostService{
		cfg:      cfg,
		content:  cont,
		rssSvc:   rssSvc,
		verseSvc: verseSvc,
	}
	s.index.Store(idx)

	return s, nil
}

type queryOptions struct {
	includeUnpublished bool
	language           *content.Language
	tags               []string
	limit              int
	offset             int
}

type QueryOption func(*queryOptions)

func withIncludeUnpublished(show bool) QueryOption {
	return func(q *queryOptions) {
		q.includeUnpublished = show
	}
}

func withLanguage(lang content.Language) QueryOption {
	return func(q *queryOptions) {
		q.language = &lang
	}
}

func withTags(tags ...string) QueryOption {
	return func(q *queryOptions) {
		q.tags = append(q.tags, tags...)
	}
}

func withLimit(limit int) QueryOption {
	return func(q *queryOptions) {
		if limit > 0 {
			q.limit = limit
		}
	}
}

func withOffset(offset int) QueryOption {
	return func(q *queryOptions) {
		if offset > 0 {
			q.offset = offset
		}
	}
}

func buildQueryOptions(opts ...QueryOption) queryOptions {
	q := queryOptions{}

	for _, opt := range opts {
		opt(&q)
	}

	return q
}

func (s *PostService) GetPostBySlug(slug string, opts ...QueryOption) (*content.Post, bool) {
	opt := buildQueryOptions(opts...)
	idx := s.index.Load()

	post, ok := idx.BySlug(slug)
	if !ok || !canView(post, time.Now().UTC(), opt) {
		return nil, false
	}
	if opt.language != nil && post.Language != *opt.language {
		return nil, false
	}

	return post, true
}

func (s *PostService) GetPosts(opts ...QueryOption) []*content.Post {
	opt := buildQueryOptions(opts...)
	idx := s.index.Load()

	// Narrow the posts set through a prebuilt index when possible; both All and
	// ByLanguage come back presorted newest-first.
	posts := idx.All()
	if opt.language != nil {
		posts = idx.ByLanguage(*opt.language)
	}

	// Allocate a new slice with the maximum possible capacity
	// to avoid mutating the global index
	filtered := make([]*content.Post, 0, len(posts))
	now := time.Now().UTC()

	for _, post := range posts {
		if canView(post, now, opt) && matchTags(post, opt) {
			filtered = append(filtered, post)
		}
	}

	if opt.offset >= len(filtered) {
		return []*content.Post{}
	}
	filtered = filtered[opt.offset:]

	if opt.limit > 0 && opt.limit < len(filtered) {
		filtered = filtered[:opt.limit]
	}

	return filtered
}

func (s *PostService) GetTags(opts ...QueryOption) []string {
	opt := buildQueryOptions(opts...)
	idx := s.index.Load()

	all := idx.Tags()
	if opt.includeUnpublished {
		return all
	}

	now := time.Now().UTC()
	tags := make([]string, 0, len(all))
	for _, tag := range all {
		for _, p := range idx.ByTag(tag) {
			if canView(p, now, opt) {
				tags = append(tags, tag)
				break
			}
		}
	}

	return tags
}

func canView(p *content.Post, now time.Time, opt queryOptions) bool {
	if opt.includeUnpublished {
		return true
	}
	return p.PublishedAt != nil && !p.PublishedAt.After(now)
}

func matchTags(p *content.Post, opt queryOptions) bool {
	if len(opt.tags) == 0 {
		return true
	}

	for _, t := range opt.tags {
		for _, pt := range p.Tags {
			if t == pt {
				return true
			}
		}
	}

	return false
}
