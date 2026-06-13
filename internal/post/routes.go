package post

import "net/http"

func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /posts", s.handlePostsIndex)
	mux.HandleFunc("GET /post/{slug}", s.handlePost)
	mux.HandleFunc("GET /about", handleAbout)
	mux.HandleFunc("GET /healthz", healthz)
}
