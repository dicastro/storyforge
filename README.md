# Storyforge

A CLI for managing, validating and generating publication materials for
picture books sold through Amazon KDP and other distributors.

All content lives in YAML files on disk — no database required.

---

## Project goals

- Manage one or more **sagas** (series of books sharing characters and world).
- Maintain **character consistency** via shared YAML files outside individual books.
- **Validate** books against structural rules and distributor-specific requirements.
- **Generate** submission-ready materials (manifests, checklists, image prompt sheets).
- Track book **lifecycle status**: draft → wip → ready → published.
- Store **notes and story seeds** alongside each book for use with AI assistants.
- Support **multiple publication targets** per book: different distributors, sizes, languages, and bindings each produce independent output artefacts.

---

## Installation

```bash
git clone https://github.com/dicastro/storyforge
cd storyforge
go build -o storyforge .
```

Move the binary to somewhere on your `$PATH`:

```bash
mv storyforge /usr/local/bin/
```

---

## Quick start

```bash
# List all books
storyforge list books

# Show details of a specific book
storyforge show book-01 --saga lucias-adventures

# Validate all targets
storyforge validate book-01 --saga lucias-adventures

# Validate only Amazon KDP targets
storyforge validate book-01 --saga lucias-adventures --distributor amazon-kdp

# Generate all publication targets
storyforge generate book-01 --saga lucias-adventures

# Generate a single target
storyforge generate book-01 --saga lucias-adventures \
  --distributor amazon-kdp --size 21x21cm --language es

# Preview layout with placeholder images (no real images needed)
storyforge generate book-01 --saga lucias-adventures --mock-images

# Update status
storyforge status book-01 ready --saga lucias-adventures --note "All images approved"
```

---

## Content directory structure

```
content/
├── sagas/
│   └── <saga-id>/
│       ├── saga.yaml           # Saga metadata, book list, themes
│       ├── CONTEXT.md          # AI briefing document for the saga
│       ├── characters/
│       │   ├── <character-id>.yaml
│       │   └── ...
│       └── books/
│           └── <book-id>/
│               ├── book.yaml           # Main book definition
│               ├── notes.md            # Story seeds, anecdotes, AI prompts
│               ├── assets/
│               │   └── images/         # Cover and spread images go here
│               └── dist/               # Generated output (git-ignored)
│                   └── amazon-kdp/
│                       └── 21x21cm-es-paperback/
│                           ├── kdp-manifest.json
│                           ├── kdp-checklist.md
│                           └── image-prompts.md
├── standalone/
│   └── <book-id>/              # Books not belonging to any saga
└── characters/
    └── <character-id>.yaml     # Characters shared across sagas
```

---

## Book status lifecycle

| Status      | Meaning                                              |
|-------------|------------------------------------------------------|
| `draft`     | Structure may be incomplete. Active editing.         |
| `wip`       | Content and images in progress.                      |
| `ready`     | Complete and validated. Ready for distributor.       |
| `published` | Submitted. Next `generate` creates a versioned copy. |

Transitioning to **published** is reversible (with confirmation) to support
new editions without losing the original generated artefacts.

---

## Publication targets

A book can have multiple publication targets, each producing its own
independent set of output files. Targets are defined in `book.yaml`:

```yaml
publication_targets:
  - distributor: amazon-kdp
    size: 21x21cm
    language: es
    binding: paperback
    paper: white
    interior_color: full_color

  - distributor: amazon-kdp
    size: 21x21cm
    language: en
    binding: paperback
    paper: white
    interior_color: full_color

  - distributor: amazon-kdp
    size: 6x9
    language: es
    binding: hardcover
    paper: white
    interior_color: full_color
```

**Physical properties** (`size`, `binding`, `paper`, `interior_color`) are
properties of the book edition itself, independent of the distributor.
The distributor validates at generation time whether it supports that
combination. If not, a validation error is raised.

**Output directory** per target:

```
dist/<distributor>/<size>-<language>-<binding>/
```

For example:
```
dist/amazon-kdp/21x21cm-es-paperback/
dist/amazon-kdp/21x21cm-en-paperback/
```

---

## Typography

Font settings can be configured at two levels:

### Book level (default for all spreads)

```yaml
font:
  family: Rounded
  size_pt: 14
  color: "#333333"
  align: left
  line_height_pct: 130
  paragraph_spacing_pt: 8
```

### Spread level (override for a specific page)

```yaml
spreads:
  - number: 10
    left:
      font:
        size_pt: 16
        align: center
        style: [italic]
```

Valid `style` values: `normal`, `bold`, `italic`, `underline`, `strikethrough`.
Multiple styles can be combined: `style: [bold, italic]`.

### Inline rich text

Text values support a subset of HTML for inline formatting:

```yaml
text:
  es: |
    Lucía abre su <b>cuaderno mágico</b> y señala el dibujo.
    Las palabras se esconden <i>muy profundo</i> en su barriga.
    <span style="color:#2a6496">¡El castillo ya tiene una carretera secreta!</span>
```

Supported tags: `<b>`, `<i>`, `<u>`, `<s>`, `<span style="color:#rrggbb">`, `<br>`.

Paragraph breaks (blank lines in the YAML literal block scalar `|`) are
rendered with the configured `paragraph_spacing_pt`.

### Text box positioning

By default text fills the page with standard margins. To position text
precisely, add a `text_box` block (all values in millimetres, relative
to the top-left corner of the page, excluding bleed):

```yaml
left:
  text_box:
    x: 10       # distance from left edge (mm)
    y: 40       # distance from top edge (mm)
    width: 0    # 0 = extend to right margin
    height: 0   # 0 = extend to bottom margin
```

---

## Mock images

When images are not yet available, you can generate white placeholder images
(with a diagonal cross) sized to the exact print dimensions at 300 DPI:

```bash
storyforge generate book-01 --saga lucias-adventures --mock-images
```

Mock images are written next to the real image paths. They are **not**
overwritten if the real file already exists.

---

## Distributor specs

Each distributor has a YAML spec file that defines supported sizes, bindings,
paper types, page count limits, and paper thickness for spine calculations.
No recompile is needed to update specs.

Location: `internal/distributor/<name>/<name>-specs.yaml`

Amazon KDP currently supports these sizes:
`5x8`, `5.5x8.5`, `6x9`, `7x10`, `8x8`, `8.5x8.5`, `8.5x11`, `21x21cm`

Hardcover is supported for: `5.5x8.5`, `6x9`, `7x10`, `8.5x11`.

---

## Adding a new distributor

1. Create `internal/distributor/<name>/`.
2. Add a `<name>-specs.yaml` with the distributor's physical constraints.
3. Implement the `distributor.Distributor` interface:
   - `Name() string`
   - `DisplayName() string`
   - `ValidationRules() []validator.DistributorRule`
   - `Generate(book, target, outputDir, opts) (*GenerationResult, error)`
4. Call `distributor.Register(&YourDistributor{})` from `init()`.
5. Add a blank import in `cmd/validate.go` and `cmd/generate.go`.

No changes to core CLI code are required.

---

## Environment variables

| Variable             | Description                                       |
|----------------------|---------------------------------------------------|
| `STORYFORGE_CONTENT` | Path to the content directory. Overrides `--content`. |

---

## Working with AI assistants

Each saga contains a `CONTEXT.md` file designed to be pasted into an AI
assistant before asking it to write or revise story content. Each book has a
`notes.md` for free-form anecdotes and story seeds.

Typical workflow:

1. Open `CONTEXT.md` + the relevant `book.yaml` + `notes.md`.
2. Paste all three into your AI assistant.
3. Ask it to write a new spread, revise text, or generate an image prompt.
4. Copy the result back into `book.yaml`.

---

## Development

```bash
go mod tidy
go build ./...
go test ./...
go vet ./...
```

All business logic lives under `internal/`; command definitions under `cmd/`.

---

## License

Copyright © 2026 Diego Castro Viadero. All rights reserved.