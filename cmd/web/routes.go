package main

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

type IsAdminRequest struct {
	Password string `json:"password"`
}

func handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var payload IsAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {

	}

	if subtle.ConstantTimeCompare([]byte(payload.Password), []byte("password")) == 0 {

	}

}

func handleAbout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("about\n"))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok\n"))
}
