package verse

import (
	"blog/internal/config"
	"blog/internal/content"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// newTestService returns a service rooted at a temp dir. When body is non-empty
// it is written to <root>/verses/verses.yaml.
func newTestService(t *testing.T, body string) (*Service, string) {
	t.Helper()

	root := t.TempDir()
	path := filepath.Join(root, "verses", "verses.yaml")

	if body != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	return New(&config.Config{LocalContentPath: root}), path
}

const validYAML = `
verses:
  - text: |
      بنی‌آدم اعضای یک پیکرند
      که در آفرینش ز یک گوهرند
    author: سعدی
    language: fa
  - text: |
      The only way out is through.
    author: Robert Frost
    language: en
  - text: Stay hungry.
    author: Anonymous
`

func TestLoadValid(t *testing.T) {
	s, _ := newTestService(t, validYAML)

	if err := s.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := s.Get()
	if len(got) != 3 {
		t.Fatalf("got %d verses, want 3", len(got))
	}

	// File order is display order.
	if got[0].Author != "سعدی" || got[1].Author != "Robert Frost" {
		t.Errorf("order not preserved: %q, %q", got[0].Author, got[1].Author)
	}

	// Block scalars keep interior newlines but lose the trailing one.
	wantFa := "بنی‌آدم اعضای یک پیکرند\nکه در آفرینش ز یک گوهرند"
	if got[0].Text != wantFa {
		t.Errorf("fa text = %q, want %q", got[0].Text, wantFa)
	}
	if got[1].Text != "The only way out is through." {
		t.Errorf("en text = %q", got[1].Text)
	}

	if got[0].Language != content.LanguageFa {
		t.Errorf("got[0].Language = %q, want fa", got[0].Language)
	}
	// Omitted language defaults to English.
	if got[2].Language != content.LanguageEn {
		t.Errorf("got[2].Language = %q, want en", got[2].Language)
	}
}

func TestLoadMissingFile(t *testing.T) {
	s, _ := newTestService(t, "")

	if err := s.Load(context.Background()); err != nil {
		t.Fatalf("Load with no file should not error, got %v", err)
	}
	if got := s.Get(); len(got) != 0 {
		t.Errorf("got %d verses, want 0", len(got))
	}
}

func TestLoadInvalidKeepsPreviousVerses(t *testing.T) {
	tests := map[string]string{
		"malformed yaml": "verses:\n  - text: [unclosed\n",
		"empty text":     "verses:\n  - text: \"  \"\n    author: Someone\n",
		"empty author":   "verses:\n  - text: Something\n    author: \"\"\n",
		"bad language":   "verses:\n  - text: Something\n    author: Someone\n    language: de\n",
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			s, path := newTestService(t, validYAML)
			ctx := context.Background()

			if err := s.Load(ctx); err != nil {
				t.Fatalf("initial Load: %v", err)
			}

			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}

			if err := s.Load(ctx); err == nil {
				t.Fatal("expected an error, got nil")
			}
			// A bad reload must not wipe good data.
			if got := s.Get(); len(got) != 3 {
				t.Errorf("got %d verses after failed reload, want the previous 3", len(got))
			}
		})
	}
}

func TestSyncSkipsUnchangedFile(t *testing.T) {
	s, path := newTestService(t, validYAML)
	ctx := context.Background()

	if err := s.Load(ctx); err != nil {
		t.Fatalf("Load: %v", err)
	}
	before := s.verses.Load()

	if err := s.Sync(ctx); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if s.verses.Load() != before {
		t.Error("unchanged file was reparsed")
	}

	body := "verses:\n  - text: Something new.\n    author: Someone\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := s.Sync(ctx); err != nil {
		t.Fatalf("Sync after edit: %v", err)
	}
	if got := s.Get(); len(got) != 1 || got[0].Text != "Something new." {
		t.Errorf("changed file was not reparsed: %+v", got)
	}
}

func TestGetBeforeLoad(t *testing.T) {
	s, _ := newTestService(t, validYAML)

	if got := s.Get(); got != nil {
		t.Errorf("Get before Load = %+v, want nil", got)
	}
}
