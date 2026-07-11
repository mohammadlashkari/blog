package rss

import (
	"blog/internal/auth"
	"net/http"
)

func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /reading", s.handleReadingPage)

	refreshHandler := http.HandlerFunc(s.handleRefreshReading)
	mux.Handle("GET /reading/refresh", auth.RequireAdmin(refreshHandler))

	mux.Handle("POST /reading/{id}/status", auth.RequireAdmin(http.HandlerFunc(s.handleSetStatus)))
	mux.Handle("POST /reading/{id}/save", auth.RequireAdmin(http.HandlerFunc(s.handleToggleSaved)))
}
