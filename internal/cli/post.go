package cli

import (
	"blog/internal/config"
	"blog/internal/content"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func postCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post",
		Short: "Manage blog posts",
	}

	cmd.AddCommand(postAddCmd(cfg))
	cmd.AddCommand(postValidateCmd())
	return cmd
}

func postAddCmd(cfg *config.Config) *cobra.Command {
	var (
		slug        string
		lang        string
		description string
		tags        []string
		favorite    bool
		embed       string
		publish     bool
		dir         string
	)

	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Scaffold a new post",
		Long: `Scaffolds a new post directory (<posts-dir>/<slug>/index.md plus an assets/
folder) from a title. The slug is derived from the title unless --slug is given,
and the front matter is validated with the same rules the server enforces before
anything is written. Posts are drafts by default; pass --publish to set published_at.`,
		Args: cobra.ExactArgs(1),
		// Show the raw validation/IO error, not the usage block dumped after it.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]

			if slug == "" {
				slug = slugify(title)
			}
			// A non-Latin or symbol-only title slugifies to nothing; slugs must be
			// ASCII kebab-case (they name the dir and the URL), so ask for one.
			if slug == "" {
				return fmt.Errorf("could not derive a slug from title %q; pass --slug", title)
			}

			var publishedAt *time.Time
			if publish {
				now := time.Now().UTC()
				publishedAt = &now
			}

			// Gate the front matter through the server's validator (single source of
			// truth) before touching the filesystem.
			post := &content.Post{
				FrontMatter: content.FrontMatter{
					Title:       title,
					Slug:        slug,
					Description: description,
					Language:    content.Language(lang),
					IsFavorite:  favorite,
					Tags:        tags,
					PublishedAt: publishedAt,
					EmbedID:     embed,
				},
			}
			if err := content.ValidatePost(post); err != nil {
				return fmt.Errorf("invalid post: %w", err)
			}

			postsDir := dir
			if postsDir == "" {
				postsDir = filepath.Join(cfg.LocalContentPath, "posts")
			}

			// The on-disk folder name must equal the slug (see walker.go): /media/<dir>
			// and /post/<slug> stay in sync only when they match.
			postDir := filepath.Join(postsDir, slug)
			if _, err := os.Stat(postDir); err == nil {
				return fmt.Errorf("post %q already exists at %s", slug, postDir)
			} else if !os.IsNotExist(err) {
				return err
			}

			if err := os.MkdirAll(filepath.Join(postDir, "assets"), 0o755); err != nil {
				return err
			}

			body, err := renderPost(post)
			if err != nil {
				return err
			}

			indexPath := filepath.Join(postDir, cfg.ContentFilename)
			if err := os.WriteFile(indexPath, body, 0o644); err != nil {
				return err
			}

			cmd.Printf("created %s\n", indexPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&slug, "slug", "s", "", "post slug (defaults to a slugified title)")
	cmd.Flags().StringVarP(&lang, "lang", "l", string(content.LanguageEn), "post language (en or fa)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "post description")
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "post tags (repeatable or comma-separated)")
	cmd.Flags().BoolVar(&favorite, "favorite", false, "mark the post as a favorite")
	cmd.Flags().StringVar(&embed, "embed", "", "embed key (e.g. game-of-life)")
	cmd.Flags().BoolVar(&publish, "publish", false, "publish now (set published_at); drafts otherwise")
	cmd.Flags().StringVar(&dir, "dir", "", "posts directory to write into (defaults to <content>/posts)")

	return cmd
}

// renderPost marshals a validated post into an index.md byte body: the post's front matter
// (in content.FrontMatter's canonical field order) between --- fences, followed by a starter
// paragraph.
func renderPost(post *content.Post) ([]byte, error) {
	var yml bytes.Buffer
	enc := yaml.NewEncoder(&yml)
	enc.SetIndent(2)
	if err := enc.Encode(post.FrontMatter); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}

	starter := "Write your post here.\n"
	if post.Language == content.LanguageFa {
		starter = "متن نوشته را اینجا بنویسید.\n"
	}

	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(yml.Bytes())
	out.WriteString("---\n\n")
	out.WriteString(starter)

	return out.Bytes(), nil
}
