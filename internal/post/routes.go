package post

import (
	"blog/internal/content"
	"net/http"
)

func (s *PostService) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /posts", s.handlePostsIndex)
	mux.HandleFunc("GET /post/{slug}", s.handlePost)
	mux.HandleFunc("POST /webhook", s.handleWebhook)

	en, fa := content.LanguageEn, content.LanguageFa
	mux.HandleFunc("GET /feed.xml", s.handleFeed(nil)) // All
	mux.HandleFunc("GET /feed/en.xml", s.handleFeed(&en))
	mux.HandleFunc("GET /feed/fa.xml", s.handleFeed(&fa))
}
