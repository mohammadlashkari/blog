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

func (s *Service) GetPostBySlug(slug string) (*content.Post, bool) {
	idx := s.index.Load()
	p, ok := idx.PostsBySlug[slug]
	return p, ok
}

func (s *Service) GetFAPosts() []*content.Post {
	idx := s.index.Load()
	return idx.PostsByLanguage[content.LanguageFa]
}

func (s *Service) GetENPosts() []*content.Post {
	idx := s.index.Load()
	return idx.PostsByLanguage[content.LanguageEn]
}

func (s *Service) GetPosts() []*content.Post {
	idx := s.index.Load()
	return idx.Posts
}

func (s *Service) GetTags() []string {
	idx := s.index.Load()
	return idx.Tags
}

func (s *Service) GetPostsWithTags(tags []string) []*content.Post {
	idx := s.index.Load()

	posts := make([]*content.Post, 0)
	seen := make(map[string]struct{})

	for _, tag := range tags {
		for _, post := range idx.PostsByTag[tag] {
			if _, ok := seen[post.Slug]; ok {
				continue
			}

			seen[post.Slug] = struct{}{}
			posts = append(posts, post)
		}
	}

	return posts
}
