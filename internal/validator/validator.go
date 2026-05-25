// Package validator checks books for structural and asset completeness.
package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dicastro/storyforge/internal/model"
)

// Severity classifies a validation finding.
type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
	SeverityInfo    Severity = "INFO"
)

// Finding is a single validation result.
type Finding struct {
	Severity Severity
	Field    string
	Message  string
}

func (f Finding) String() string {
	return fmt.Sprintf("[%s] %s: %s", f.Severity, f.Field, f.Message)
}

// Report collects findings from a validation run.
type Report struct {
	BookID   string
	Findings []Finding
}

// HasErrors returns true if any finding has ERROR severity.
func (r *Report) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Summary returns a one-line status string.
func (r *Report) Summary() string {
	errors, warnings := 0, 0
	for _, f := range r.Findings {
		switch f.Severity {
		case SeverityError:
			errors++
		case SeverityWarning:
			warnings++
		}
	}
	if errors == 0 && warnings == 0 {
		return "OK"
	}
	return fmt.Sprintf("%d error(s), %d warning(s)", errors, warnings)
}

// BookValidator validates a book against schema and optional distributor rules.
type BookValidator struct {
	distributorRules []DistributorRule
}

// DistributorRule is an additional check injected by a distributor implementation.
type DistributorRule func(book *model.Book) []Finding

// NewBookValidator creates a validator with optional extra distributor rules.
func NewBookValidator(rules ...DistributorRule) *BookValidator {
	return &BookValidator{distributorRules: rules}
}

// Validate runs all checks and returns a Report.
func (v *BookValidator) Validate(book *model.Book) *Report {
	report := &Report{BookID: book.ID}

	report.Findings = append(report.Findings, v.checkMetadata(book)...)
	report.Findings = append(report.Findings, v.checkFormat(book)...)
	report.Findings = append(report.Findings, v.checkPublicationTargets(book)...)
	report.Findings = append(report.Findings, v.checkSpreads(book)...)
	report.Findings = append(report.Findings, v.checkImages(book)...)
	report.Findings = append(report.Findings, v.checkCharacters(book)...)
	report.Findings = append(report.Findings, v.checkTypography(book)...)

	for _, rule := range v.distributorRules {
		report.Findings = append(report.Findings, rule(book)...)
	}

	return report
}

func (v *BookValidator) checkMetadata(book *model.Book) []Finding {
	var findings []Finding

	if book.ID == "" {
		findings = append(findings, Finding{SeverityError, "id", "book id is required"})
	}
	if book.Metadata.Title.Get("es") == "" && book.Metadata.Title.Get("en") == "" {
		findings = append(findings, Finding{SeverityError, "metadata.title", "at least one language title is required"})
	}
	if book.Metadata.Author == "" {
		findings = append(findings, Finding{SeverityWarning, "metadata.author", "author is not set"})
	}
	if book.Metadata.Status == "" {
		findings = append(findings, Finding{SeverityError, "metadata.status", "status is required (draft|wip|ready|published)"})
	}
	validStatuses := map[model.BookStatus]bool{
		model.StatusDraft:     true,
		model.StatusWIP:       true,
		model.StatusReady:     true,
		model.StatusPublished: true,
	}
	if book.Metadata.Status != "" && !validStatuses[book.Metadata.Status] {
		findings = append(findings, Finding{
			SeverityError, "metadata.status",
			fmt.Sprintf("invalid status %q (must be draft|wip|ready|published)", book.Metadata.Status),
		})
	}

	return findings
}

func (v *BookValidator) checkFormat(book *model.Book) []Finding {
	var findings []Finding

	if book.Format.PagesInterior == 0 {
		findings = append(findings, Finding{SeverityWarning, "format.pages_interior", "number of interior pages is not set"})
	} else if book.Format.PagesInterior%2 != 0 {
		findings = append(findings, Finding{SeverityError, "format.pages_interior", "interior pages must be an even number"})
	}

	return findings
}

func (v *BookValidator) checkPublicationTargets(book *model.Book) []Finding {
	var findings []Finding

	if len(book.PublicationTargets) == 0 {
		findings = append(findings, Finding{
			SeverityWarning, "publication_targets",
			"no publication targets defined; add at least one target to generate output",
		})
		return findings
	}

	validBindings := map[model.BindingType]bool{
		model.BindingPaperback: true,
		model.BindingHardcover: true,
	}
	validPapers := map[model.PaperType]bool{
		model.PaperWhite: true,
		model.PaperCream: true,
		model.PaperColor: true,
	}
	validColors := map[model.InteriorColor]bool{
		model.InteriorFullColor:  true,
		model.InteriorBlackWhite: true,
	}

	for i, t := range book.PublicationTargets {
		field := fmt.Sprintf("publication_targets[%d]", i)

		if t.Distributor == "" {
			findings = append(findings, Finding{SeverityError, field + ".distributor", "distributor is required"})
		}
		if t.Size == "" {
			findings = append(findings, Finding{SeverityError, field + ".size", "size is required (e.g. \"21x21cm\")"})
		}
		if t.Language == "" {
			findings = append(findings, Finding{SeverityError, field + ".language", "language is required (e.g. \"es\")"})
		}
		if t.Binding == "" {
			findings = append(findings, Finding{SeverityError, field + ".binding", "binding is required (paperback|hardcover)"})
		} else if !validBindings[t.Binding] {
			findings = append(findings, Finding{
				SeverityError, field + ".binding",
				fmt.Sprintf("invalid binding %q (must be paperback|hardcover)", t.Binding),
			})
		}
		if t.Paper == "" {
			findings = append(findings, Finding{SeverityError, field + ".paper", "paper is required (white|cream|color)"})
		} else if !validPapers[t.Paper] {
			findings = append(findings, Finding{
				SeverityError, field + ".paper",
				fmt.Sprintf("invalid paper %q (must be white|cream|color)", t.Paper),
			})
		}
		if t.InteriorColor == "" {
			findings = append(findings, Finding{SeverityError, field + ".interior_color", "interior_color is required (full_color|black_white)"})
		} else if !validColors[t.InteriorColor] {
			findings = append(findings, Finding{
				SeverityError, field + ".interior_color",
				fmt.Sprintf("invalid interior_color %q (must be full_color|black_white)", t.InteriorColor),
			})
		}

		// Warn if the target language has no text in any spread.
		missingLang := true
		for _, s := range book.Spreads {
			if s.Left.Text.Get(t.Language) != "" {
				missingLang = false
				break
			}
		}
		if missingLang && len(book.Spreads) > 0 {
			findings = append(findings, Finding{
				SeverityWarning, field + ".language",
				fmt.Sprintf("no spread text found for language %q", t.Language),
			})
		}
	}

	return findings
}

func (v *BookValidator) checkSpreads(book *model.Book) []Finding {
	var findings []Finding

	if len(book.Spreads) == 0 {
		findings = append(findings, Finding{SeverityError, "spreads", "book has no spreads"})
		return findings
	}

	for i, spread := range book.Spreads {
		field := fmt.Sprintf("spreads[%d]", i)
		if spread.Number == 0 {
			findings = append(findings, Finding{SeverityError, field + ".number", "spread number is required"})
		}
		if spread.Left.Text.Get("es") == "" && spread.Left.Text.Get("en") == "" {
			findings = append(findings, Finding{SeverityWarning, field + ".left.text", "spread has no text in any language"})
		}
		if spread.Right.ImagePath == "" && spread.Right.ImagePrompt == "" {
			findings = append(findings, Finding{SeverityWarning, field + ".right", "spread has neither image_path nor image_prompt"})
		}
	}

	return findings
}

func (v *BookValidator) checkImages(book *model.Book) []Finding {
	var findings []Finding

	checkPath := func(field, relPath string) {
		if relPath == "" {
			return
		}
		absPath := filepath.Join(book.RootPath, relPath)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			findings = append(findings, Finding{
				SeverityWarning, field,
				fmt.Sprintf("image file not found: %s", relPath),
			})
		}
	}

	checkPath("cover.image_path", book.Cover.ImagePath)
	checkPath("back_cover.image_path", book.BackCover.ImagePath)
	for i, spread := range book.Spreads {
		checkPath(fmt.Sprintf("spreads[%d].right.image_path", i), spread.Right.ImagePath)
	}

	return findings
}

func (v *BookValidator) checkCharacters(book *model.Book) []Finding {
	var findings []Finding

	for _, c := range book.ResolvedCharacters {
		if c.VisualPrompt == "" {
			findings = append(findings, Finding{
				SeverityInfo,
				fmt.Sprintf("characters[%s]", c.ID),
				"character has no visual_prompt — image consistency may suffer",
			})
		}
		if strings.TrimSpace(c.Description) == "" {
			findings = append(findings, Finding{
				SeverityInfo,
				fmt.Sprintf("characters[%s]", c.ID),
				"character has no description",
			})
		}
	}

	return findings
}

// checkTypography validates font configuration at book and spread level.
func (v *BookValidator) checkTypography(book *model.Book) []Finding {
	var findings []Finding

	if book.Font != nil {
		findings = append(findings, validateFontConfig("font", book.Font)...)
	}

	for i, spread := range book.Spreads {
		if spread.Left.Font != nil {
			findings = append(findings, validateFontConfig(
				fmt.Sprintf("spreads[%d].left.font", i), spread.Left.Font)...)
		}
		if spread.Right.Font != nil {
			findings = append(findings, validateFontConfig(
				fmt.Sprintf("spreads[%d].right.font", i), spread.Right.Font)...)
		}
		if spread.Left.TextBox != nil {
			findings = append(findings, validateTextBox(
				fmt.Sprintf("spreads[%d].left.text_box", i), spread.Left.TextBox)...)
		}
	}

	return findings
}

func validateFontConfig(field string, f *model.FontConfig) []Finding {
	var findings []Finding

	validStyles := map[model.FontStyle]bool{
		model.FontStyleNormal:    true,
		model.FontStyleBold:      true,
		model.FontStyleItalic:    true,
		model.FontStyleUnderline: true,
		model.FontStyleStrike:    true,
	}
	for _, s := range f.Style {
		if !validStyles[s] {
			findings = append(findings, Finding{
				SeverityError, field + ".style",
				fmt.Sprintf("invalid font style %q (valid: normal|bold|italic|underline|strikethrough)", s),
			})
		}
	}

	validAligns := map[model.TextAlign]bool{
		model.TextAlignLeft:   true,
		model.TextAlignCenter: true,
		model.TextAlignRight:  true,
	}
	if f.Align != "" && !validAligns[f.Align] {
		findings = append(findings, Finding{
			SeverityError, field + ".align",
			fmt.Sprintf("invalid text alignment %q (valid: left|center|right)", f.Align),
		})
	}

	if f.SizePt < 0 {
		findings = append(findings, Finding{
			SeverityError, field + ".size_pt",
			"font size must be a positive number",
		})
	}

	return findings
}

func validateTextBox(field string, tb *model.TextBox) []Finding {
	var findings []Finding
	if tb.X < 0 || tb.Y < 0 {
		findings = append(findings, Finding{
			SeverityError, field,
			"text_box x and y coordinates must be >= 0",
		})
	}
	if tb.Width < 0 || tb.Height < 0 {
		findings = append(findings, Finding{
			SeverityError, field,
			"text_box width and height must be >= 0",
		})
	}
	return findings
}