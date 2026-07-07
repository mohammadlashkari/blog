package cli

import (
	"blog/internal/config"
	"blog/internal/db"
	"blog/internal/rss"
	"blog/internal/rss/store"
	"fmt"

	"github.com/spf13/cobra"
)

func rssCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rss",
		Short: "Manage RSS reading",
	}

	cmd.AddCommand(rssSyncCmd(cfg))
	cmd.AddCommand(rssListCmd())
	cmd.AddCommand(rssAddCmd())
	return cmd
}

func rssSyncCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Fetch followed feeds",
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

			if err := svc.FetchAll(ctx); err != nil {

			}

			cmd.Println("rss feeds refreshed")
			return nil
		},
	}

	return cmd
}

func rssListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List followed feeds",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
		},
	}

	return cmd
}

func rssAddCmd() *cobra.Command {
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
