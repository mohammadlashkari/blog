package rss

import "net/http"

func (rs *RSSService) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /reading-list", rs.handleReadingListPage)
}
