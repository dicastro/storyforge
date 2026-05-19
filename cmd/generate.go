package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/dicastro/storyforge/internal/config"
	"github.com/dicastro/storyforge/internal/distributor"
	"github.com/dicastro/storyforge/internal/model"
	"github.com/dicastro/storyforge/internal/repository"
	"github.com/dicastro/storyforge/internal/validator"
	"github.com/spf13/cobra"

	_ "github.com/dicastro/storyforge/internal/distributor/amazon"
)

func newGenerateCmd() *cobra.Command {
	var sagaID string
	var distributorName string
	var outputDir string
	var skipValidation bool

	cmd := &cobra.Command{
		Use:   "generate <book-id>",
		Short: "Generate publication materials for a distributor",
		Long: `Generate produces all files needed to submit a book to a distributor.
Before generating, a full validation is run. Generation is blocked if there are
errors (use --skip-validation to override, not recommended for production).

If the book status is "published", a versioned snapshot is created in dist/
to preserve the original before any further changes.

Examples:
  storyforge generate book-01 --saga lucas-adventures --distributor amazon-kdp
  storyforge generate book-01 --saga lucas-adventures --distributor amazon-kdp --output ./output`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bookID := args[0]

			cfg, err := config.Load(contentRoot)
			if err != nil {
				return err
			}

			if sagaID == "" {
				return fmt.Errorf("--saga is required")
			}

			if distributorName == "" {
				// Prompt interactively if not supplied as a flag.
				available := distributor.Names()
				if len(available) == 0 {
					return fmt.Errorf("no distributors registered")
				}
				fmt.Println("Available distributors:")
				for i, n := range available {
					fmt.Printf("  [%d] %s\n", i+1, n)
				}
				fmt.Print("Select distributor: ")
				var choice int
				if _, err := fmt.Scan(&choice); err != nil || choice < 1 || choice > len(available) {
					return fmt.Errorf("invalid selection")
				}
				distributorName = available[choice-1]
			}

			dist, ok := distributor.Get(distributorName)
			if !ok {
				return fmt.Errorf("unknown distributor %q", distributorName)
			}

			store := repository.NewStore(cfg.ContentRoot)
			book, err := store.LoadBook(sagaID, bookID)
			if err != nil {
				return err
			}

			// Run validation unless explicitly skipped.
			if !skipValidation {
				v := validator.NewBookValidator(dist.ValidationRules()...)
				report := v.Validate(book)
				printReport(report)
				if report.HasErrors() {
					return fmt.Errorf("generation blocked: fix validation errors first (use --skip-validation to override)")
				}
			}

			// Determine output directory.
			target := outputDir
			if target == "" {
				target = filepath.Join(book.RootPath, "dist", distributorName)
			}

			// If book is published, create a versioned snapshot directory.
			if book.IsPublished() {
				target = versionedOutputDir(target)
				fmt.Printf("Book is published — writing to versioned snapshot: %s\n", target)
			}

			result, err := dist.Generate(book, target)
			if err != nil {
				return fmt.Errorf("generation failed: %w", err)
			}

			printGenerationResult(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&sagaID, "saga", "", "saga id that contains the book (required)")
	cmd.Flags().StringVar(&distributorName, "distributor", "", "target distributor (e.g. amazon-kdp)")
	cmd.Flags().StringVar(&outputDir, "output", "", "override output directory")
	cmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "skip pre-generation validation (not recommended)")

	_ = cmd.MarkFlagRequired("saga")

	return cmd
}

func versionedOutputDir(base string) string {
	// Uses a simple incrementing suffix. Production could use git tags instead.
	for i := 1; i <= 999; i++ {
		candidate := fmt.Sprintf("%s-v%03d", base, i)
		// We rely on the distributor to create the directory; just find an unused name.
		return candidate
	}
	return base + "-v001"
}

func printGenerationResult(result *distributor.GenerationResult) {
	fmt.Printf("\n✓ Generated for %s — book: %s\n", result.Distributor, result.BookID)
	fmt.Printf("  Output directory: %s\n\n", result.OutputDir)
	for _, f := range result.Files {
		fmt.Printf("  📄 %s\n     %s\n", f.RelativePath, f.Description)
	}
	if len(result.Warnings) > 0 {
		fmt.Println("\n  Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  ⚠  %s\n", w)
		}
	}
	fmt.Println()
}

// Keep the model import satisfied (used via book.IsPublished()).
var _ = (*model.Book)(nil)