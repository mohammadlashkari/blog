package auth

import "net/http"

func (s *AuthService) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /login", s.handleAdminLogin)
}
