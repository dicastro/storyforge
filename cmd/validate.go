package cmd

import (
	"fmt"
	"os"

	"github.com/dicastro/storyforge/internal/config"
	"github.com/dicastro/storyforge/internal/distributor"
	"github.com/dicastro/storyforge/internal/repository"
	"github.com/dicastro/storyforge/internal/validator"
	"github.com/spf13/cobra"

	// Register distributors.
	_ "github.com/dicastro/storyforge/internal/distributor/amazon"
)

func newValidateCmd() *cobra.Command {
	var distributorName string
	var sagaID string

	cmd := &cobra.Command{
		Use:   "validate [book-id]",
		Short: "Validate a book (or all books) against schema and distributor requirements",
		Long: `Validate checks that a book's YAML is complete and that all referenced
assets exist on disk. Distributor-specific rules are applied for every
publication target defined in the book.

When --distributor is supplied, only targets for that distributor are checked.

Examples:
  storyforge validate book-01 --saga lucias-adventures
  storyforge validate book-01 --saga lucias-adventures --distributor amazon-kdp
  storyforge validate --saga lucias-adventures`,
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
				bookID := args[0]
				if sagaID == "" {
					return fmt.Errorf("--saga is required when specifying a book id")
				}
				_, err := runBookValidation(store, sagaID, bookID, dist)
				return err
			}

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
					if _, err := runBookValidation(store, sid, bid, dist); err != nil {
						fmt.Fprintf(os.Stderr, "validation failed for %s/%s: %v\n", sid, bid, err)
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
	cmd.Flags().StringVar(&distributorName, "distributor", "", "filter validation to a specific distributor (e.g. amazon-kdp)")

	return cmd
}

// runBookValidation loads and validates a single book, printing the report.
func runBookValidation(store *repository.Store, sagaID, bookID string, dist distributor.Distributor) (*validator.Report, error) {
	book, err := store.LoadBook(sagaID, bookID)
	if err != nil {
		return nil, err
	}

	var rules []validator.DistributorRule
	if dist != nil {
		rules = dist.ValidationRules()
	} else {
		// Apply rules from all distributors referenced in the book's targets.
		seen := map[string]bool{}
		for _, t := range book.PublicationTargets {
			if seen[t.Distributor] {
				continue
			}
			seen[t.Distributor] = true
			if d, ok := distributor.Get(t.Distributor); ok {
				rules = append(rules, d.ValidationRules()...)
			}
		}
	}

	v := validator.NewBookValidator(rules...)
	report := v.Validate(book)

	printReport(report)

	if report.HasErrors() {
		return report, fmt.Errorf("book %q has validation errors", bookID)
	}
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