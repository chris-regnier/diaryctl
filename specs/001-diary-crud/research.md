# Research: Basic Diary CRUD

**Feature**: 001-diary-crud
**Date**: 2026-01-31

## Technology Decisions

### Go CLI Framework

- **Decision**: Cobra (`github.com/spf13/cobra`)
- **Rationale**: De facto standard for Go CLI tools with subcommand
  support. Excellent `--help` generation, shell completion, and
  community support. Used by kubectl, hugo, gh, and most major Go
  CLIs. Pairs well with Viper for configuration.
- **Alternatives considered**:
  - `urfave/cli`: Simpler but less flexible for nested subcommands.
  - `kong`: Clean API but smaller community; less ecosystem tooling.

### ID Generation

- **Decision**: nanoid (`github.com/matoous/go-nanoid/v2`)
- **Rationale**: Generates short, URL-safe, human-typeable unique
  strings. Configurable alphabet and length. Globally unique with
  sufficient entropy (default 21 chars; can use shorter with custom
  alphabet for human-typeable IDs, e.g., 8 lowercase alphanumeric
  chars gives ~2.8 trillion combinations). Supports future
  multi-device sync without coordination.
- **Alternatives considered**:
  - `rs/xid`: 20-char, includes machine ID + timestamp, but not
    as human-typeable (hex-like).
  - `oklog/ulid`: Lexicographically sortable but 26 chars, too long
    for comfortable CLI typing.
  - Sequential numeric: Simple but not globally unique across
    devices.

### SQLite Backend

- **Decision**: Turso (`github.com/tursodatabase/go-libsql`)
- **Rationale**: User-specified. Turso provides libSQL, a fork of
  SQLite with enhancements including embedded replicas and edge
  sync capabilities. Aligns with future multi-device sync goals.
  For local-only use, works as a drop-in SQLite replacement with
  the same file format.
- **Alternatives considered**:
  - `modernc.org/sqlite`: Pure Go, good for cross-compilation but
    no sync story.
  - `mattn/go-sqlite3`: CGO-based, well-tested but complicates
    cross-compilation.

### Markdown File Backend

- **Decision**: Individual Markdown files with YAML front-matter.
  Use `adrg/frontmatter` for parsing.
- **Rationale**: One file per entry stored as
  `~/.diaryctl/entries/YYYY/MM/DD/<id>.md`. YAML front-matter holds
  metadata (id, created, modified). Human-readable and version-
  control friendly. Directory structure enables efficient date-based
  queries without scanning all files.
- **Alternatives considered**:
  - Single monolithic file: Simpler but doesn't scale, hard to
    merge across devices.
  - JSON files: Less human-readable for diary content.

### Terminal UI

- **Decision**: Charmbracelet Bubble Tea + Lip Gloss
  (`github.com/charmbracelet/bubbletea`,
  `github.com/charmbracelet/lipgloss`)
- **Rationale**: User-specified. Bubble Tea provides an Elm-
  architecture TUI framework ideal for the interactive pager and
  future daily-view-picker feature. Lip Gloss handles terminal
  styling. For the CRUD feature specifically, Bubble Tea is used
  for the delete confirmation prompt and auto-pager. This also
  establishes the TUI foundation for feature 003-daily-view-picker.
- **Alternatives considered**:
  - Raw `os/exec` pager: Simpler for paging but no path to
    interactive UI.
  - `tview`: More widget-based but heavier; Bubble Tea is more
    composable and idiomatic Go.

### Configuration Management

- **Decision**: Viper (`github.com/spf13/viper`)
- **Rationale**: Natural pairing with Cobra. Supports TOML/YAML
  config files, environment variables, and flag binding. Handles
  XDG config paths. Provides the config file management needed for
  editor override and storage backend selection.
- **Alternatives considered**:
  - `koanf`: Lighter but less Cobra integration.
  - Plain TOML with `BurntSushi/toml`: Minimal but requires manual
    env var and flag merging.

### Configuration File Format

- **Decision**: TOML at `~/.diaryctl/config.toml` (or
  `$XDG_CONFIG_HOME/diaryctl/config.toml`)
- **Rationale**: TOML is human-friendly, well-suited for
  configuration files, and has strong Go library support. TOML is
  the conventional choice for Go CLI tools.
- **Alternatives considered**:
  - YAML: More complex syntax, easy to make whitespace errors.
  - JSON: No comments, less human-friendly for config.

### Pager Integration

- **Decision**: Use Bubble Tea's built-in viewport/pager component
  (`github.com/charmbracelet/bubbles/viewport`) for TTY output.
  Fall back to direct stdout when not a TTY.
- **Rationale**: Since Bubble Tea is already a dependency, using its
  viewport component keeps the pager consistent with the rest of
  the TUI. Avoids shelling out to external `less` process.
- **Alternatives considered**:
  - External pager via `$PAGER`/`less`: Standard Unix approach but
    inconsistent with the Bubble Tea TUI experience.

### Front-matter Library

- **Decision**: `adrg/frontmatter`
- **Rationale**: Simple, well-maintained, supports YAML/TOML/JSON
  front-matter detection and extraction. Minimal API surface.
- **Alternatives considered**:
  - `yuin/goldmark-meta`: Tied to goldmark Markdown parser; heavier
    dependency for just front-matter.
  - Manual parsing: Error-prone, especially for edge cases.

## Unresolved Items

None. All technology decisions are resolved.
