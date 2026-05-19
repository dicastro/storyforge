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

---

## Technology choices

| Choice        | Rationale                                                         |
|---------------|-------------------------------------------------------------------|
| Go            | Strong CLI ecosystem (cobra), fast binaries, easy distribution.   |
| cobra         | Standard Go CLI framework; provides subcommands and flag parsing.  |
| YAML (yaml.v3)| Human-editable content files. gopkg.in/yaml.v3 preserves structure.|
| No database   | All state lives in files. Git-friendly, no infra required.        |

---

## Package structure

```
main.go                         Entry point
cmd/                            CLI commands (cobra)
  root.go                       Root command + shared flags
  list.go                       storyforge list
  validate.go                   storyforge validate
  generate.go                   storyforge generate
  status.go                     storyforge status
  show.go                       storyforge show
internal/
  config/config.go              Content root resolution
  model/model.go                Domain types (Book, Saga, Character, …)
  repository/store.go           Filesystem reader (YAML → structs)
  validator/validator.go        Validation engine + Finding/Report types
  distributor/distributor.go    Distributor interface + registry
  distributor/amazon/kdp.go     Amazon KDP implementation
content/                        All book/saga/character data (not code)
```

---

## Key design decisions

### Distributor as a plugin
The `distributor.Distributor` interface makes adding a new platform
(IngramSpark, Draft2Digital, etc.) a matter of creating one new package
with no changes to the CLI core.

### Localised strings
`model.LocalisedString` is a `map[string]string` keyed by BCP-47 language
tags. Every piece of user-facing text in a book can have multiple language
versions. The `.Get(lang)` method falls back gracefully.

### Status lifecycle with versioned snapshots
When a book is `published`, the next `generate` run writes output to a
versioned subdirectory (e.g. `dist/amazon-kdp-v001/`) instead of overwriting.
This preserves artefacts for each published edition.

### YAML preservation on status updates
`cmd/status.go` uses `yaml.Node` (the low-level gopkg.in/yaml.v3 API) to
update a single field without re-marshalling the entire file. This preserves
comments, ordering and formatting.

### Guest characters
Characters that appear in only one book can be defined inline inside
`book.yaml` under `characters:` with a full description, rather than
requiring a separate file. They are merged with saga-level characters during
loading.

---

## YAML schema reference

### book.yaml top-level keys

| Key                | Type              | Required | Notes                                  |
|--------------------|-------------------|----------|----------------------------------------|
| `id`               | string            | yes      | Matches folder name                    |
| `saga_ref`         | object            | no       | `{saga_id, book_number}`               |
| `metadata`         | object            | yes      | title, author, language, status, tags  |
| `characters`       | list              | no       | Refs to saga chars + inline guests     |
| `format`           | object            | yes      | size, pages_interior, cover            |
| `distributor_hints`| object            | no       | Per-distributor overrides              |
| `cover`            | object            | yes      | image_path, image_prompt, title_style  |
| `back_cover`       | object            | no       |                                        |
| `front_matter`     | object            | no       | half_title, copyright, dedication      |
| `spreads`          | list              | yes      | Numbered left/right pairs              |
| `back_matter`      | object            | no       | closing page                           |

### spread structure

```yaml
spreads:
  - number: 1
    left:
      background_color: "very soft pastel blue"
      text:
        es: "Spanish text..."
        en: "English text..."
      short_text:
        es: "Short es"
        en: "Short en"
    right:
      image_path: assets/images/spread-01-right.png
      image_prompt: |
        Detailed prompt for image generation...
```

### character.yaml structure

```yaml
id: lucia
name: Lucía
age: 4
role: protagonist
description: |
  Narrative description of the character...
visual_prompt: |
  Detailed image generation prompt for consistent rendering...
personality:
  - observant
  - empathetic
relationships:
  - character_id: leo
    relation: cousin
    description: Optional description of the relationship
```

---

## What kind of help might be needed

- Adding new CLI commands (e.g. `storyforge notes`, `storyforge export-prompts`).
- Adding a new distributor adapter.
- Improving the validator (new rules, richer output).
- Adding shell autocompletion (cobra supports `__complete` out of the box via
  `rootCmd.GenBashCompletion`, `GenZshCompletion`, etc.).
- Adding an `init` command to scaffold a new saga or book from a template.
- Adding a `version` command.
- Writing tests (repository, validator, distributor are all easily unit-testable).

---

*Last updated: 2026*