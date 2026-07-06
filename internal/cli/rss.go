package cli

import (
	"blog/internal/config"
	"blog/internal/db"
	"blog/internal/rss"
	"blog/internal/rss/store"

	"github.com/spf13/cobra"
)

func rssCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "refresh rss reading",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cfg, err := config.Load()
			if err != nil {
				return err
			}

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
