package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func postCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post",
		Short: "Manage blog posts",
	}

	cmd.AddCommand(postAddCmd())
	cmd.AddCommand(postValidateCmd())
	return cmd
}

func postAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Scaffold a new post",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
		},
	}

	return cmd
}
