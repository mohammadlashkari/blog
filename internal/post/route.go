package post

import "net/http"

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("GET /healthz", healthz)

}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok\n"))
}
