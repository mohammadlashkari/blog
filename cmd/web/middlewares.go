package main

import (
	"blog/internal/auth"
	"blog/internal/config"
	"crypto/rand"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/a-h/templ"
	"golang.org/x/time/rate"
)

func chainMiddlewars(
	h http.Handler, mws ...func(h http.Handler) http.Handler,
) http.Handler {
	for _, mw := range slices.Backward(mws) {
		h = mw(h)
	}

	return h
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")

				slog.ErrorContext(r.Context(), "panic recovered", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ip     = r.RemoteAddr
			agent  = r.UserAgent()
			method = r.Method
			url    = r.URL.RequestURI()
			ctx    = r.Context()
		)

		userAttrs := slog.Group("user", "ip", ip, "agent", agent)
		slog.InfoContext(ctx, "request received", userAttrs, "method", method, "url", url)

		next.ServeHTTP(w, r)
	})
}

func rateLimit(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !cfg.LimiterEnable {
			return next
		}

		type client struct {
			lastSeen time.Time
			limiter  *rate.Limiter
		}

		var (
			mu      sync.Mutex
			clients = make(map[string]*client)
		)

		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()

			for range ticker.C {
				mu.Lock()
				for ip, c := range clients {
					if time.Since(c.lastSeen) > 3*time.Minute {
						delete(clients, ip)
					}
				}
				mu.Unlock()
			}
		}()

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			mu.Lock()

			if _, ok := clients[ip]; !ok {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(cfg.LimiterRPS), cfg.LimiterBurst),
				}
			}

			clients[ip].lastSeen = time.Now().UTC()

			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

func isAdmin(authSvc *auth.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			isAdmin := authSvc.IsSessionValid(ctx, r) || authSvc.IsBasicAuthValid(ctx, r)

			ctx = auth.WithAdmin(ctx, isAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The nonce covers templ script templates, which read it from the
		// context. WebAssembly.instantiateStreaming needs 'wasm-unsafe-eval'.
		nonce := rand.Text()

		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'wasm-unsafe-eval' 'nonce-"+nonce+"'; "+
				"style-src 'self'; "+
				"font-src 'self'; "+
				"img-src 'self' https: data:; "+
				"object-src 'none'; "+
				"base-uri 'none'; "+
				"form-action 'self'; "+
				"frame-ancestors 'none'",
		)
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")

		next.ServeHTTP(w, r.WithContext(templ.WithNonce(r.Context(), nonce)))
	})
}
