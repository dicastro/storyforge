package cmd

import (
	"fmt"
	"strings"

	"github.com/dicastro/storyforge/internal/config"
	"github.com/dicastro/storyforge/internal/model"
	"github.com/dicastro/storyforge/internal/repository"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var filterStatus string
	var filterSaga string

	cmd := &cobra.Command{
		Use:   "list [books|sagas|characters]",
		Short: "List content (books, sagas, or characters)",
		Long: `List books, sagas, or characters stored in the content directory.

Examples:
  storyforge list books
  storyforge list books --status draft
  storyforge list books --saga lucas-adventures
  storyforge list sagas
  storyforge list characters --saga lucas-adventures`,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"books", "sagas", "characters"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(contentRoot)
			if err != nil {
				return err
			}
			store := repository.NewStore(cfg.ContentRoot)

			target := "books"
			if len(args) > 0 {
				target = args[0]
			}

			switch target {
			case "books":
				return listBooks(store, filterStatus, filterSaga)
			case "sagas":
				return listSagas(store)
			case "characters":
				return listCharacters(store, filterSaga)
			default:
				return fmt.Errorf("unknown target %q: use books, sagas, or characters", target)
			}
		},
	}

	cmd.Flags().StringVar(&filterStatus, "status", "", "filter books by status (draft|wip|ready|published)")
	cmd.Flags().StringVar(&filterSaga, "saga", "", "filter by saga id")

	return cmd
}

func listBooks(store *repository.Store, filterStatus, filterSaga string) error {
	books, err := store.ListAllBooks()
	if err != nil {
		return err
	}

	fmt.Printf("%-20s %-12s %-12s %-40s\n", "BOOK ID", "SAGA", "STATUS", "TITLE")
	fmt.Println(strings.Repeat("─", 90))

	count := 0
	for _, book := range books {
		if filterStatus != "" && string(book.Metadata.Status) != filterStatus {
			continue
		}
		sagaID := ""
		if book.SagaRef != nil {
			sagaID = book.SagaRef.SagaID
		}
		if filterSaga != "" && sagaID != filterSaga {
			continue
		}
		title := book.Metadata.Title.Get(book.Metadata.Language)
		fmt.Printf("%-20s %-12s %-12s %-40s\n",
			book.ID,
			truncate(sagaID, 12),
			statusIcon(book.Metadata.Status)+" "+string(book.Metadata.Status),
			truncate(title, 40),
		)
		count++
	}

	if count == 0 {
		fmt.Println("  (no books found)")
	}
	return nil
}

func listSagas(store *repository.Store) error {
	sagaIDs, err := store.ListSagaIDs()
	if err != nil {
		return err
	}

	fmt.Printf("%-20s %-10s %-40s\n", "SAGA ID", "BOOKS", "TITLE")
	fmt.Println(strings.Repeat("─", 75))

	for _, id := range sagaIDs {
		saga, err := store.LoadSaga(id)
		if err != nil {
			fmt.Printf("%-20s  (error: %v)\n", id, err)
			continue
		}
		bookIDs, _ := store.ListBookIDsForSaga(id)
		fmt.Printf("%-20s %-10d %-40s\n",
			id,
			len(bookIDs),
			truncate(saga.Title.Get("es"), 40),
		)
	}

	if len(sagaIDs) == 0 {
		fmt.Println("  (no sagas found)")
	}
	return nil
}

func listCharacters(store *repository.Store, filterSaga string) error {
	sagaIDs, err := store.ListSagaIDs()
	if err != nil {
		return err
	}

	fmt.Printf("%-15s %-20s %-8s %-10s %-30s\n", "SAGA", "ID", "AGE", "ROLE", "NAME")
	fmt.Println(strings.Repeat("─", 90))

	for _, sagaID := range sagaIDs {
		if filterSaga != "" && sagaID != filterSaga {
			continue
		}
		saga, err := store.LoadSaga(sagaID)
		if err != nil {
			continue
		}
		for _, c := range saga.ResolvedCharacters {
			fmt.Printf("%-15s %-20s %-8d %-10s %-30s\n",
				truncate(sagaID, 15),
				truncate(c.ID, 20),
				c.Age,
				truncate(c.Role, 10),
				c.Name,
			)
		}
	}
	return nil
}

// ---- helpers -----------------------------------------------------------------

func statusIcon(s model.BookStatus) string {
	switch s {
	case model.StatusDraft:
		return "○"
	case model.StatusWIP:
		return "◑"
	case model.StatusReady:
		return "●"
	case model.StatusPublished:
		return "✓"
	default:
		return "?"
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}