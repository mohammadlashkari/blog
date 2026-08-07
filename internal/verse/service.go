package verse

import (
	"blog/internal/config"
	"blog/internal/content"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

type Service struct {
	cfg    *config.Config
	verses atomic.Pointer[[]Verse]

	// mu guards lastHash. Reads of verses stay lock-free via the atomic pointer.
	mu       sync.Mutex
	lastHash [md5.Size]byte
}

func New(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

// Load parses the verses file unconditionally. Called once at boot, after the
// content repo has been cloned.
func (s *Service) Load(ctx context.Context) error {
	return s.update(ctx, false)
}

// Sync is triggered by the content webhook. It skips reparsing when the verses
// file hasn't changed.
func (s *Service) Sync(ctx context.Context) error {
	return s.update(ctx, true)
}

// Get returns the current verses. The slice is shared, never mutate it.
func (s *Service) Get() []Verse {
	v := s.verses.Load()
	if v == nil {
		return nil
	}
	return *v
}

func (s *Service) update(ctx context.Context, checkHash bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.versesPath()

	data, err := os.ReadFile(path)
	if err != nil {
		// A content repo without a verses/ directory is a valid state: serve an
		// empty page rather than failing.
		if errors.Is(err, fs.ErrNotExist) {
			slog.WarnContext(ctx, "no verses file found, serving none", "path", path)
			s.verses.Store(&[]Verse{})
			s.lastHash = [md5.Size]byte{}
			return nil
		}
		return fmt.Errorf("failed to read verses file: %w", err)
	}

	newHash := md5.Sum(data)
	if checkHash && s.lastHash == newHash {
		slog.InfoContext(ctx, "verses file unchanged, skipping reload")
		return nil
	}

	var f versesFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("failed to unmarshal verses yaml: %w", err)
	}

	verses, err := normalize(f.Verses)
	if err != nil {
		// Leave lastHash untouched so a later Sync retries this same file, and
		// keep serving whatever was loaded before.
		return err
	}

	s.verses.Store(&verses)
	s.lastHash = newHash
	slog.InfoContext(ctx, "verses loaded", "count", len(verses))

	return nil
}

// normalize validates every verse and trims the trailing newline that YAML block
// scalars carry. It returns a new slice so a failed run can't leave the caller
// with partially rewritten data.
func normalize(verses []Verse) ([]Verse, error) {
	out := make([]Verse, 0, len(verses))

	for i, v := range verses {
		v.Text = strings.TrimRight(v.Text, "\n")
		if strings.TrimSpace(v.Text) == "" {
			return nil, fmt.Errorf("verse %d: text is empty", i)
		}
		if strings.TrimSpace(v.Author) == "" {
			return nil, fmt.Errorf("verse %d: author is empty", i)
		}

		switch v.Language {
		case "":
			v.Language = content.LanguageEn
		case content.LanguageEn, content.LanguageFa:
		default:
			return nil, fmt.Errorf("verse %d: invalid language %q", i, v.Language)
		}

		out = append(out, v)
	}

	return out, nil
}

func (s *Service) versesPath() string {
	return filepath.Join(s.cfg.LocalContentPath, "verses", "verses.yaml")
}
