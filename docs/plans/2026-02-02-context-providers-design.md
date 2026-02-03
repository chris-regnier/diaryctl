# Context Providers Design

## Overview

Contexts are first-class objects representing recurring themes, projects, or environments that diary entries relate to. They serve two purposes: **semantic grouping** (entries that share a context are related and queryable together) and **content generation** (some contexts contribute text to the editor buffer when creating entries).

Contexts are created and attached to entries through two mechanisms:

- **Context resolvers** automatically detect contexts from the environment (e.g., the current git branch) at command invocation time. They create context objects if they don't already exist and attach them to the entry being created.
- **Manual contexts** are explicitly set by the user (`diaryctl context set sprint:23`) and persist in a state file until unset. They're attached to every entry created while active.

Two separate interfaces keep concerns clean:

- **Content providers** generate text snippets for the editor buffer. Not every source needs to produce contexts.
- **Context resolvers** detect contexts from the environment. Not every source needs to produce editor content.

Some sources (like `git`) export both. Others (like `datetime`) only provide content.

Active contexts for any given entry = union of manually set contexts + auto-resolved contexts at time of creation.

## Data Model

### Context Object

```go
type Context struct {
    ID        string    // unique identifier (ULID, like entries)
    Name      string    // human-readable, unique (e.g., "feature/auth", "sprint:23")
    Source    string    // how it was created: "manual", "git", etc.
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Context Reference on Entry

Mirrors the existing `TemplateRef` pattern:

```go
type ContextRef struct {
    ID   string
    Name string
}
```

The `entry.Entry` struct gains a `Contexts []ContextRef` field alongside the existing `Templates []TemplateRef`.

### Manual Context State

A file at `~/.diaryctl/active-contexts.json`:

```json
["sprint:23", "project:auth-refactor"]
```

These are context names, not IDs. At entry creation time, the system resolves names to context objects, creating them with `source: "manual"` if they don't exist yet.

### Storage Additions

New methods on the `Storage` interface:

```go
// Context CRUD
CreateContext(c Context) error
GetContext(id string) (Context, error)
GetContextByName(name string) (Context, error)
ListContexts() ([]Context, error)
DeleteContext(id string) error

// Entry-context association
AttachContext(entryID string, contextID string) error
DetachContext(entryID string, contextID string) error

// Query
ListEntriesByContext(contextID string, opts ListOptions) ([]entry.Entry, error)
```

### Markdown Backend

Contexts stored in entry frontmatter:

```yaml
---
id: 01HQXYZ...
contexts: ["feature/auth", "sprint:23"]
---
```

Context objects themselves stored as `~/.diaryctl/contexts/<name>.json`.

### SQLite Backend

`contexts` table + `entry_contexts` join table, same pattern as templates.

## Interfaces

Two separate interfaces in a new `internal/context/` package:

```go
// ContentProvider generates text for the editor buffer.
type ContentProvider interface {
    Name() string
    Generate() (string, error)
}

// ContextResolver detects contexts from the environment.
type ContextResolver interface {
    Name() string
    Resolve() ([]string, error) // returns context names
}
```

`Resolve()` returns context **names**, not full objects. The caller looks up or creates context objects in storage. This keeps resolvers free of storage dependencies — they just inspect the environment and report what they find.

### Composition Function

Assembles the editor buffer:

```go
func ComposeContent(providers []ContentProvider, templateContent string) string
```

Order: provider output (in config order, separated by newlines) → blank line → template content.

### Context Attachment Function

Gathers and attaches all active contexts:

```go
func ResolveActiveContexts(
    resolvers []ContextResolver,
    manualContexts []string,
    store storage.Storage,
) ([]entry.ContextRef, error)
```

This function:

1. Runs each resolver, collects context names
2. Merges with manual context names (deduplicates)
3. For each name, calls `GetContextByName` — if not found, creates it with the appropriate source
4. Returns the list of `ContextRef` values to attach to the entry

### Registration

A simple registry mapping config names to constructors:

```go
var contentProviders = map[string]func() ContentProvider{
    "datetime": NewDatetimeProvider,
    "git":      NewGitContentProvider,
}

var contextResolvers = map[string]func() ContextResolver{
    "git": NewGitContextResolver,
}
```

No plugin system, no dynamic loading. Just a map. New providers are added by editing the map.

## Built-in Providers

### `datetime` — ContentProvider Only

Generates a formatted date header. No context resolution — dates aren't a useful grouping mechanism (that's what `list --date` is for).

```
# Saturday, February 1, 2025
```

Format is hardcoded to `# Weekday, Month Day, Year`. No configuration needed — if someone wants a different format, they put it in a template instead.

If called when content already exists (e.g., appending a jot), it is not invoked.

### `git` — ContentProvider + ContextResolver

Two separate types in an `internal/context/git/` package, both inspecting the same environment.

**Content provider output:**

```
branch: feature/auth | 3 uncommitted files
latest: a1b2c3d refactor auth middleware (2h ago)
```

Compact, two lines max. Branch + dirty count on line one, most recent commit on line two. If not in a git repo, `Generate()` returns `("", nil)` — silent skip, not an error.

**Context resolver behavior:**

- Returns the current branch name as a context name (e.g., `feature/auth`)
- Detached HEAD → returns no context
- `main`/`master` → still returned. Filtering out default branches is user preference, not provider opinion.

Created context objects get `source: "git"`.

Both use `os/exec` to call `git` commands. No git library dependency.

## CLI Commands

### Context Management

```bash
diaryctl context list                  # list all known contexts
diaryctl context show <name>           # show context details + entry count
diaryctl context delete <name>         # delete context (detaches from entries)

diaryctl context set <name>            # add to active manual contexts
diaryctl context unset <name>          # remove from active manual contexts
diaryctl context active                # show currently active contexts (manual + auto-resolved)
```

`context set/unset` modify `~/.diaryctl/active-contexts.json`. `context active` runs the resolvers and merges with manual contexts to show what would be attached to an entry created right now.

### Modified Existing Commands

`create`, `jot`, `today` — at entry creation time, resolve active contexts and attach them. No new flags needed; context attachment is automatic based on config + state.

`list`, `show` — gain `--context <name>` filter flag. `show` output includes context labels. `list` output shows context labels per entry.

`daily` — TUI picker shows context labels on entries.

### Configuration

```toml
context_providers = ["datetime", "git"]   # content providers, in order
context_resolvers = ["git"]               # context resolvers to run
```

Both default to empty (opt-in). If neither is configured, context behavior is entirely manual.

### Output Examples

```bash
$ diaryctl context active
manual:  sprint:23, project:auth-refactor
auto:    feature/auth (git)

$ diaryctl list --context feature/auth
2025-02-01  01HQXYZ  [feature/auth, sprint:23]  standup notes...
2025-01-31  01HQXYW  [feature/auth]              auth middleware refactor...

$ diaryctl context list
NAME                SOURCE   ENTRIES
feature/auth        git      12
sprint:23           manual   8
main                git      3
project:auth        manual   5
```

## Integration Points

### Entry Creation Flow

Applies to `create`, `jot`, `today`:

1. Load config → get `context_providers`, `context_resolvers` lists
2. Run content providers → collect text snippets
3. Load template (if configured) → get template content
4. `ComposeContent(snippets, templateContent)` → editor buffer
5. Open editor / append jot / create entry
6. `ResolveActiveContexts(resolvers, manualContexts, store)` → `[]ContextRef`
7. Attach contexts to entry via storage

Steps 2–4 only apply when creating new content (not when appending a jot to an existing entry). Steps 6–7 always apply — even a jot to an existing entry attaches any new contexts that weren't there before. For example, you switch branches mid-day, then jot — the new branch context gets added to today's entry alongside the old one.

### Context Accumulation

Contexts on an entry are **append-only** during the entry's lifetime. A jot never removes existing contexts, only adds new ones. This preserves the history of what contexts were active across the day.

### Failure Behavior

- Content provider fails → skip silently, log at debug level
- Context resolver fails → skip silently, log at debug level
- Manual context name can't be resolved to a storage object → create it
- Storage error attaching context → surface as warning, don't fail the entry creation

The principle: context is supplementary. A failure in the context system should never prevent capturing a diary entry.

## Out of Scope

- Browse-by-context view in the `daily` TUI
- Context renaming or merging
- Context descriptions or metadata beyond name/source
- `lastentry` content provider
- `project` context resolver
- Entry linking (`[[id]]` syntax)
