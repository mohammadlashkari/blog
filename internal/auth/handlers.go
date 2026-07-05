package auth

import (
	"blog/internal/ui"
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/gob"
	"log/slog"
	"net/http"
	"time"
)

const (
	passwordFormField = "password"
	sessionCookie     = "blog_session"
	sessionTTL        = time.Hour * 48
)

type Session struct {
	Version int   `json:"v"`
	Exp     int64 `json:"exp"`
}

func (s *AuthService) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	loginFailed := r.URL.Query().Get("error") == "1"
	ui.Render(w, r, LoginPage(loginFailed))
}

func (s *AuthService) handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request: Unable to parse form", http.StatusBadRequest)
		return
	}

	inputPassword := r.FormValue(passwordFormField)
	if inputPassword == "" {
		slog.WarnContext(ctx, "login attempt with empty password", "ip", r.RemoteAddr)
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}

	if subtle.ConstantTimeCompare([]byte(inputPassword), []byte(s.cfg.AdminToken)) != 1 {
		slog.WarnContext(ctx, "failed login attempt", "ip", r.RemoteAddr)
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}

	session := Session{
		Version: s.cfg.TokenVersion,
		Exp:     time.Now().Add(sessionTTL).Unix(),
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&session); err != nil {
		slog.ErrorContext(ctx, "failed to encode session", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	cookie := http.Cookie{
		Name:     sessionCookie,
		Value:    buf.String(),
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	if err := writeEncrypted(w, cookie, s.cfg.CookieSecret); err != nil {
		slog.ErrorContext(ctx, "failed to write cookie", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/posts", http.StatusSeeOther)
}

func (s *AuthService) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *AuthService) IsBasicAuthValid(ctx context.Context, r *http.Request) bool {
	if user, pass, ok := r.BasicAuth(); ok {
		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(s.cfg.AdminUser)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(s.cfg.AdminToken)) == 1
		if userMatch && passMatch {
			return true
		}
	}

	return false
}
