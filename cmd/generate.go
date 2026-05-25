package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

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
	var distributorFilter string
	var sizeFilter string
	var languageFilter string
	var outputDir string
	var skipValidation bool
	var mockImages bool

	cmd := &cobra.Command{
		Use:   "generate <book-id>",
		Short: "Generate publication materials for all (or filtered) targets",
		Long: `Generate produces all files needed to submit a book to a distributor.

A book can have multiple publication targets (different distributors, sizes, or
languages). By default all targets are generated. Use --distributor, --size,
and --language to generate a single specific target.

Before generating, a full validation is run. Generation is blocked if there are
errors (use --skip-validation to override, not recommended for production).

If the book status is "published", output is written to a versioned snapshot
directory to preserve previous artefacts.

The --mock-images flag generates white placeholder images (with a diagonal
cross) for any image path that does not yet have a real file on disk. This
lets you preview the full PDF layout before illustrations are ready.

Examples:
  storyforge generate book-01 --saga lucias-adventures
  storyforge generate book-01 --saga lucias-adventures --distributor amazon-kdp
  storyforge generate book-01 --saga lucias-adventures --distributor amazon-kdp --size 21x21cm --language es
  storyforge generate book-01 --saga lucias-adventures --mock-images`,
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

			// Collect targets that match the supplied filters.
			targets := filterTargets(book.PublicationTargets, distributorFilter, sizeFilter, languageFilter)
			if len(targets) == 0 {
				return fmt.Errorf("no publication targets match the given filters")
			}

			// Validate before generating (once, using rules from all relevant distributors).
			if !skipValidation {
				seen := map[string]bool{}
				var rules []validator.DistributorRule
				for _, t := range targets {
					if seen[t.Distributor] {
						continue
					}
					seen[t.Distributor] = true
					if d, ok := distributor.Get(t.Distributor); ok {
						rules = append(rules, d.ValidationRules()...)
					}
				}
				v := validator.NewBookValidator(rules...)
				report := v.Validate(book)
				printReport(report)
				if report.HasErrors() {
					return fmt.Errorf("generation blocked: fix validation errors first (use --skip-validation to override)")
				}
			}

			opts := distributor.GenerateOptions{
				MockImages: mockImages,
			}

			hasErrors := false
			for _, target := range targets {
				dist, ok := distributor.Get(target.Distributor)
				if !ok {
					fmt.Printf("⚠  Distributor %q is not registered — skipping target %s/%s/%s\n",
						target.Distributor, target.Size, target.Language, target.Binding)
					hasErrors = true
					continue
				}

				targetDir := resolveOutputDir(outputDir, book, target, dist)
				if book.IsPublished() {
					targetDir = versionedOutputDir(targetDir)
					fmt.Printf("Book is published — writing to versioned snapshot: %s\n", targetDir)
				}

				result, err := dist.Generate(book, target, targetDir, opts)
				if err != nil {
					fmt.Printf("✗ Generation failed for %s/%s/%s: %v\n",
						target.Distributor, target.Size, target.Language, err)
					hasErrors = true
					continue
				}
				printGenerationResult(result)
			}

			if hasErrors {
				return fmt.Errorf("one or more targets failed to generate")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&sagaID, "saga", "", "saga id that contains the book (required)")
	cmd.Flags().StringVar(&distributorFilter, "distributor", "", "generate only targets for this distributor (e.g. amazon-kdp)")
	cmd.Flags().StringVar(&sizeFilter, "size", "", "generate only targets with this size (e.g. 21x21cm)")
	cmd.Flags().StringVar(&languageFilter, "language", "", "generate only targets with this language (e.g. es)")
	cmd.Flags().StringVar(&outputDir, "output", "", "override base output directory")
	cmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "skip pre-generation validation (not recommended)")
	cmd.Flags().BoolVar(&mockImages, "mock-images", false, "generate placeholder images for missing image files")

	_ = cmd.MarkFlagRequired("saga")

	return cmd
}

// filterTargets returns the subset of targets matching all non-empty filters.
func filterTargets(targets []model.PublicationTarget, dist, size, lang string) []model.PublicationTarget {
	var out []model.PublicationTarget
	for _, t := range targets {
		if dist != "" && t.Distributor != dist {
			continue
		}
		if size != "" && t.Size != size {
			continue
		}
		if lang != "" && t.Language != lang {
			continue
		}
		out = append(out, t)
	}
	return out
}

// resolveOutputDir builds the output path for a target:
//
//	<book-root>/dist/<distributor>/<size>-<language>-<binding>
func resolveOutputDir(override string, book *model.Book, target model.PublicationTarget, dist distributor.Distributor) string {
	if override != "" {
		return filepath.Join(override, dist.Name(), targetDirName(target))
	}
	return filepath.Join(book.RootPath, "dist", dist.Name(), targetDirName(target))
}

// targetDirName returns a slug for the target combination, e.g. "21x21cm-es-paperback".
func targetDirName(t model.PublicationTarget) string {
	return strings.Join([]string{t.Size, t.Language, string(t.Binding)}, "-")
}

func versionedOutputDir(base string) string {
	for i := 1; i <= 999; i++ {
		candidate := fmt.Sprintf("%s-v%03d", base, i)
		return candidate
	}
	return base + "-v001"
}

func printGenerationResult(result *distributor.GenerationResult) {
	fmt.Printf("\n✓ Generated for %s — book: %s (%s / %s / %s)\n",
		result.Distributor, result.BookID,
		result.Target.Size, result.Target.Language, result.Target.Binding)
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