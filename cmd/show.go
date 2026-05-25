package cmd

import (
	"fmt"
	"strings"

	"github.com/dicastro/storyforge/internal/config"
	"github.com/dicastro/storyforge/internal/repository"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var sagaID string

	cmd := &cobra.Command{
		Use:   "show <book-id>",
		Short: "Show detailed information about a book",
		Long: `Show prints a structured summary of a book: metadata, characters,
publication targets, spread count, image asset status, and typography settings.

Examples:
  storyforge show book-01 --saga lucias-adventures`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bookID := args[0]

			if sagaID == "" {
				return fmt.Errorf("--saga is required")
			}

			cfg, err := config.Load(contentRoot)
			if err != nil {
				return err
			}
			store := repository.NewStore(cfg.ContentRoot)
			book, err := store.LoadBook(sagaID, bookID)
			if err != nil {
				return err
			}

			sep := strings.Repeat("─", 60)
			fmt.Println(sep)
			fmt.Printf("  %s\n", book.Metadata.Title.Get(book.Metadata.Language))
			fmt.Println(sep)

			fmt.Printf("  ID:       %s\n", book.ID)
			fmt.Printf("  Author:   %s\n", book.Metadata.Author)
			fmt.Printf("  Language: %s\n", book.Metadata.Language)
			fmt.Printf("  Status:   %s %s\n", statusIcon(book.Metadata.Status), book.Metadata.Status)
			if book.SagaRef != nil {
				fmt.Printf("  Saga:     %s (book %d)\n", book.SagaRef.SagaID, book.SagaRef.BookNumber)
			}
			fmt.Printf("  Format:   %s, %d interior pages\n", book.Format.Size, book.Format.PagesInterior)

			if len(book.Metadata.Tags) > 0 {
				fmt.Printf("  Tags:     %s\n", strings.Join(book.Metadata.Tags, ", "))
			}

			// Typography
			if book.Font != nil {
				fmt.Printf("\n  Book font: %s %.1fpt %s\n",
					book.Font.Family, book.Font.SizePt, book.Font.Color)
			}

			// Publication targets
			if len(book.PublicationTargets) > 0 {
				fmt.Printf("\n  Publication targets (%d):\n", len(book.PublicationTargets))
				for _, t := range book.PublicationTargets {
					fmt.Printf("    • %-12s  %-10s  %-4s  %-10s  %-10s  %s\n",
						t.Distributor, t.Size, t.Language, t.Binding, t.Paper, t.InteriorColor)
				}
			} else {
				fmt.Println("\n  Publication targets: (none defined)")
			}

			fmt.Printf("\n  Characters (%d):\n", len(book.ResolvedCharacters))
			for _, c := range book.ResolvedCharacters {
				hasPrompt := "✗ no visual prompt"
				if c.VisualPrompt != "" {
					hasPrompt = "✓ visual prompt ok"
				}
				fmt.Printf("    • %s (%s) — %s — %s\n", c.Name, c.ID, c.Role, hasPrompt)
			}

			fmt.Printf("\n  Spreads (%d):\n", len(book.Spreads))
			for _, s := range book.Spreads {
				hasText := s.Left.Text.Get("es") != "" || s.Left.Text.Get("en") != ""
				hasImg := s.Right.ImagePath != ""
				hasPrompt := s.Right.ImagePrompt != ""
				hasFont := s.Left.Font != nil || s.Right.Font != nil
				hasBox := s.Left.TextBox != nil

				textMark := "✗"
				if hasText {
					textMark = "✓"
				}
				imgMark := "✗"
				if hasImg {
					imgMark = "✓"
				} else if hasPrompt {
					imgMark = "⚠ prompt only"
				}
				extra := ""
				if hasFont {
					extra += " [font override]"
				}
				if hasBox {
					extra += " [text box]"
				}
				fmt.Printf("    [%2d]  text: %s  image: %s%s\n", s.Number, textMark, imgMark, extra)
			}

			missing := book.MissingImages()
			if len(missing) > 0 {
				fmt.Printf("\n  Missing image files (%d):\n", len(missing))
				for _, m := range missing {
					fmt.Printf("    ○ %s\n", m)
				}
			}

			fmt.Println(sep)
			return nil
		},
	}

	cmd.Flags().StringVar(&sagaID, "saga", "", "saga id that contains the book (required)")
	_ = cmd.MarkFlagRequired("saga")

	return cmd
}