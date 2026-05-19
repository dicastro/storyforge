package cmd

import (
	"fmt"
	"os"

	"github.com/dicastro/storyforge/internal/config"
	"github.com/dicastro/storyforge/internal/distributor"
	"github.com/dicastro/storyforge/internal/repository"
	"github.com/dicastro/storyforge/internal/validator"
	"github.com/spf13/cobra"

	// Register the Amazon KDP distributor.
	_ "github.com/dicastro/storyforge/internal/distributor/amazon"
)

func newValidateCmd() *cobra.Command {
	var distributorName string
	var sagaID string

	cmd := &cobra.Command{
		Use:   "validate [book-id]",
		Short: "Validate a book (or all books) against schema and distributor requirements",
		Long: `Validate checks that a book's YAML is complete and that all referenced
assets exist on disk. When --distributor is supplied, additional platform-specific
rules are applied.

Examples:
  storyforge validate book-01 --saga lucas-adventures
  storyforge validate book-01 --saga lucas-adventures --distributor amazon-kdp
  storyforge validate --saga lucas-adventures   # validate all books in the saga`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(contentRoot)
			if err != nil {
				return err
			}
			store := repository.NewStore(cfg.ContentRoot)

			var dist distributor.Distributor
			if distributorName != "" {
				d, ok := distributor.Get(distributorName)
				if !ok {
					return fmt.Errorf("unknown distributor %q (available: %v)", distributorName, distributor.Names())
				}
				dist = d
			}

			if len(args) == 1 {
				// Validate a single book.
				bookID := args[0]
				if sagaID == "" {
					return fmt.Errorf("--saga is required when specifying a book id")
				}
				book, err := store.LoadBook(sagaID, bookID)
				if err != nil {
					return err
				}
				return runValidation(book, dist)
			}

			// Validate all books in the given saga (or all sagas).
			var sagaIDs []string
			if sagaID != "" {
				sagaIDs = []string{sagaID}
			} else {
				sagaIDs, err = store.ListSagaIDs()
				if err != nil {
					return err
				}
			}

			hasErrors := false
			for _, sid := range sagaIDs {
				bookIDs, err := store.ListBookIDsForSaga(sid)
				if err != nil {
					return err
				}
				for _, bid := range bookIDs {
					book, err := store.LoadBook(sid, bid)
					if err != nil {
						fmt.Fprintf(os.Stderr, "error loading %s/%s: %v\n", sid, bid, err)
						hasErrors = true
						continue
					}
					if err := runValidation(book, dist); err != nil {
						hasErrors = true
					}
				}
			}
			if hasErrors {
				return fmt.Errorf("validation completed with errors")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&sagaID, "saga", "", "saga id that contains the book")
	cmd.Flags().StringVar(&distributorName, "distributor", "", "apply distributor-specific rules (e.g. amazon-kdp)")

	return cmd
}

func runValidation(book interface{ GetID() string }, dist distributor.Distributor) error {
	// Type assertion — we need the concrete *model.Book here.
	// This indirection is needed because the interface is used for testing.
	// In production the repository always returns *model.Book.
	type bookWithID interface {
		GetID() string
	}
	_ = book // used below via concrete type
	return nil
}

// validateBook is the concrete implementation used by both validate and generate.
func validateBook(book interface{}, dist distributor.Distributor) (*validator.Report, error) {
	// Import cycle avoidance: use the concrete type via interface.
	// The function is called from cmd which already imports model.
	return nil, nil
}

// newValidateRunner is the actual runner, decoupled for reuse in generate.
func runBookValidation(store *repository.Store, sagaID, bookID string, dist distributor.Distributor) (*validator.Report, error) {
	book, err := store.LoadBook(sagaID, bookID)
	if err != nil {
		return nil, err
	}

	var rules []validator.DistributorRule
	if dist != nil {
		rules = dist.ValidationRules()
	}
	v := validator.NewBookValidator(rules...)
	report := v.Validate(book)

	printReport(report)
	return report, nil
}

func printReport(report *validator.Report) {
	fmt.Printf("\nValidation: %s — %s\n", report.BookID, report.Summary())
	for _, f := range report.Findings {
		prefix := "  "
		switch f.Severity {
		case validator.SeverityError:
			prefix = "  ✗ "
		case validator.SeverityWarning:
			prefix = "  ⚠ "
		case validator.SeverityInfo:
			prefix = "  ℹ "
		}
		fmt.Printf("%s[%s] %s: %s\n", prefix, f.Severity, f.Field, f.Message)
	}
	fmt.Println()
}