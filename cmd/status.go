package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dicastro/storyforge/internal/config"
	"github.com/dicastro/storyforge/internal/model"
	"github.com/dicastro/storyforge/internal/repository"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newStatusCmd() *cobra.Command {
	var sagaID string
	var note string

	cmd := &cobra.Command{
		Use:   "status <book-id> <new-status>",
		Short: "Update the status of a book",
		Long: `Update a book's status. Valid statuses are:

  draft      Initial working state — structure may be incomplete.
  wip        Actively being worked on — content and images in progress.
  ready      Complete and validated — ready for submission.
  published  Submitted to a distributor. Generates a versioned snapshot on next generate.

A note can be attached to record why the status changed.

Examples:
  storyforge status book-01 wip --saga lucias-adventures
  storyforge status book-01 ready --saga lucias-adventures --note "All images approved"
  storyforge status book-01 published --saga lucias-adventures`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			bookID := args[0]
			newStatus := model.BookStatus(args[1])

			validStatuses := []model.BookStatus{
				model.StatusDraft, model.StatusWIP, model.StatusReady, model.StatusPublished,
			}
			valid := false
			for _, s := range validStatuses {
				if s == newStatus {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid status %q (must be: draft|wip|ready|published)", newStatus)
			}

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

			oldStatus := book.Metadata.Status

			// Guard: downgrading from published requires explicit confirmation.
			if oldStatus == model.StatusPublished && newStatus != model.StatusPublished {
				fmt.Printf("⚠  Book %q is currently published. Downgrading status may indicate a new edition.\n", bookID)
				fmt.Print("Continue? [y/N] ")
				var answer string
				fmt.Scanln(&answer)
				if strings.ToLower(strings.TrimSpace(answer)) != "y" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			if err := updateBookStatus(book, newStatus); err != nil {
				return err
			}

			if note != "" {
				notesPath := book.RootPath + "/notes.md"
				entry := fmt.Sprintf("\n---\n\n## Status change: %s → %s (%s)\n\n%s\n",
					oldStatus, newStatus, time.Now().Format("2006-01-02"), note)
				f, err := os.OpenFile(notesPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err == nil {
					_, _ = f.WriteString(entry)
					f.Close()
				}
			}

			fmt.Printf("✓ Status updated: %s → %s\n", oldStatus, newStatus)
			return nil
		},
	}

	cmd.Flags().StringVar(&sagaID, "saga", "", "saga id that contains the book (required)")
	cmd.Flags().StringVar(&note, "note", "", "optional note to append to notes.md")
	_ = cmd.MarkFlagRequired("saga")

	return cmd
}

func updateBookStatus(book *model.Book, newStatus model.BookStatus) error {
	bookFile := book.RootPath + "/book.yaml"

	raw, err := os.ReadFile(bookFile)
	if err != nil {
		return fmt.Errorf("reading book file: %w", err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return fmt.Errorf("parsing book YAML: %w", err)
	}

	if len(root.Content) > 0 {
		setYAMLField(root.Content[0], "metadata", "status", string(newStatus))
	}

	out, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("marshalling updated YAML: %w", err)
	}

	if err := os.WriteFile(bookFile, out, 0644); err != nil {
		return fmt.Errorf("writing book file: %w", err)
	}
	return nil
}

func setYAMLField(node *yaml.Node, parent, child, value string) {
	if node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		val := node.Content[i+1]
		if key.Value == parent && val.Kind == yaml.MappingNode {
			for j := 0; j+1 < len(val.Content); j += 2 {
				if val.Content[j].Value == child {
					val.Content[j+1].Value = value
					return
				}
			}
		}
	}
}