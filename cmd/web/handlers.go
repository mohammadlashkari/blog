package main

import (
	"net/http"

	"blog/internal/ui"
)

func handleAbout(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, ui.AboutPage())
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok\n"))
}
