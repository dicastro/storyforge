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

// BindingType represents the physical binding of a book.
type BindingType string

const (
	BindingPaperback BindingType = "paperback"
	BindingHardcover BindingType = "hardcover"
)

// PaperType represents the paper stock used for interior pages.
type PaperType string

const (
	PaperWhite PaperType = "white"
	PaperCream PaperType = "cream"
	PaperColor PaperType = "color"
)

// InteriorColor represents whether the interior is printed in color or B&W.
type InteriorColor string

const (
	InteriorFullColor  InteriorColor = "full_color"
	InteriorBlackWhite InteriorColor = "black_white"
)

// FontStyle controls typographic decoration on a run of text.
type FontStyle string

const (
	FontStyleNormal    FontStyle = "normal"
	FontStyleBold      FontStyle = "bold"
	FontStyleItalic    FontStyle = "italic"
	FontStyleUnderline FontStyle = "underline"
	FontStyleStrike    FontStyle = "strikethrough"
)

// TextAlign controls horizontal alignment of a text block.
type TextAlign string

const (
	TextAlignLeft   TextAlign = "left"
	TextAlignCenter TextAlign = "center"
	TextAlignRight  TextAlign = "right"
)

// LocalisedString holds a piece of text in multiple languages.
// Keys are BCP-47 language tags (e.g. "es", "en").
// Values may contain inline HTML tags for rich formatting:
// <b>, <i>, <u>, <s>, <span style="color:#rrggbb">, <br>.
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

// ---- Typography --------------------------------------------------------------

// FontConfig defines typographic properties for a block of text.
// All fields are optional; unset fields inherit from the book-level default.
type FontConfig struct {
	// Family is the font family name, e.g. "Roboto", "OpenDyslexic".
	Family string `yaml:"family,omitempty"`
	// SizePt is the font size in points.
	SizePt float64 `yaml:"size_pt,omitempty"`
	// Color is a CSS hex color string, e.g. "#2a6496".
	Color string `yaml:"color,omitempty"`
	// Style lists one or more decorations: bold, italic, underline, strikethrough.
	// Multiple values are combined: ["bold", "italic"].
	Style []FontStyle `yaml:"style,omitempty"`
	// Align controls horizontal text alignment within the text box.
	Align TextAlign `yaml:"align,omitempty"`
	// LineHeightPct is the line height as a percentage of font size (default 120).
	LineHeightPct int `yaml:"line_height_pct,omitempty"`
	// ParagraphSpacingPt is extra space added between paragraphs, in points.
	ParagraphSpacingPt float64 `yaml:"paragraph_spacing_pt,omitempty"`
}

// TextBox defines the position and size of a text area on a page.
// All measurements are in millimetres, relative to the top-left corner
// of the page (excluding bleed). Zero values mean "use the full page margin".
type TextBox struct {
	// X is the distance from the left edge of the page (mm).
	X float64 `yaml:"x,omitempty"`
	// Y is the distance from the top edge of the page (mm).
	Y float64 `yaml:"y,omitempty"`
	// Width of the text box (mm). 0 = extend to right margin.
	Width float64 `yaml:"width,omitempty"`
	// Height of the text box (mm). 0 = extend to bottom margin.
	Height float64 `yaml:"height,omitempty"`
}

// ---- Character ---------------------------------------------------------------

// Relationship describes how two characters are connected.
type Relationship struct {
	CharacterID string `yaml:"character_id"`
	Relation    string `yaml:"relation"`
	Description string `yaml:"description,omitempty"`
}

// Character holds the canonical description of a recurring person.
type Character struct {
	ID            string         `yaml:"id"`
	Name          string         `yaml:"name"`
	Age           int            `yaml:"age,omitempty"`
	Role          string         `yaml:"role,omitempty"`
	Description   string         `yaml:"description,omitempty"`
	VisualPrompt  string         `yaml:"visual_prompt,omitempty"`
	Personality   []string       `yaml:"personality,omitempty"`
	Traits        map[string]any `yaml:"traits,omitempty"`
	Relationships []Relationship `yaml:"relationships,omitempty"`
}

// ---- Saga --------------------------------------------------------------------

// SagaBookRef is a lightweight reference used inside saga.yaml.
type SagaBookRef struct {
	ID     string          `yaml:"id"`
	Title  LocalisedString `yaml:"title"`
	Number int             `yaml:"number"`
	Status BookStatus      `yaml:"status"`
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
	Title    LocalisedString `yaml:"title"`
	Language string          `yaml:"language"`
	Author   string          `yaml:"author"`
	Edition  int             `yaml:"edition,omitempty"`
	Status   BookStatus      `yaml:"status"`
	Tags     []string        `yaml:"tags,omitempty"`
}

// BookFormat describes physical production parameters shared across all targets.
type BookFormat struct {
	// Size is the default page size, e.g. "21x21cm". Can be overridden per target.
	Size string `yaml:"size"`
	// PagesInterior is the total number of interior pages (must be even).
	PagesInterior int `yaml:"pages_interior"`
	// Cover type hint, e.g. "extended".
	Cover string `yaml:"cover,omitempty"`
}

// PublicationTarget describes one specific edition of a book:
// a combination of distributor, physical format, language, and paper options.
// Whether a distributor supports a given combination is enforced at validation time.
type PublicationTarget struct {
	// Distributor is the registered distributor id, e.g. "amazon-kdp".
	Distributor string `yaml:"distributor"`
	// Size overrides BookFormat.Size for this target, e.g. "21x21cm".
	Size string `yaml:"size"`
	// Language is the BCP-47 content language for this edition, e.g. "es".
	Language string `yaml:"language"`
	// Binding is the physical binding type: paperback or hardcover.
	Binding BindingType `yaml:"binding"`
	// Paper is the paper stock: white, cream, or color.
	Paper PaperType `yaml:"paper"`
	// InteriorColor indicates full_color or black_white printing.
	InteriorColor InteriorColor `yaml:"interior_color"`
	// Options holds distributor-specific properties not covered by common fields.
	Options map[string]any `yaml:"options,omitempty"`
}

// CoverStyle holds typographic preferences for cover text.
type CoverStyle struct {
	Font  string `yaml:"font,omitempty"`
	Color string `yaml:"color,omitempty"`
}

// CoverPage describes both front and back cover content.
type CoverPage struct {
	ImagePath       string          `yaml:"image_path,omitempty"`
	ImagePrompt     string          `yaml:"image_prompt,omitempty"`
	TitleStyle      CoverStyle      `yaml:"title_style,omitempty"`
	BackgroundColor string          `yaml:"background_color,omitempty"`
	Summary         LocalisedString `yaml:"summary,omitempty"`
}

// FrontMatterPage represents a single courtesy/front-matter page.
type FrontMatterPage struct {
	BackgroundColor string          `yaml:"background_color,omitempty"`
	Title           LocalisedString `yaml:"title,omitempty"`
	Text            LocalisedString `yaml:"text,omitempty"`
}

// FrontMatter groups the non-story pages at the start of a book.
type FrontMatter struct {
	HalfTitle  FrontMatterPage `yaml:"half_title,omitempty"`
	Copyright  FrontMatterPage `yaml:"copyright,omitempty"`
	Dedication FrontMatterPage `yaml:"dedication,omitempty"`
}

// SpreadSide is the content of one side (left or right) of a spread.
type SpreadSide struct {
	BackgroundColor string          `yaml:"background_color,omitempty"`
	// Text contains the full text for this side, optionally with inline HTML
	// markup for rich formatting: <b>, <i>, <u>, <s>, <span style="color:#rrggbb">, <br>.
	// Newlines in the YAML literal block scalar are preserved as paragraph breaks.
	Text      LocalisedString `yaml:"text,omitempty"`
	ShortText LocalisedString `yaml:"short_text,omitempty"`
	ImagePath   string          `yaml:"image_path,omitempty"`
	ImagePrompt string          `yaml:"image_prompt,omitempty"`
	// Font overrides the book-level font for this side only.
	Font *FontConfig `yaml:"font,omitempty"`
	// TextBox controls the position and size of the text area on the page.
	// If omitted, text fills the page with standard margins.
	TextBox *TextBox `yaml:"text_box,omitempty"`
}

// Spread represents a two-page spread (left text + right illustration).
type Spread struct {
	Number int        `yaml:"number"`
	Left   SpreadSide `yaml:"left"`
	Right  SpreadSide `yaml:"right"`
}

// BookCharacterRef is used inside book.yaml — either an id string or an inline guest.
type BookCharacterRef struct {
	ID           string `yaml:"id,omitempty"`
	Name         string `yaml:"name,omitempty"`
	Role         string `yaml:"role,omitempty"`
	Description  string `yaml:"description,omitempty"`
	VisualPrompt string `yaml:"visual_prompt,omitempty"`
}

// BackMatter groups the closing pages.
type BackMatter struct {
	Closing FrontMatterPage `yaml:"closing,omitempty"`
}

// Book is the complete representation of a picture book.
type Book struct {
	ID       string       `yaml:"id"`
	SagaRef  *SagaRef     `yaml:"saga_ref,omitempty"`
	Metadata BookMetadata `yaml:"metadata"`
	// Font is the book-level default typography. All spreads inherit this
	// unless they define their own font override.
	Font               *FontConfig          `yaml:"font,omitempty"`
	Characters         []BookCharacterRef   `yaml:"characters,omitempty"`
	Format             BookFormat           `yaml:"format"`
	PublicationTargets []PublicationTarget  `yaml:"publication_targets,omitempty"`
	Cover              CoverPage            `yaml:"cover"`
	BackCover          CoverPage            `yaml:"back_cover,omitempty"`
	FrontMatter        FrontMatter          `yaml:"front_matter,omitempty"`
	Spreads            []Spread             `yaml:"spreads"`
	BackMatter         BackMatter           `yaml:"back_matter,omitempty"`

	// Resolved after loading — not in YAML.
	RootPath           string       `yaml:"-"`
	ResolvedCharacters []*Character `yaml:"-"`
	Saga               *Saga        `yaml:"-"`
}

// IsPublished returns true when the book has been published.
func (b *Book) IsPublished() bool {
	return b.Metadata.Status == StatusPublished
}

// EffectiveSizeForTarget returns the page size to use for a given target,
// falling back to the book-level format size if the target does not override it.
func (b *Book) EffectiveSizeForTarget(t PublicationTarget) string {
	if t.Size != "" {
		return t.Size
	}
	return b.Format.Size
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