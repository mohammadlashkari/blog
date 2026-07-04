package auth

import (
	"crypto/subtle"
	"net/http"
)

const (
	CookieName        = "auth_cookie"
	passwordFormField = "password"
)

func (s *AuthService) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request: Unable to parse form", http.StatusBadRequest)
		return
	}

	inputPassword := r.FormValue(passwordFormField)
	if inputPassword == "" {
		http.Error(w, "Unauthorized: Missing password", http.StatusUnauthorized)
		return
	}

	if subtle.ConstantTimeCompare([]byte(inputPassword), []byte(s.cfg.AdminToken)) != 1 {
		http.Error(w, "Unauthorized: Invalid password", http.StatusUnauthorized)
		return
	}

	cookie := http.Cookie{
		Name:     CookieName,
		Value:    s.cfg.AdminToken,
		Path:     "/",
		MaxAge:   3600, // 1 hour
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Login successful"))
}
