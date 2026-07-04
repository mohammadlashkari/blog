package post

import (
	"blog/internal/config"
	"blog/internal/content"
	"context"
	"sync/atomic"
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

func (s *Service) GetPostBySlug(slug string, opts ...QueryOption) (*content.Post, bool) {
	opt := buildQueryOptions(opts...)
	idx := s.index.Load()

	post, ok := idx.BySlug(slug)
	if !ok || !canView(post, opt) {
		return nil, false
	}
	if opt.language != nil && post.Language != *opt.language {
		return nil, false
	}

	return post, true
}

func (s *Service) GetPosts(opts ...QueryOption) []*content.Post {
	opt := buildQueryOptions(opts...)
	idx := s.index.Load()

	// Narrow the base set through a prebuilt index when possible; both All and
	// ByLanguage come back presorted newest-first.
	base := idx.All()
	if opt.language != nil {
		base = idx.ByLanguage(*opt.language)
	}

	// Copy matches into a fresh slice: base shares backing storage with the
	// immutable snapshot and must not be mutated (offset/limit re-slice below).
	matched := make([]*content.Post, 0, len(base))
	for _, post := range base {
		if !canView(post, opt) || !matchTags(post, opt) {
			continue
		}
		matched = append(matched, post)
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
	return s.index.Load().Tags()
}

func canView(p *content.Post, opt queryOptions) bool {
	return p.PublishedAt != nil || opt.includeUnpublished
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
