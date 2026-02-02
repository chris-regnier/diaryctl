# Templates Feature Design

## Overview

Add composable, storage-backed templates to diaryctl. Templates serve two purposes:

1. **Default on create** — a configured template pre-fills the editor buffer when creating new entries.
2. **Append on edit** — templates can be appended to existing entries for prompt-driven journaling, time tracking, or custom contexts.

Templates are first-class entities in the storage system with attribution tracking on entries.

## Template Storage & Discovery

Templates are stored in the same backend as entries (markdown or SQLite) via the `Storage` interface. This enables sharing across devices (via Turso sync) and consistent data management.

Template names are unique identifiers: lowercase alphanumeric, hyphens, and underscores. No subdirectories or namespacing in the initial implementation.

A config key `default_template` in `config.toml` specifies which template(s) to use on `create` by default:

```toml
default_template = "daily"
# or for composition:
default_template = "daily,prompts"
```

## Composition

Templates are composable. When multiple templates are specified (`--template a,b,c`), their contents are concatenated in order, separated by a single newline. This follows the `cat` mental model — predictable, easy to reason about.

## CLI Interface

### `create` command changes

- `--template <names>` flag (comma-separated): pre-fills editor buffer with concatenated template content.
- `--no-template` flag: skips the configured default template.
- When content is provided via args or stdin, templates are not applied (editor flow only).
- On successful create, the entry records which templates were used (attribution).

### `edit` command changes

- `--template <names>` flag (comma-separated): appends concatenated template content to existing entry content, then opens the editor.
- Appended template refs are added to the entry's existing template list (no duplicates).

### New `template` command group

- `diaryctl template list` — lists all templates (name, ID, preview). Supports `--json`.
- `diaryctl template show <name>` — prints full template content to stdout.
- `diaryctl template create <name>` — opens editor to author a new template (or via stdin).
- `diaryctl template edit <name>` — opens editor with existing template content.
- `diaryctl template delete <name>` — deletes with confirmation (or `--force`).

### Filtering by template

- `list --template <name>` — show only entries attributed to a given template.
- `daily --template <name>` — same filter on the daily view.
- `show <id>` — displays template attribution in metadata (e.g., `Templates: daily, prompts`).

## Internal Architecture

### Storage interface changes (`internal/storage/storage.go`)

New types:

```go
type Template struct {
    ID        string
    Name      string
    Content   string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type TemplateRef struct {
    TemplateID   string
    TemplateName string
}
```

New methods on `Storage` interface:

```go
CreateTemplate(t Template) error
GetTemplate(id string) (Template, error)
GetTemplateByName(name string) (Template, error)
ListTemplates() ([]Template, error)
UpdateTemplate(id string, name string, content string) (Template, error)
DeleteTemplate(id string) error
```

### Entry changes (`internal/entry/entry.go`)

Add `Templates []TemplateRef` field to `Entry` — records which templates were used to create or compose the entry.

### Storage implementations

Both backends implement the new template methods:

- **Markdown**: templates stored as files in a `templates/` subdirectory with YAML front-matter (same pattern as entries).
- **SQLite**: new `templates` table, plus a join table `entry_templates` for attribution.

`Create(e entry.Entry)` persists the `Templates` attribution alongside the entry.

### New `internal/template/` package

- `Compose(store storage.Storage, names []string) (string, []TemplateRef, error)` — loads and concatenates templates, returns combined content and refs for attribution.
- `ValidateName(name string) error` — name validation.

### Config changes (`internal/config/config.go`)

- Add `DefaultTemplate string` field (maps to `default_template` config key, comma-separated names).

## Error Handling & Edge Cases

### Template resolution

- Template name not found: clear error `template "foo" not found`.
- In a composed list, if any template is missing: fail fast, don't partially apply.
- `--template` and `--no-template` used together: error.

### Attribution integrity

- If a template is deleted, existing entry attributions are preserved (the ref stores both ID and name, so history survives deletion).
- `list --template <name>` matches against stored template name in the attribution, not the live template. Renaming a template doesn't retroactively change filtering of old entries.

### Default template misconfiguration

- If `default_template` names a template that doesn't exist: warn but don't block entry creation. Users shouldn't be locked out of creating entries because of a config issue.

### Edit flow

- Duplicate template refs: if an entry was created with `daily` and you `edit --template daily`, the ref list doesn't duplicate.
- Appending to an empty entry works fine.

### Template CRUD

- Template names must be unique. `CreateTemplate` with a duplicate name returns `ErrConflict`.
- Name validation: lowercase alphanumeric, hyphens, underscores. No spaces, no path separators.
