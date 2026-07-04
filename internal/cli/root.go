package cli

import "github.com/spf13/cobra"

func Execute() error {
	return rootCmd().Execute()
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "blog",
		Short: "My blog CLI",
	}

	root.AddCommand(validateCmd())
	return root
}
