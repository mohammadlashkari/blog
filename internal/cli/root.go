package cli

import (
	"blog/internal/config"

	"github.com/spf13/cobra"
)

func Execute() error {
	cfg, err := config.Dev()
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

	root.AddCommand(postCmd())
	root.AddCommand(readingCmd(cfg))
	return root
}
