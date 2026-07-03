package main

import (
	"crypto/subtle"
	"net/http"

	"blog/internal/config"
	"blog/internal/ui"
)

const (
	adminCookie       = "admin_cookie"
	passwordFormField = "password"
)

func HandleAdminLogin(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad Request: Unable to parse form", http.StatusBadRequest)
			return
		}

		inputPassword := r.FormValue(passwordFormField)
		if inputPassword == "" {
			http.Error(w, "Unauthorized: Missing password", http.StatusUnauthorized)
			return
		}

		if subtle.ConstantTimeCompare([]byte(inputPassword), []byte(cfg.AdminToken)) != 1 {
			http.Error(w, "Unauthorized: Invalid password", http.StatusUnauthorized)
			return
		}

		cookie := http.Cookie{
			Name:     adminCookie,
			Value:    cfg.AdminToken,
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
}

func handleAbout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AboutPage().Render(r.Context(), w); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok\n"))
}
