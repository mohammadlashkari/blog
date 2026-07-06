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
	mux.HandleFunc("GET /rss.xml", s.handleRSS(nil)) // All
	mux.HandleFunc("GET /rss/en.xml", s.handleRSS(&en))
	mux.HandleFunc("GET /rss/fa.xml", s.handleRSS(&fa))
}
