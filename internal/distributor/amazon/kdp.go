// Package amazon implements the Amazon KDP distributor adapter.
// It validates KDP-specific requirements and generates a submission manifest.
package amazon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dicastro/storyforge/internal/distributor"
	"github.com/dicastro/storyforge/internal/model"
	"github.com/dicastro/storyforge/internal/validator"
)

func init() {
	distributor.Register(&KDPDistributor{})
}

// KDPDistributor implements distributor.Distributor for Amazon KDP.
type KDPDistributor struct{}

func (k *KDPDistributor) Name() string        { return "amazon-kdp" }
func (k *KDPDistributor) DisplayName() string  { return "Amazon KDP" }

// ValidationRules returns KDP-specific checks.
func (k *KDPDistributor) ValidationRules() []validator.DistributorRule {
	return []validator.DistributorRule{
		k.rulePageCount,
		k.ruleCoverImage,
		k.ruleISBN,
	}
}

func (k *KDPDistributor) rulePageCount(book *model.Book) []validator.Finding {
	var findings []validator.Finding
	pages := book.Format.PagesInterior
	if pages < 24 {
		findings = append(findings, validator.Finding{
			Severity: validator.SeverityError,
			Field:    "format.pages_interior",
			Message:  fmt.Sprintf("Amazon KDP requires at least 24 interior pages (got %d)", pages),
		})
	}
	if pages > 828 {
		findings = append(findings, validator.Finding{
			Severity: validator.SeverityError,
			Field:    "format.pages_interior",
			Message:  fmt.Sprintf("Amazon KDP allows a maximum of 828 interior pages (got %d)", pages),
		})
	}
	return findings
}

func (k *KDPDistributor) ruleCoverImage(book *model.Book) []validator.Finding {
	var findings []validator.Finding
	if book.Cover.ImagePath == "" {
		findings = append(findings, validator.Finding{
			Severity: validator.SeverityError,
			Field:    "cover.image_path",
			Message:  "Amazon KDP requires a cover image file",
		})
	}
	return findings
}

func (k *KDPDistributor) ruleISBN(book *model.Book) []validator.Finding {
	// ISBN is optional for KDP (they assign an ASIN), but recommended.
	// Return an info-level notice rather than an error.
	return []validator.Finding{
		{
			Severity: validator.SeverityInfo,
			Field:    "metadata.isbn",
			Message:  "No ISBN set. Amazon will assign a free ASIN; add your own ISBN if you want wider distribution.",
		},
	}
}

// Generate produces the KDP submission manifest and a checklist.
func (k *KDPDistributor) Generate(book *model.Book, outputDir string) (*distributor.GenerationResult, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	result := &distributor.GenerationResult{
		Distributor: k.Name(),
		BookID:      book.ID,
		OutputDir:   outputDir,
	}

	// 1. Write submission manifest (JSON).
	manifestFile, err := k.writeManifest(book, outputDir)
	if err != nil {
		return nil, err
	}
	result.Files = append(result.Files, distributor.GeneratedFile{
		RelativePath: manifestFile,
		Description:  "KDP submission metadata manifest",
	})

	// 2. Write human-readable checklist (Markdown).
	checklistFile, err := k.writeChecklist(book, outputDir)
	if err != nil {
		return nil, err
	}
	result.Files = append(result.Files, distributor.GeneratedFile{
		RelativePath: checklistFile,
		Description:  "Pre-submission checklist",
	})

	// 3. Write image prompt sheet (Markdown) — useful before images are ready.
	promptsFile, err := k.writeImagePrompts(book, outputDir)
	if err != nil {
		return nil, err
	}
	result.Files = append(result.Files, distributor.GeneratedFile{
		RelativePath: promptsFile,
		Description:  "Image generation prompt sheet",
	})

	return result, nil
}

// ---- internal generation helpers --------------------------------------------

type kdpManifest struct {
	GeneratedAt string            `json:"generated_at"`
	Distributor string            `json:"distributor"`
	BookID      string            `json:"book_id"`
	Title       string            `json:"title"`
	Author      string            `json:"author"`
	Language    string            `json:"language"`
	Edition     int               `json:"edition"`
	Format      kdpFormatManifest `json:"format"`
	Status      string            `json:"status"`
	CoverImage  string            `json:"cover_image"`
	Spreads     int               `json:"spreads"`
}

type kdpFormatManifest struct {
	Size          string `json:"size"`
	PagesInterior int    `json:"pages_interior"`
	InteriorColor string `json:"interior_color"`
	Paper         string `json:"paper"`
	Bleed         bool   `json:"bleed"`
	Cover         string `json:"cover"`
}

func (k *KDPDistributor) writeManifest(book *model.Book, outputDir string) (string, error) {
	hints := book.DistributorHints.AmazonKDP

	interiorColor, _ := hints["interior_color"].(string)
	paper, _ := hints["paper"].(string)
	bleed, _ := hints["bleed"].(bool)

	m := kdpManifest{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Distributor: k.DisplayName(),
		BookID:      book.ID,
		Title:       book.Metadata.Title.Get(book.Metadata.Language),
		Author:      book.Metadata.Author,
		Language:    book.Metadata.Language,
		Edition:     book.Metadata.Edition,
		Status:      string(book.Metadata.Status),
		CoverImage:  book.Cover.ImagePath,
		Spreads:     len(book.Spreads),
		Format: kdpFormatManifest{
			Size:          book.Format.Size,
			PagesInterior: book.Format.PagesInterior,
			InteriorColor: interiorColor,
			Paper:         paper,
			Bleed:         bleed,
			Cover:         book.Format.Cover,
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

func (k *KDPDistributor) writeChecklist(book *model.Book, outputDir string) (string, error) {
	var sb strings.Builder
	title := book.Metadata.Title.Get(book.Metadata.Language)

	sb.WriteString(fmt.Sprintf("# KDP Submission Checklist — %s\n\n", title))
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().UTC().Format("2006-01-02 15:04 UTC")))

	sb.WriteString("## Metadata\n\n")
	sb.WriteString(checkItem(book.Metadata.Author != "", "Author set"))
	sb.WriteString(checkItem(book.Metadata.Language != "", "Language set"))
	sb.WriteString(checkItem(book.Format.Size != "", "Book size specified"))
	sb.WriteString(checkItem(book.Format.PagesInterior >= 24, "Minimum 24 interior pages"))

	sb.WriteString("\n## Cover\n\n")
	sb.WriteString(checkItem(book.Cover.ImagePath != "", "Cover image path set"))
	sb.WriteString(checkItem(book.BackCover.ImagePath != "", "Back cover image path set"))

	sb.WriteString("\n## Interior pages\n\n")
	for i, spread := range book.Spreads {
		hasText := spread.Left.Text.Get("es") != "" || spread.Left.Text.Get("en") != ""
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

func (k *KDPDistributor) writeImagePrompts(book *model.Book, outputDir string) (string, error) {
	var sb strings.Builder
	title := book.Metadata.Title.Get(book.Metadata.Language)

	sb.WriteString(fmt.Sprintf("# Image Prompt Sheet — %s\n\n", title))
	sb.WriteString("> Copy each prompt into your image generation tool of choice.\n")
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