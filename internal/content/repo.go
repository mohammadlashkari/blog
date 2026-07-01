package content

import (
	"blog/internal/config"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

type ContentRepo struct {
	LocalPath string
	RemoteURL string
	Branch    string
}

func New(cfg *config.Config) *ContentRepo {
	return &ContentRepo{
		LocalPath: cfg.LocalContentRepo,
		RemoteURL: cfg.RemoteContentRepo,
		Branch:    "master",
	}
}

func (r ContentRepo) EnsureUpdated(ctx context.Context) error {
	if _, err := os.ReadDir(r.LocalPath); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			slog.ErrorContext(ctx, "failed to read content repo dir", "error", err)
			return err
		}

		cmd := exec.CommandContext(ctx, "git", "clone", r.RemoteURL, r.LocalPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			slog.ErrorContext(ctx, "git clone failed", "error", err, "output", string(output))
			return err
		}
		slog.InfoContext(ctx, "git clone succeeded")
		return nil
	}

	cmd := exec.CommandContext(ctx, "git", "pull", "origin", "master")
	cmd.Dir = r.LocalPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.ErrorContext(ctx, "git pull failed", "error", err, "output", string(output))
		return err
	}
	if strings.Contains(string(output), "Already up to date") {
		slog.InfoContext(ctx, "git pull succeeded already up to date")
	} else {
		slog.InfoContext(ctx, "git pull succeeded")
	}

	return nil
}
