package content

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

func (c *Content) reconcile(ctx context.Context) error {
	info, err := os.Stat(c.localPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			slog.ErrorContext(ctx, "failed to read content repo dir", "error", err)
			return err
		}

		if err := c.clone(ctx); err != nil {
			return err
		}

		return nil
	}

	// Guard against the path being a file instead of a directory
	if !info.IsDir() {
		slog.ErrorContext(ctx, "content path exists but is not a directory", "path", c.localPath)
		return fmt.Errorf("local path %s is a file, expected directory", c.localPath)
	}

	if err := c.sync(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Content) clone(ctx context.Context) error {
	cmd := exec.CommandContext(
		ctx, "git", "clone", c.remotePath, c.localPath, "--branch", c.branch, "--depth", "1",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		slog.ErrorContext(ctx, "git clone failed", "error", err, "output", string(output))
		return fmt.Errorf("content clone failed: %w", err)
	}
	slog.InfoContext(ctx, "git clone succeeded")

	return nil
}

func (c *Content) sync(ctx context.Context) error {
	preResetHash, err := c.commitHash(ctx, "HEAD")
	if err != nil {
		slog.WarnContext(ctx, "could not resolve local HEAD hash before pull", "error", err)
	}

	// Fetch the latest changes from the remote tracking branch
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin", c.branch)
	fetchCmd.Dir = c.localPath
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		slog.ErrorContext(ctx, "git fetch failed", "error", err, "output", string(output))
		return fmt.Errorf("content fetch failed: %w", err)
	}

	// Forcefully reset local branch to match origin/<branch> exactly
	targetRemoteBranch := fmt.Sprintf("origin/%s", c.branch)
	resetCmd := exec.CommandContext(ctx, "git", "reset", "--hard", targetRemoteBranch)
	resetCmd.Dir = c.localPath
	output, err := resetCmd.CombinedOutput()
	if err != nil {
		slog.ErrorContext(ctx, "git reset --hard failed", "error", err, "output", string(output))
		return fmt.Errorf("content reset failed: %w", err)
	}

	postResetHash, err := c.commitHash(ctx, "HEAD")
	if err != nil {
		slog.ErrorContext(ctx, "could not resolve local HEAD hash after pull", "error", err)
		return fmt.Errorf("failed to verify post-reset hash: %w", err)
	}

	// Compare hashes to check if anything actually updated
	if preResetHash != "" && preResetHash == postResetHash {
		slog.InfoContext(ctx, "git pull completed: already up to date")
	} else {
		slog.InfoContext(
			ctx,
			"git repository forcefully updated to match remote source of truth",
			"from", preResetHash, "to", postResetHash,
		)
	}

	return nil
}

func (c *Content) commitHash(ctx context.Context, ref string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", ref)
	cmd.Dir = c.localPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
