package post

import (
	"blog/internal/config"
	"blog/internal/content"
	"context"
	"sort"
	"sync/atomic"
	"time"
)

type Service struct {
	cfg     *config.Config
	content *content.Content
	index   atomic.Pointer[content.Index]
}

func New(ctx context.Context, cfg *config.Config, cont *content.Content) (*Service, error) {
	idx, err := cont.Build(ctx)
	if err != nil {
		return nil, err
	}
	s := &Service{
		cfg:     cfg,
		content: cont,
	}
	s.index.Store(idx)

	return s, nil
}

type SortOrder int

const (
	SortNone SortOrder = iota
	SortNewestFirst
	SortOldestFirst
)

type queryOptions struct {
	includeUnpublished bool
	language           *content.Language
	tags               []string
	limit              int
	offset             int
	sortOrder          SortOrder
}

type QueryOption func(*queryOptions)

func WithIncludeUnpublished(show bool) QueryOption {
	return func(q *queryOptions) {
		q.includeUnpublished = show
	}
}

func WithLanguage(lang content.Language) QueryOption {
	return func(q *queryOptions) {
		q.language = &lang
	}
}

func WithTags(tags ...string) QueryOption {
	return func(q *queryOptions) {
		q.tags = append(q.tags, tags...)
	}
}

func WithLimit(limit int) QueryOption {
	return func(q *queryOptions) {
		if limit > 0 {
			q.limit = limit
		}
	}
}

func WithOffset(offset int) QueryOption {
	return func(q *queryOptions) {
		if offset > 0 {
			q.offset = offset
		}
	}
}

func WithSortOrder(order SortOrder) QueryOption {
	return func(q *queryOptions) { q.sortOrder = order }
}

func buildQueryOptions(opts ...QueryOption) queryOptions {
	q := queryOptions{}

	for _, opt := range opts {
		opt(&q)
	}

	return q
}

func (s *Service) GetPostBySlug(slug string, opts ...QueryOption) (*content.Post, bool) {
	opt := buildQueryOptions(opts...)
	idx := s.index.Load()

	p, ok := idx.PostsBySlug[slug]
	if !ok {
		return nil, false
	}

	if !canView(p, opt) {
		return nil, false
	}

	return p, true
}

func (s *Service) GetPosts(opts ...QueryOption) []*content.Post {
	opt := buildQueryOptions(opts...)
	idx := s.index.Load()

	matched := make([]*content.Post, 0, len(idx.PostsBySlug))
	for _, post := range idx.PostsBySlug {

		if !canView(post, opt) || !matchTags(post, opt) {
			continue
		}
		matched = append(matched, post)
	}

	// SortNone together with offset/limit => paginating over Go's randomized map iteration order
	if opt.sortOrder != SortNone {
		sort.Slice(matched, func(i, j int) bool {
			ti, tj := zeroIfNil(matched[i].PublishedAt), zeroIfNil(matched[j].PublishedAt)
			if !ti.Equal(tj) {
				if opt.sortOrder == SortOldestFirst {
					return ti.Before(tj)
				}
				return ti.After(tj)
			}
			return matched[i].Slug < matched[j].Slug
		})
	}
	if opt.offset >= len(matched) {
		return []*content.Post{}
	}
	matched = matched[opt.offset:]

	if opt.limit > 0 && opt.limit < len(matched) {
		matched = matched[:opt.limit]
	}

	return matched
}

func (s *Service) GetTags() []string {
	idx := s.index.Load()
	return idx.Tags
}

func canView(p *content.Post, opt queryOptions) bool {
	if p.PublishedAt == nil && !opt.includeUnpublished {
		return false
	}

	if opt.language != nil && p.Language != *opt.language {
		return false
	}

	return true
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

func zeroIfNil(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
