package cli

import (
	"blog/internal/config"

	"github.com/spf13/cobra"
)

func Execute() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	return rootCmd(cfg).Execute()
}

func rootCmd(cfg *config.Config) *cobra.Command {
	root := &cobra.Command{
		Use:   "blog",
		Short: "My blog CLI",
	}

	root.AddCommand(postCmd(cfg))
	root.AddCommand(readingCmd(cfg))
	return root
}
