package auth

import "net/http"

func (s *AuthService) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", s.handleLoginPage)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("GET /logout", s.handleLogout)
}
