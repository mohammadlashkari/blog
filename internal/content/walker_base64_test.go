package content

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 1x1 transparent PNG — a real, mimetype-detectable image so img64 will inline it.
var onePixelPNG, _ = base64.StdEncoding.DecodeString(
	"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")

func writeB64Post(t *testing.T, root, slug, body string) {
	t.Helper()
	dir := filepath.Join(root, "posts", slug)
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "x.png"), onePixelPNG, 0o644); err != nil {
		t.Fatal(err)
	}
	md := fmt.Sprintf("---\ntitle: %s\nslug: %s\nlanguage: en\n---\n%s", slug, slug, body)
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Run with -race: many posts share the single c.md across the 20-worker pool while each
// carries its own directory via parser context. Confirms the shared parser is race-free
// and that relative images are base64-inlined (not left as relative paths).
func TestFsPostsSharedParserBase64(t *testing.T) {
	root := t.TempDir()
	const n = 40
	for i := 0; i < n; i++ {
		writeB64Post(t, root, fmt.Sprintf("post-%02d", i), "![pic](assets/x.png)\n")
	}

	c := New(root, "", "master", "index.md")
	posts, err := c.fsPosts(filepath.Join(root, "posts"))
	if err != nil {
		t.Fatalf("fsPosts: %v", err)
	}
	if len(posts) != n {
		t.Fatalf("got %d posts, want %d", len(posts), n)
	}
	for slug, p := range posts {
		if !strings.Contains(p.HTML, "data:image/png;base64,") {
			t.Errorf("post %s: image not base64-inlined\n%s", slug, p.HTML)
		}
		if strings.Contains(p.HTML, "assets/x.png") {
			t.Errorf("post %s: relative path leaked into HTML", slug)
		}
	}
}

func TestFsPostsBase64LeavesRemote(t *testing.T) {
	root := t.TempDir()
	writeB64Post(t, root, "remote", "![r](https://ex.com/r.png)\n")

	c := New(root, "", "master", "index.md")
	posts, err := c.fsPosts(filepath.Join(root, "posts"))
	if err != nil {
		t.Fatalf("fsPosts: %v", err)
	}
	if !strings.Contains(posts["remote"].HTML, "https://ex.com/r.png") {
		t.Errorf("remote image should stay a link: %s", posts["remote"].HTML)
	}
	if strings.Contains(posts["remote"].HTML, "data:") {
		t.Errorf("remote image should not be inlined: %s", posts["remote"].HTML)
	}
}
