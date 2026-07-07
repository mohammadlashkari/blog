package cli

import (
	"blog/internal/config"
	"blog/internal/db"
	"blog/internal/rss"
	"blog/internal/rss/store"
	"fmt"

	"github.com/spf13/cobra"
)

func readingCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reading",
		Short: "Manage RSS reading",
	}

	cmd.AddCommand(readingRefreshCmd(cfg))
	cmd.AddCommand(readingListCmd())
	cmd.AddCommand(readingAddCmd())
	return cmd
}

func readingRefreshCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh followed feeds",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			db, err := db.Open(ctx, cfg.DBPath)
			if err != nil {
				return err
			}
			defer db.Close()

			svc := rss.New(
				cfg,
				store.NewRSSStore(db),
			)

			if err := svc.Refresh(ctx); err != nil {
				return fmt.Errorf("failed to refresh reading list: %w", err)
			}

			cmd.Println("reading list refreshed")
			return nil
		},
	}

	return cmd
}

func readingListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List followed feeds",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
		},
	}

	return cmd
}

func readingAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <feed-url>",
		Short: "Follow new feed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
		},
	}

	return cmd
}
