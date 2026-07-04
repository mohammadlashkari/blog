package cli

import (
	"blog/internal/content"
	"fmt"

	"github.com/spf13/cobra"
)

func validateCmd() *cobra.Command {
	var filename string

	cmd := &cobra.Command{
		Use:   "validate <posts-dir>",
		Short: "Validate every post in a directory",
		Long: `Walks <posts-dir>, checking each post's front matter (title, slug,
language) and enforcing cross-post rules such as unique slugs. Exits
non-zero on the first violation, so it can gate content-repo CI.`,
		Args: cobra.ExactArgs(1),
		// CI wants the raw validation error, not the usage block dumped after it.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]

			c := content.New(dir, "", "", filename)
			if err := c.ValidatePosts(dir); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			cmd.Println("all posts valid")
			return nil
		},
	}

	cmd.Flags().StringVarP(&filename, "filename", "f", "index.md", "content filename to look for in each post directory")

	return cmd
}
