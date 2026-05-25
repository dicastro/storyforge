// Package distributor defines the interface that every publishing platform
// adapter must implement. Adding a new distributor means creating a new package
// that satisfies this interface — no changes to core CLI code are required.
package distributor

import (
	"github.com/dicastro/storyforge/internal/model"
	"github.com/dicastro/storyforge/internal/validator"
)

// Distributor is the contract every publishing platform must satisfy.
type Distributor interface {
	// Name returns the short identifier for this distributor (e.g. "amazon-kdp").
	Name() string

	// DisplayName returns a human-readable name (e.g. "Amazon KDP").
	DisplayName() string

	// ValidationRules returns extra validation rules specific to this distributor.
	// These are injected into the BookValidator before a validate or generate run.
	ValidationRules() []validator.DistributorRule

	// Generate produces all output files required for submission and writes
	// them into outputDir. It returns a GenerationResult describing what was
	// created and any warnings encountered.
	//
	// target specifies which publication target to generate for.
	// opts controls optional generation behaviour (e.g. mock images).
	Generate(book *model.Book, target model.PublicationTarget, outputDir string, opts GenerateOptions) (*GenerationResult, error)
}

// GenerateOptions controls optional behaviour during generation.
type GenerateOptions struct {
	// MockImages replaces missing or all images with a white placeholder bearing
	// a diagonal cross, sized to the exact required dimensions. Useful for
	// previewing PDF layout before real images are ready.
	MockImages bool
}

// GeneratedFile describes a single output file produced during generation.
type GeneratedFile struct {
	// RelativePath is the path relative to outputDir.
	RelativePath string
	// Description explains what the file is for.
	Description string
}

// GenerationResult summarises the outcome of a Generate call.
type GenerationResult struct {
	Distributor string
	BookID      string
	Target      model.PublicationTarget
	OutputDir   string
	Files       []GeneratedFile
	Warnings    []string
}

// Registry maps distributor name → implementation.
var Registry = map[string]Distributor{}

// Register adds a distributor to the global registry.
// Call this from your distributor package's init() function.
func Register(d Distributor) {
	Registry[d.Name()] = d
}

// Get retrieves a distributor by name.
func Get(name string) (Distributor, bool) {
	d, ok := Registry[name]
	return d, ok
}

// Names returns all registered distributor names.
func Names() []string {
	names := make([]string, 0, len(Registry))
	for n := range Registry {
		names = append(names, n)
	}
	return names
}