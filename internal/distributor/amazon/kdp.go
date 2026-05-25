// Package amazon implements the Amazon KDP distributor adapter.
package amazon

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dicastro/storyforge/internal/distributor"
	"github.com/dicastro/storyforge/internal/imaging"
	"github.com/dicastro/storyforge/internal/model"
	"github.com/dicastro/storyforge/internal/validator"
	"gopkg.in/yaml.v3"
)

//go:embed kdp-specs.yaml
var specsData []byte

func init() {
	distributor.Register(&KDPDistributor{})
}

// KDPDistributor implements distributor.Distributor for Amazon KDP.
type KDPDistributor struct {
	specs kdpSpecs
}

func (k *KDPDistributor) init() error {
	if k.specs.Name != "" {
		return nil
	}
	return yaml.Unmarshal(specsData, &k.specs)
}

func (k *KDPDistributor) Name() string        { return "amazon-kdp" }
func (k *KDPDistributor) DisplayName() string { return "Amazon KDP" }

// findSize returns the sizeSpec for the given size id, or nil if not found.
func (k *KDPDistributor) findSize(sizeID string) *sizeSpec {
	for i := range k.specs.Sizes {
		if k.specs.Sizes[i].ID == sizeID {
			return &k.specs.Sizes[i]
		}
	}
	return nil
}

// SpineWidthInches calculates the cover spine width for the given target and page count.
func (k *KDPDistributor) SpineWidthInches(target model.PublicationTarget, pages int) (float64, error) {
	if err := k.init(); err != nil {
		return 0, err
	}
	sz := k.findSize(target.Size)
	if sz == nil {
		return 0, fmt.Errorf("unknown size %q", target.Size)
	}
	binding, ok := sz.Bindings[string(target.Binding)]
	if !ok || !binding.Supported {
		return 0, fmt.Errorf("binding %q not supported for size %q", target.Binding, target.Size)
	}
	paper, ok := binding.PaperTypes[string(target.Paper)]
	if !ok || !paper.Supported {
		return 0, fmt.Errorf("paper %q not supported for %q/%q", target.Paper, target.Size, target.Binding)
	}
	return float64(pages)*paper.ThicknessPerPageInches + k.specs.CoverBoardInches, nil
}

// ValidationRules returns KDP-specific checks derived from the specs file.
func (k *KDPDistributor) ValidationRules() []validator.DistributorRule {
	return []validator.DistributorRule{
		k.ruleTargets,
		k.ruleCoverImage,
		k.ruleISBN,
	}
}

// ruleTargets validates every publication target that references this distributor.
func (k *KDPDistributor) ruleTargets(book *model.Book) []validator.Finding {
	if err := k.init(); err != nil {
		return []validator.Finding{{
			Severity: validator.SeverityError,
			Field:    "distributor.specs",
			Message:  fmt.Sprintf("failed to load KDP specs: %v", err),
		}}
	}

	var findings []validator.Finding
	for i, t := range book.PublicationTargets {
		if t.Distributor != k.Name() {
			continue
		}
		field := fmt.Sprintf("publication_targets[%d]", i)
		findings = append(findings, k.validateTarget(field, t, book.Format.PagesInterior)...)
	}
	return findings
}

func (k *KDPDistributor) validateTarget(field string, t model.PublicationTarget, pages int) []validator.Finding {
	var findings []validator.Finding

	sz := k.findSize(t.Size)
	if sz == nil {
		findings = append(findings, validator.Finding{
			Severity: validator.SeverityError,
			Field:    field + ".size",
			Message:  fmt.Sprintf("size %q is not supported by Amazon KDP", t.Size),
		})
		return findings // further checks meaningless without a valid size
	}

	binding, ok := sz.Bindings[string(t.Binding)]
	if !ok || !binding.Supported {
		findings = append(findings, validator.Finding{
			Severity: validator.SeverityError,
			Field:    field + ".binding",
			Message:  fmt.Sprintf("binding %q is not supported for size %q on Amazon KDP", t.Binding, t.Size),
		})
		return findings
	}

	if pages < binding.MinPages {
		findings = append(findings, validator.Finding{
			Severity: validator.SeverityError,
			Field:    field + ".pages",
			Message:  fmt.Sprintf("Amazon KDP requires at least %d interior pages for %s/%s (got %d)", binding.MinPages, t.Size, t.Binding, pages),
		})
	}
	if pages > binding.MaxPages {
		findings = append(findings, validator.Finding{
			Severity: validator.SeverityError,
			Field:    field + ".pages",
			Message:  fmt.Sprintf("Amazon KDP allows a maximum of %d interior pages for %s/%s (got %d)", binding.MaxPages, t.Size, t.Binding, pages),
		})
	}

	paper, ok := binding.PaperTypes[string(t.Paper)]
	if !ok || !paper.Supported {
		findings = append(findings, validator.Finding{
			Severity: validator.SeverityError,
			Field:    field + ".paper",
			Message:  fmt.Sprintf("paper %q is not supported for %s/%s on Amazon KDP", t.Paper, t.Size, t.Binding),
		})
	}

	return findings
}

func (k *KDPDistributor) ruleCoverImage(book *model.Book) []validator.Finding {
	var findings []validator.Finding
	if book.Cover.ImagePath == "" && book.Cover.ImagePrompt == "" {
		findings = append(findings, validator.Finding{
			Severity: validator.SeverityError,
			Field:    "cover.image_path",
			Message:  "Amazon KDP requires a cover image (image_path or image_prompt must be set)",
		})
	}
	return findings
}

func (k *KDPDistributor) ruleISBN(book *model.Book) []validator.Finding {
	return []validator.Finding{{
		Severity: validator.SeverityInfo,
		Field:    "metadata.isbn",
		Message:  "No ISBN set. Amazon will assign a free ASIN; add your own ISBN if you want wider distribution.",
	}}
}

// Generate produces the KDP submission manifest, checklist, and image prompt sheet.
func (k *KDPDistributor) Generate(
	book *model.Book,
	target model.PublicationTarget,
	outputDir string,
	opts distributor.GenerateOptions,
) (*distributor.GenerationResult, error) {
	if err := k.init(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	result := &distributor.GenerationResult{
		Distributor: k.Name(),
		BookID:      book.ID,
		Target:      target,
		OutputDir:   outputDir,
	}

	sz := k.findSize(target.Size)
	if sz == nil {
		return nil, fmt.Errorf("size %q not found in KDP specs", target.Size)
	}

	// If mock images requested, generate placeholders for any missing image files.
	if opts.MockImages {
		if warnings, err := k.generateMockImages(book, sz, outputDir); err != nil {
			return nil, err
		} else {
			result.Warnings = append(result.Warnings, warnings...)
		}
	}

	manifestFile, err := k.writeManifest(book, target, sz, outputDir)
	if err != nil {
		return nil, err
	}
	result.Files = append(result.Files, distributor.GeneratedFile{
		RelativePath: manifestFile,
		Description:  "KDP submission metadata manifest",
	})

	checklistFile, err := k.writeChecklist(book, target, outputDir)
	if err != nil {
		return nil, err
	}
	result.Files = append(result.Files, distributor.GeneratedFile{
		RelativePath: checklistFile,
		Description:  "Pre-submission checklist",
	})

	promptsFile, err := k.writeImagePrompts(book, target, outputDir)
	if err != nil {
		return nil, err
	}
	result.Files = append(result.Files, distributor.GeneratedFile{
		RelativePath: promptsFile,
		Description:  "Image generation prompt sheet",
	})

	return result, nil
}

// generateMockImages creates white+cross placeholder PNG files for every image
// path referenced by the book that does not already exist on disk.
func (k *KDPDistributor) generateMockImages(book *model.Book, sz *sizeSpec, outputDir string) ([]string, error) {
	const dpi = 300
	var warnings []string

	widthPx := int((sz.WidthInches + 2*sz.BleedInches) * dpi)
	heightPx := int((sz.HeightInches + 2*sz.BleedInches) * dpi)

	generate := func(relPath string) error {
		if relPath == "" {
			return nil
		}
		absPath := filepath.Join(book.RootPath, relPath)
		if _, err := os.Stat(absPath); err == nil {
			return nil // file exists — do not overwrite real assets
		}
		if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
			return fmt.Errorf("creating directory for mock image %s: %w", relPath, err)
		}
		if err := imaging.WriteMockImage(absPath, widthPx, heightPx); err != nil {
			return fmt.Errorf("writing mock image %s: %w", relPath, err)
		}
		warnings = append(warnings, fmt.Sprintf("mock image generated: %s (%dx%d px)", relPath, widthPx, heightPx))
		return nil
	}

	if err := generate(book.Cover.ImagePath); err != nil {
		return warnings, err
	}
	if err := generate(book.BackCover.ImagePath); err != nil {
		return warnings, err
	}
	for _, s := range book.Spreads {
		if err := generate(s.Right.ImagePath); err != nil {
			return warnings, err
		}
	}
	return warnings, nil
}

// ---- manifest ----------------------------------------------------------------

type kdpManifest struct {
	GeneratedAt   string            `json:"generated_at"`
	Distributor   string            `json:"distributor"`
	BookID        string            `json:"book_id"`
	Title         string            `json:"title"`
	Author        string            `json:"author"`
	Language      string            `json:"language"`
	Edition       int               `json:"edition"`
	Format        kdpFormatManifest `json:"format"`
	Status        string            `json:"status"`
	CoverImage    string            `json:"cover_image"`
	Spreads       int               `json:"spreads"`
	SpineWidthIn  float64           `json:"spine_width_inches"`
}

type kdpFormatManifest struct {
	Size          string  `json:"size"`
	WidthInches   float64 `json:"width_inches"`
	HeightInches  float64 `json:"height_inches"`
	BleedInches   float64 `json:"bleed_inches"`
	PagesInterior int     `json:"pages_interior"`
	InteriorColor string  `json:"interior_color"`
	Paper         string  `json:"paper"`
	Binding       string  `json:"binding"`
}

func (k *KDPDistributor) writeManifest(book *model.Book, target model.PublicationTarget, sz *sizeSpec, outputDir string) (string, error) {
	spine, _ := k.SpineWidthInches(target, book.Format.PagesInterior)

	m := kdpManifest{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Distributor:  k.DisplayName(),
		BookID:       book.ID,
		Title:        book.Metadata.Title.Get(target.Language),
		Author:       book.Metadata.Author,
		Language:     target.Language,
		Edition:      book.Metadata.Edition,
		Status:       string(book.Metadata.Status),
		CoverImage:   book.Cover.ImagePath,
		Spreads:      len(book.Spreads),
		SpineWidthIn: spine,
		Format: kdpFormatManifest{
			Size:          target.Size,
			WidthInches:   sz.WidthInches,
			HeightInches:  sz.HeightInches,
			BleedInches:   sz.BleedInches,
			PagesInterior: book.Format.PagesInterior,
			InteriorColor: string(target.InteriorColor),
			Paper:         string(target.Paper),
			Binding:       string(target.Binding),
		},
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshalling manifest: %w", err)
	}
	filename := "kdp-manifest.json"
	if err := os.WriteFile(filepath.Join(outputDir, filename), data, 0644); err != nil {
		return "", fmt.Errorf("writing manifest: %w", err)
	}
	return filename, nil
}

// ---- checklist ---------------------------------------------------------------

func (k *KDPDistributor) writeChecklist(book *model.Book, target model.PublicationTarget, outputDir string) (string, error) {
	var sb strings.Builder
	title := book.Metadata.Title.Get(target.Language)

	sb.WriteString(fmt.Sprintf("# KDP Submission Checklist — %s (%s/%s)\n\n", title, target.Size, target.Language))
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().UTC().Format("2006-01-02 15:04 UTC")))

	sb.WriteString("## Metadata\n\n")
	sb.WriteString(checkItem(book.Metadata.Author != "", "Author set"))
	sb.WriteString(checkItem(target.Language != "", "Language set"))
	sb.WriteString(checkItem(target.Size != "", "Book size specified"))

	binding := k.findSize(target.Size)
	if binding != nil {
		b := binding.Bindings[string(target.Binding)]
		sb.WriteString(checkItem(book.Format.PagesInterior >= b.MinPages,
			fmt.Sprintf("Minimum %d interior pages for %s/%s", b.MinPages, target.Size, target.Binding)))
	}

	sb.WriteString("\n## Cover\n\n")
	sb.WriteString(checkItem(book.Cover.ImagePath != "", "Cover image path set"))
	sb.WriteString(checkItem(book.BackCover.ImagePath != "", "Back cover image path set"))

	sb.WriteString("\n## Interior spreads\n\n")
	for i, spread := range book.Spreads {
		hasText := spread.Left.Text.Get(target.Language) != ""
		hasImage := spread.Right.ImagePath != ""
		sb.WriteString(checkItem(hasText && hasImage, fmt.Sprintf("Spread %d — text + image", i+1)))
	}

	sb.WriteString("\n## Characters\n\n")
	for _, c := range book.ResolvedCharacters {
		sb.WriteString(checkItem(c.VisualPrompt != "", fmt.Sprintf("Character %q has visual prompt", c.Name)))
	}

	filename := "kdp-checklist.md"
	if err := os.WriteFile(filepath.Join(outputDir, filename), []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("writing checklist: %w", err)
	}
	return filename, nil
}

// ---- image prompts -----------------------------------------------------------

func (k *KDPDistributor) writeImagePrompts(book *model.Book, target model.PublicationTarget, outputDir string) (string, error) {
	var sb strings.Builder
	title := book.Metadata.Title.Get(target.Language)

	sz := k.findSize(target.Size)
	widthPx := 0
	heightPx := 0
	if sz != nil {
		const dpi = 300
		widthPx = int((sz.WidthInches + 2*sz.BleedInches) * dpi)
		heightPx = int((sz.HeightInches + 2*sz.BleedInches) * dpi)
	}

	sb.WriteString(fmt.Sprintf("# Image Prompt Sheet — %s (%s / %s)\n\n", title, target.Size, target.Language))
	sb.WriteString("> Copy each prompt into your image generation tool of choice.\n")
	if widthPx > 0 {
		sb.WriteString(fmt.Sprintf("> Required image size (300 DPI, with bleed): **%d × %d px**\n", widthPx, heightPx))
	}
	sb.WriteString("> All images must be 300 DPI minimum, RGB, no embedded text.\n\n")

	sb.WriteString("## Cover\n\n")
	if book.Cover.ImagePrompt != "" {
		sb.WriteString("```\n" + strings.TrimSpace(book.Cover.ImagePrompt) + "\n```\n\n")
	}

	sb.WriteString("## Back cover\n\n")
	if book.BackCover.ImagePrompt != "" {
		sb.WriteString("```\n" + strings.TrimSpace(book.BackCover.ImagePrompt) + "\n```\n\n")
	}

	sb.WriteString("## Interior spreads\n\n")
	for _, spread := range book.Spreads {
		if spread.Right.ImagePrompt != "" {
			sb.WriteString(fmt.Sprintf("### Spread %d\n\n", spread.Number))
			sb.WriteString("```\n" + strings.TrimSpace(spread.Right.ImagePrompt) + "\n```\n\n")
		}
	}

	filename := "image-prompts.md"
	if err := os.WriteFile(filepath.Join(outputDir, filename), []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("writing image prompts: %w", err)
	}
	return filename, nil
}

func checkItem(ok bool, label string) string {
	if ok {
		return fmt.Sprintf("- [x] %s\n", label)
	}
	return fmt.Sprintf("- [ ] %s\n", label)
}