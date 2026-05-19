// Package model defines the core domain types for storyforge.
// All structs map directly to the YAML file schema.
package model

// BookStatus represents the lifecycle state of a book.
type BookStatus string

const (
	StatusDraft     BookStatus = "draft"
	StatusWIP       BookStatus = "wip"
	StatusReady     BookStatus = "ready"
	StatusPublished BookStatus = "published"
)

// LocalisedString holds a piece of text in multiple languages.
// Keys are BCP-47 language tags (e.g. "es", "en").
type LocalisedString map[string]string

// Get returns the value for the given language, falling back to "es" then any.
func (l LocalisedString) Get(lang string) string {
	if v, ok := l[lang]; ok {
		return v
	}
	if v, ok := l["es"]; ok {
		return v
	}
	for _, v := range l {
		return v
	}
	return ""
}

// ---- Character ----------------------------------------------------------------

// Relationship describes how two characters are connected.
type Relationship struct {
	CharacterID string `yaml:"character_id"`
	Relation    string `yaml:"relation"`
	Description string `yaml:"description,omitempty"`
}

// Character holds the canonical description of a recurring person.
type Character struct {
	ID           string         `yaml:"id"`
	Name         string         `yaml:"name"`
	Age          int            `yaml:"age,omitempty"`
	Role         string         `yaml:"role,omitempty"`
	Description  string         `yaml:"description,omitempty"`
	VisualPrompt string         `yaml:"visual_prompt,omitempty"`
	Personality  []string       `yaml:"personality,omitempty"`
	Traits       map[string]any `yaml:"traits,omitempty"`
	Relationships []Relationship `yaml:"relationships,omitempty"`
}

// ---- Saga --------------------------------------------------------------------

// SagaBookRef is a lightweight reference used inside saga.yaml.
type SagaBookRef struct {
	ID     string     `yaml:"id"`
	Title  LocalisedString `yaml:"title"`
	Number int        `yaml:"number"`
	Status BookStatus `yaml:"status"`
}

// Saga groups books and shared characters under a common narrative.
type Saga struct {
	ID              string          `yaml:"id"`
	Title           LocalisedString `yaml:"titles"`
	Description     string          `yaml:"description,omitempty"`
	TargetAge       string          `yaml:"target_age,omitempty"`
	LanguageDefault string          `yaml:"language_default,omitempty"`
	Characters      []string        `yaml:"characters"`
	Books           []SagaBookRef   `yaml:"books"`
	Themes          []string        `yaml:"themes,omitempty"`
	Notes           string          `yaml:"notes,omitempty"`

	// Resolved after loading — not in YAML.
	ResolvedCharacters []*Character `yaml:"-"`
}

// ---- Book --------------------------------------------------------------------

// SagaRef links a book back to its saga.
type SagaRef struct {
	SagaID     string `yaml:"saga_id"`
	BookNumber int    `yaml:"book_number"`
}

// BookMetadata contains editorial information about a book.
type BookMetadata struct {
	Title           LocalisedString `yaml:"title"`
	Language        string          `yaml:"language"`
	Author          string          `yaml:"author"`
	Edition         int             `yaml:"edition,omitempty"`
	Status          BookStatus      `yaml:"status"`
	Tags            []string        `yaml:"tags,omitempty"`
}

// BookFormat describes physical production parameters.
type BookFormat struct {
	Size          string `yaml:"size"`
	PagesInterior int    `yaml:"pages_interior"`
	Cover         string `yaml:"cover,omitempty"`
}

// DistributorHints holds optional overrides per distributor.
type DistributorHints struct {
	AmazonKDP map[string]any `yaml:"amazon_kdp,omitempty"`
}

// CoverStyle holds typographic preferences for cover text.
type CoverStyle struct {
	Font  string `yaml:"font,omitempty"`
	Color string `yaml:"color,omitempty"`
}

// CoverPage describes both front and back cover content.
type CoverPage struct {
	ImagePath   string          `yaml:"image_path,omitempty"`
	ImagePrompt string          `yaml:"image_prompt,omitempty"`
	TitleStyle  CoverStyle      `yaml:"title_style,omitempty"`
	BackgroundColor string      `yaml:"background_color,omitempty"`
	Summary     LocalisedString `yaml:"summary,omitempty"`
}

// FrontMatterPage represents a single courtesy/front-matter page.
type FrontMatterPage struct {
	BackgroundColor string          `yaml:"background_color,omitempty"`
	Title           LocalisedString `yaml:"title,omitempty"`
	Text            LocalisedString `yaml:"text,omitempty"`
}

// FrontMatter groups the non-story pages at the start of a book.
type FrontMatter struct {
	HalfTitle   FrontMatterPage `yaml:"half_title,omitempty"`
	Copyright   FrontMatterPage `yaml:"copyright,omitempty"`
	Dedication  FrontMatterPage `yaml:"dedication,omitempty"`
}

// SpreadSide is the content of one side (left or right) of a spread.
type SpreadSide struct {
	BackgroundColor string          `yaml:"background_color,omitempty"`
	Text            LocalisedString `yaml:"text,omitempty"`
	ShortText       LocalisedString `yaml:"short_text,omitempty"`
	ImagePath       string          `yaml:"image_path,omitempty"`
	ImagePrompt     string          `yaml:"image_prompt,omitempty"`
}

// Spread represents a two-page spread (left text + right illustration).
type Spread struct {
	Number int        `yaml:"number"`
	Left   SpreadSide `yaml:"left"`
	Right  SpreadSide `yaml:"right"`
}

// GuestCharacter is a character defined inline inside a book (not in a character file).
type GuestCharacter struct {
	ID           string `yaml:"id"`
	Name         string `yaml:"name"`
	Role         string `yaml:"role,omitempty"`
	Description  string `yaml:"description,omitempty"`
	VisualPrompt string `yaml:"visual_prompt,omitempty"`
}

// BookCharacterRef is used inside book.yaml — either an id string or an inline guest.
type BookCharacterRef struct {
	ID          string `yaml:"id,omitempty"`
	Name        string `yaml:"name,omitempty"`
	Role        string `yaml:"role,omitempty"`
	Description string `yaml:"description,omitempty"`
	VisualPrompt string `yaml:"visual_prompt,omitempty"`
}

// BackMatter groups the closing pages.
type BackMatter struct {
	Closing FrontMatterPage `yaml:"closing,omitempty"`
}

// Book is the complete representation of a picture book.
type Book struct {
	ID               string           `yaml:"id"`
	SagaRef          *SagaRef         `yaml:"saga_ref,omitempty"`
	Metadata         BookMetadata     `yaml:"metadata"`
	Characters       []BookCharacterRef `yaml:"characters,omitempty"`
	Format           BookFormat       `yaml:"format"`
	DistributorHints DistributorHints `yaml:"distributor_hints,omitempty"`
	Cover            CoverPage        `yaml:"cover"`
	BackCover        CoverPage        `yaml:"back_cover,omitempty"`
	FrontMatter      FrontMatter      `yaml:"front_matter,omitempty"`
	Spreads          []Spread         `yaml:"spreads"`
	BackMatter       BackMatter       `yaml:"back_matter,omitempty"`

	// Resolved after loading — not in YAML.
	RootPath           string       `yaml:"-"`
	ResolvedCharacters []*Character `yaml:"-"`
	Saga               *Saga        `yaml:"-"`
}

// IsPublished returns true when the book has been published.
func (b *Book) IsPublished() bool {
	return b.Metadata.Status == StatusPublished
}

// MissingImages returns a list of expected image paths that do not have a file yet.
// It only checks paths that are explicitly set (non-empty).
func (b *Book) MissingImages() []string {
	var missing []string
	check := func(path string) {
		if path != "" {
			missing = append(missing, path)
		}
	}
	check(b.Cover.ImagePath)
	check(b.BackCover.ImagePath)
	for _, s := range b.Spreads {
		check(s.Right.ImagePath)
	}
	return missing
}