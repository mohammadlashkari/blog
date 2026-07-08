package post

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"blog/internal/config"
)

func TestHandleMediaGuard(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "posts", "go-tips", "assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "x.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A sibling private file that must never be served.
	if err := os.WriteFile(filepath.Join(root, "posts", "go-tips", "index.md"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &PostService{cfg: &config.Config{LocalContentPath: root}}

	cases := []struct {
		name, dir, path string
		wantCode        int
	}{
		{"asset ok", "go-tips", "assets/x.png", 200},
		{"index.md blocked", "go-tips", "index.md", 404},
		{"traversal blocked", "go-tips", "assets/../index.md", 404},
		{"escape root blocked", "go-tips", "../../../etc/passwd", 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/media/"+tc.dir+"/"+tc.path, nil)
			r.SetPathValue("dir", tc.dir)
			r.SetPathValue("path", tc.path)
			w := httptest.NewRecorder()
			s.handleMedia(w, r)
			if w.Code != tc.wantCode {
				t.Errorf("code = %d, want %d", w.Code, tc.wantCode)
			}
		})
	}
}
