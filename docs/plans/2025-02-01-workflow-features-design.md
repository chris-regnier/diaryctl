# Workflow Features Design

## Overview

Five features that address two core problems: **capture friction** (the editor is the bottleneck) and **retrieval** (can't find things later). Together they make diaryctl useful as a daily professional tool and composable as a Unix-philosophy substrate for scripts and other tools.

The daily entry becomes a first-class concept: one entry per day, accumulated via jots, structured via templates, searchable, and pipe-able.

## Priority Order

1. Jot — frictionless capture
2. Today — daily entry shortcut
3. Context providers — smart editor defaults
4. Guided capture — template-driven prompt flows
5. Search — full-text retrieval

## Prerequisites

- Templates feature (in progress on `feature/templates` branch)
- Template front-matter parsing (needed for guided capture)

---

## Feature 1: Jot — Frictionless Capture

### Concept

`diaryctl jot "text"` appends a timestamped line to today's daily entry, creating it if one doesn't exist. No editor opens. This is the lowest-friction capture path.

### Usage

```bash
diaryctl jot "standup: working on auth refactor"
diaryctl jot "decision: going with JWT over sessions"
diaryctl jot "blocker: waiting on DB creds from ops"

# Pipe support
echo "some note" | diaryctl jot -
```

### Resulting Entry

```markdown
# 2025-02-01

- **09:15** standup: working on auth refactor
- **10:42** decision: going with JWT over sessions
- **14:03** blocker: waiting on DB creds from ops
```

### Behavior

- Content comes from args (joined with spaces) or stdin (with `-`).
- If no daily entry exists for today, one is created. If `default_template` is configured, the template content is used as the initial body before appending the jot.
- Each jot is appended as a timestamped bullet: `- **HH:MM** <content>`.
- Timestamps are local time.
- The entry's `UpdatedAt` advances with each jot.

### Storage Interface

New method on `Storage`:

```go
// GetByDate returns the entry for a given calendar date (local timezone).
// Returns ErrNotFound if no entry exists for that date.
GetByDate(date time.Time) (entry.Entry, error)
```

Or alternatively, `jot` uses `List(ListOptions{Date: today, Limit: 1})` to find today's entry, then `Update` to append. This avoids a new interface method.

Decision: use existing `List` + `Update` to avoid interface churn. The `jot` command handles the "create if not exists" logic internally.

---

## Feature 2: Today — Daily Entry Shortcut

### Concept

`diaryctl today` is a shortcut to view, edit, or reference today's daily entry. It makes the one-entry-per-day convention a first-class CLI concept.

### Usage

```bash
diaryctl today                # show today's entry (create if missing)
diaryctl today --edit         # open today's entry in editor
diaryctl today --id-only      # print just the entry ID (for scripting)
diaryctl today --content-only # print just the content (for piping)
diaryctl today --json         # JSON output
```

### Behavior

- If no entry exists for today, creates one (using default template if configured).
- `--edit` opens the editor with today's entry content.
- `--id-only` and `--content-only` are pipe-friendly output modes.
- Combines naturally with jot: `diaryctl jot "note"` then `diaryctl today` to see the full day.

---

## Feature 3: Context Providers — Smart Editor Defaults

### Concept

When the editor opens (via `create`, `edit`, or `today --edit`), the buffer can be pre-populated with contextual information beyond templates. Context providers are small functions that generate text snippets.

### Built-in Providers

- `datetime` — current date/time header (e.g., `# Saturday, February 1, 2025`)
- `git` — current branch, recent commits, dirty files (when inside a git repo)
- `lastentry` — preview of the most recent entry for continuity

### Configuration

```toml
default_template = "daily"
context_providers = ["datetime", "git"]
```

### Assembly Order

Context providers first, then templates, then cursor position:

```markdown
# Saturday, February 1, 2025
branch: feature/auth | 3 uncommitted files

## What am I working on?

## Blockers?

## Notes

```

### Architecture

New `internal/context/` package:

```go
type Provider interface {
    Name() string
    Generate() (string, error)
}
```

Built-in providers registered by name. Config lists which to activate. `Compose` function concatenates provider output + template content.

Providers are opt-in. Zero configured = just the template (or blank buffer). Providers that fail (e.g., `git` when not in a repo) are silently skipped.

---

## Feature 4: Guided Capture — Template-Driven Prompt Flows

### Concept

Templates can declare prompts in their front-matter. When invoked with `--guided`, diaryctl walks through the prompts in a TUI and assembles the entry from answers.

### Template Format

```markdown
---
name: standup
prompts:
  - key: yesterday
    question: "What did you accomplish yesterday?"
  - key: today
    question: "What are you working on today?"
  - key: blockers
    question: "Any blockers?"
---

## Yesterday
{{yesterday}}

## Today
{{today}}

## Blockers
{{blockers}}
```

### Usage

```bash
diaryctl create --template standup --guided
diaryctl create --template standup --guided --edit  # review in editor before saving

# Scripted: answers from JSON stdin
echo '{"yesterday":"auth work","today":"API endpoints","blockers":"none"}' \
  | diaryctl create --template standup --guided -
```

### Behavior

- `--guided` triggers the prompt flow. Without it, the template is used as static content.
- TUI presents each question one at a time (Bubble Tea text input), collects answers, substitutes `{{key}}` placeholders in the template body.
- If a template has no `prompts` front-matter, `--guided` is a no-op (falls back to editor).
- `--edit` flag opens the editor with the assembled content for review before saving.
- JSON stdin mode enables scripting and automation.

### Architecture

This introduces simple variable substitution into templates — the first step toward a template engine. The substitution is intentionally minimal: `{{key}}` replaced with the answer string. No expressions, no conditionals, no loops. Future work can add Sprig or similar.

Template front-matter parsing needs to handle the `prompts` field alongside existing fields (`name`, etc.). The `storage.Template` type may need a `Prompts []Prompt` field, or prompts can be parsed on-demand from the content.

---

## Feature 5: Search

### Concept

`diaryctl search <query>` provides full-text search across all entries with date and template filters.

### Usage

```bash
diaryctl search "API design"
diaryctl search "blocker" --from 2025-01-01
diaryctl search "blocker" --template standup
diaryctl search "decision" --id-only | xargs -I{} diaryctl show {}
```

### Output

Matches the `list` format: ID, date, and a preview with matching text. Supports `--json` and `--id-only` for piping.

### Implementation

- **Markdown backend:** Scan all entries, substring match. Simple, works for moderate volumes.
- **SQLite backend:** FTS5 virtual table for efficient full-text search with ranking.

### Storage Interface

```go
type SearchOptions struct {
    Query        string
    StartDate    *time.Time
    EndDate      *time.Time
    TemplateName string
    Limit        int
    Offset       int
}

// New method on Storage interface
Search(opts SearchOptions) ([]entry.Entry, error)
```

---

## Pipe-ability Conventions

Woven throughout all commands, not a separate feature:

- `--id-only` — print just entry IDs, one per line. For scripting with `xargs`.
- `--content-only` — print just entry content, no metadata. For piping to other tools.
- `--json` — existing flag, already supported.

```bash
# Chain commands
diaryctl today --id-only | xargs diaryctl show
diaryctl search "decision" --id-only | xargs -I{} diaryctl show {}
diaryctl today --content-only | grep "blocker:"
```

These flags should be added to: `show`, `list`, `daily`, `today`, `search`.

---

## Entry Linking

Convention-based, not enforced:

```markdown
- **14:03** decision: going with JWT, see [[abc12345]] for context
```

`[[id]]` syntax is a convention for referencing other entries. Initial implementation:
- `show` renders `[[id]]` references (display the referenced entry's date and preview inline).
- No validation that the referenced ID exists.
- Future: `backlinks` command to find entries referencing a given ID.

---

## What's Excluded (YAGNI)

- AI/LLM integration (summarization, auto-tagging)
- Sync between backends
- Mobile/web interface
- Plugin system
- Complex query language for search
- Regex search (initially)
