package verse

import (
	"blog/internal/ui"
	"net/http"
)

func (s *Service) handleVersesPage(w http.ResponseWriter, r *http.Request) {
	ui.Render(w, r, VersesPage(s.Get()))
}
