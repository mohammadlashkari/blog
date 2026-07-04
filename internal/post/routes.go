package post

import "net/http"

func (s *PostService) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /posts", s.handlePostsIndex)
	mux.HandleFunc("GET /post/{slug}", s.handlePost)
	mux.HandleFunc("POST /webhook", s.handleWebhook)
}
