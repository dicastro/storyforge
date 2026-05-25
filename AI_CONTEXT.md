# AI_CONTEXT.md — Storyforge project

> This file gives an AI assistant full context about the storyforge project:
> what it does, how it is structured, design decisions, and what kind of help
> might be needed. Paste this file when asking for code generation, refactoring,
> or architecture advice.

---

## What is storyforge?

Storyforge is a Go CLI tool for managing picture books intended for publication
via Amazon KDP (and potentially other distributors in the future).

It is **not** a tool that generates story content or images. It:
- Reads YAML files from disk (the "database").
- Validates books against structural and distributor-specific rules.
- Generates submission-ready artefacts (JSON manifests, Markdown checklists,
  image prompt sheets).
- Tracks book lifecycle (draft → wip → ready → published).
- Provides a structured home for story notes and AI briefing documents.
- Supports multiple **publication targets** per book (distributor × size × language × binding).

---

## Technology choices

| Choice        | Rationale                                                          |
|---------------|--------------------------------------------------------------------|
| Go            | Strong CLI ecosystem (cobra), fast binaries, easy distribution.    |
| cobra         | Standard Go CLI framework; provides subcommands, flag parsing, and shell autocompletion. |
| YAML (yaml.v3)| Human-editable content files. gopkg.in/yaml.v3 preserves structure.|
| embed         | Distributor spec YAML files are embedded in the binary at compile time. |
| No database   | All state lives in files. Git-friendly, no infra required.         |

---

## Package structure

```
main.go                              Entry point
cmd/                                 CLI commands (cobra)
  root.go                            Root command + shared flags
  list.go                            storyforge list
  validate.go                        storyforge validate
  generate.go                        storyforge generate
  status.go                          storyforge status
  show.go                            storyforge show
internal/
  config/config.go                   Content root resolution
  model/model.go                     Domain types (Book, Saga, Character, PublicationTarget, FontConfig, TextBox, …)
  repository/store.go                Filesystem reader (YAML → structs)
  validator/validator.go             Validation engine + Finding/Report types
  distributor/distributor.go         Distributor interface + registry
  distributor/amazon/
    kdp.go                           Amazon KDP implementation
    kdp-specs.yaml                   Physical specs (sizes, bindings, paper, thickness) — embedded
    specs.go                         Go types for kdp-specs.yaml
  imaging/mock.go                    Mock image generation + resolution validation helpers
  pdf/                               (planned) PDF assembly for interior and cover
content/                             All book/saga/character data (not code)
```

---

## Key design decisions

### Publication targets
A book declares a list of `publication_targets`, each combining a distributor,
size, language, binding, paper type, and interior colour. Every target produces
its own independent output directory under `dist/`. The distributor validates
at generation time whether it supports the requested combination.

### Distributor as a plugin
The `distributor.Distributor` interface makes adding a new platform
(IngramSpark, Draft2Digital, etc.) a matter of creating one new package.
No changes to the CLI core are required.

### Distributor specs in YAML
Each distributor ships a `<name>-specs.yaml` file embedded in the binary
via `//go:embed`. Specs define supported sizes, bindings, paper types, page
count limits, and paper thickness (for spine width calculation). Updating
specs does not require changing Go code.

### Spine width calculation
Cover spine width = `pages × thickness_per_page_inches + cover_board_inches`.
Both constants come from the distributor's spec file and vary by binding and
paper type.

### Typography
Font settings are configured at two levels:
- **Book level** (`font:` in `book.yaml`) — default for all spreads.
- **Spread level** (`left.font:` or `right.font:`) — overrides for a specific page side.

Text values support inline HTML markup (`<b>`, `<i>`, `<u>`, `<s>`,
`<span style="color:#rrggbb">`, `<br>`) for rich formatting without a
separate templating language.

Paragraph breaks from YAML literal block scalars (`|`) are preserved.

### Text box positioning
A `text_box:` block on a spread side positions the text area in millimetres
relative to the top-left corner of the page (excluding bleed). Omitting it
uses the full page with standard margins.

### Mock images
`--mock-images` on `generate` writes white PNG placeholders (with a diagonal
grey cross) for any referenced image path that does not exist on disk. Sized
to the exact print dimensions at 300 DPI. Real files are never overwritten.

### Localised strings
`model.LocalisedString` is a `map[string]string` keyed by BCP-47 language
tags. Every piece of user-facing text in a book can have multiple language
versions. The `.Get(lang)` method falls back gracefully to `"es"` then any.

### Status lifecycle with versioned snapshots
When a book is `published`, the next `generate` run writes output to a
versioned subdirectory (e.g. `dist/amazon-kdp/21x21cm-es-paperback-v001/`)
instead of overwriting, preserving artefacts for each published edition.

### YAML preservation on status updates
`cmd/status.go` uses `yaml.Node` (the low-level gopkg.in/yaml.v3 API) to
update a single field without re-marshalling the entire file. This preserves
comments, ordering and formatting.

### Guest characters
Characters that appear in only one book can be defined inline inside
`book.yaml` under `characters:`. They are merged with saga-level characters
during loading.

---

## YAML schema reference

### book.yaml top-level keys

| Key                   | Type              | Required | Notes                                    |
|-----------------------|-------------------|----------|------------------------------------------|
| `id`                  | string            | yes      | Matches folder name                      |
| `saga_ref`            | object            | no       | `{saga_id, book_number}`                 |
| `metadata`            | object            | yes      | title, author, language, status, tags    |
| `font`                | object            | no       | Book-level default typography            |
| `characters`          | list              | no       | Refs to saga chars + inline guests       |
| `format`              | object            | yes      | size, pages_interior, cover              |
| `publication_targets` | list              | yes      | One entry per distributor/size/lang/binding |
| `cover`               | object            | yes      | image_path, image_prompt, title_style    |
| `back_cover`          | object            | no       |                                          |
| `front_matter`        | object            | no       | half_title, copyright, dedication        |
| `spreads`             | list              | yes      | Numbered left/right pairs                |
| `back_matter`         | object            | no       | closing page                             |

### publication_target fields

| Field            | Type   | Required | Values                                  |
|------------------|--------|----------|-----------------------------------------|
| `distributor`    | string | yes      | Registered distributor id, e.g. `amazon-kdp` |
| `size`           | string | yes      | e.g. `21x21cm`, `6x9`                  |
| `language`       | string | yes      | BCP-47, e.g. `es`, `en`                |
| `binding`        | string | yes      | `paperback` \| `hardcover`              |
| `paper`          | string | yes      | `white` \| `cream` \| `color`          |
| `interior_color` | string | yes      | `full_color` \| `black_white`           |
| `options`        | map    | no       | Distributor-specific free-form overrides |

### font fields

| Field                  | Type     | Notes                                         |
|------------------------|----------|-----------------------------------------------|
| `family`               | string   | Font family name, e.g. `Rounded`              |
| `size_pt`              | float    | Font size in points                           |
| `color`                | string   | CSS hex color, e.g. `#333333`                 |
| `style`                | list     | `normal` \| `bold` \| `italic` \| `underline` \| `strikethrough` |
| `align`                | string   | `left` \| `center` \| `right`                |
| `line_height_pct`      | int      | Line height as % of font size (default 120)   |
| `paragraph_spacing_pt` | float    | Extra space between paragraphs in points      |

### text_box fields (millimetres from top-left of page, excluding bleed)

| Field    | Type  | Notes                              |
|----------|-------|------------------------------------|
| `x`      | float | Distance from left edge (mm)       |
| `y`      | float | Distance from top edge (mm)        |
| `width`  | float | 0 = extend to right margin         |
| `height` | float | 0 = extend to bottom margin        |

### spread structure

```yaml
spreads:
  - number: 1
    left:
      background_color: "very soft pastel blue"
      font:                          # optional spread-level override
        size_pt: 16
        align: center
        style: [italic]
      text_box:                      # optional text positioning (mm)
        x: 10
        y: 40
        width: 0
        height: 0
      text:
        es: |
          Spanish text here. Supports <b>bold</b> and <i>italic</i> inline.
          Blank lines between paragraphs are preserved.
        en: "English text here."
      short_text:
        es: "Short es"
        en: "Short en"
    right:
      image_path: assets/images/spread-01-right.png
      image_prompt: |
        Detailed prompt for image generation...
```

---

## Planned work

- `internal/pdf/` — PDF assembly for interior and cover (two PDFs per target).
  - Interior PDF: all spreads, with bleed, at correct size.
  - Cover PDF: front + spine + back, width = 2×page + spine, calculated per target.
- Image resolution validation against 300 DPI requirement (`imaging.ValidateResolution`).
- Image cropping/scaling from a master high-res source to each target size.
- Shell autocompletion (cobra supports `__complete` natively via
  `rootCmd.GenBashCompletionFile`, `GenZshCompletionFile`, etc.).
- `storyforge init` command to scaffold a new saga or book from a template.
- `storyforge version` command.
- Unit tests for repository, validator, and distributor packages.

---

*Last updated: 2026*