package verse

import "net/http"

func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /verses", s.handleVersesPage)
}
