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
	report.Findings = append(report.Findings, v.checkSpreads(book)...)
	report.Findings = append(report.Findings, v.checkImages(book)...)
	report.Findings = append(report.Findings, v.checkCharacters(book)...)

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

	if book.Format.Size == "" {
		findings = append(findings, Finding{SeverityWarning, "format.size", "book size is not specified"})
	}
	if book.Format.PagesInterior == 0 {
		findings = append(findings, Finding{SeverityWarning, "format.pages_interior", "number of interior pages is not set"})
	} else if book.Format.PagesInterior%2 != 0 {
		findings = append(findings, Finding{SeverityError, "format.pages_interior", "interior pages must be an even number"})
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

	// Warn if a character referenced in a spread prompt doesn't have a visual_prompt.
	charMap := make(map[string]*model.Character)
	for _, c := range book.ResolvedCharacters {
		charMap[c.ID] = c
	}
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