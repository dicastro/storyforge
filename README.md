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
storyforge show book-01 --saga lucas-adventures

# Validate before submitting
storyforge validate book-01 --saga lucas-adventures --distributor amazon-kdp

# Generate submission materials
storyforge generate book-01 --saga lucas-adventures --distributor amazon-kdp

# Update status
storyforge status book-01 ready --saga lucas-adventures --note "All 12 images approved"
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
├── standalone/
│   └── <book-id>/              # Books not belonging to any saga
│       ├── book.yaml
│       ├── notes.md
│       └── assets/
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

## Adding a new distributor

1. Create a new package under `internal/distributor/<name>/`.
2. Implement the `distributor.Distributor` interface (Name, DisplayName, ValidationRules, Generate).
3. Call `distributor.Register(&YourDistributor{})` from the package's `init()`.
4. Import the package (blank import) in `cmd/validate.go` and `cmd/generate.go`.

No changes to core CLI code are required.

---

## Environment variables

| Variable             | Description                                      |
|----------------------|--------------------------------------------------|
| `STORYFORGE_CONTENT` | Path to the content directory. Overrides `--content`. |

---

## Working with AI assistants

Each saga contains a `CONTEXT.md` file designed to be pasted into an AI
assistant before asking it to write or revise story content. Each book has a
`notes.md` for free-form anecdotes and story seeds.

Typical AI workflow:

1. Open `CONTEXT.md` + the relevant `book.yaml` + `notes.md`.
2. Paste all three into your AI assistant.
3. Ask it to write a new spread, revise text, or generate a new story outline.
4. Copy the result back into `book.yaml`.

---

## Development

```bash
go test ./...
go vet ./...
```

The project follows standard Go module conventions. All business logic lives
under `internal/`; command definitions live under `cmd/`.

---

## License

Copyright © 2026 Diego Castro Viadero. All rights reserved.