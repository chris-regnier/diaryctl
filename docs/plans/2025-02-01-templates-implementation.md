# Templates Feature Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add composable, storage-backed templates with entry attribution to diaryctl.

**Architecture:** Templates are first-class entities in the `Storage` interface, stored alongside entries in both markdown and SQLite backends. Entries track which templates were used via a `[]TemplateRef` field. A new `internal/template/` package provides composition logic. New CLI commands manage templates; existing `create` and `edit` commands gain `--template` flags.

**Tech Stack:** Go 1.24, Cobra, Bubble Tea, Viper, existing markdown + SQLite storage backends.

**Worktree:** `/Users/chris-regnier/code/diaryctl/.worktrees/feature-templates` (branch: `feature/templates`)

---

### Task 1: Add Template and TemplateRef types to storage package

**Files:**
- Modify: `internal/storage/storage.go`
- Modify: `internal/entry/entry.go`

**Step 1: Add Template and TemplateRef types to storage.go**

Add after the `ListDaysOptions` struct (after line 39):

```go
// Template represents a reusable content template.
type Template struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TemplateRef is a lightweight reference to a template, stored on entries for attribution.
type TemplateRef struct {
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
}
```

**Step 2: Add TemplateRef to Entry struct**

In `internal/entry/entry.go`, add `Templates` field to the Entry struct:

```go
type Entry struct {
	ID        string        `json:"id"`
	Content   string        `json:"content"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Templates []storage.TemplateRef `json:"templates,omitempty"`
}
```

Note: This creates a dependency from `entry` → `storage`. If this causes a circular import (since `storage` imports `entry`), move `TemplateRef` to `internal/entry/` instead:

```go
// In internal/entry/entry.go
type TemplateRef struct {
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
}

type Entry struct {
	ID        string        `json:"id"`
	Content   string        `json:"content"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Templates []TemplateRef `json:"templates,omitempty"`
}
```

And reference `entry.TemplateRef` from storage types. This is the safer approach — use it.

**Step 3: Add template methods to Storage interface**

Add to the `Storage` interface in `storage.go`:

```go
type Storage interface {
	// Entry methods (existing)
	Create(e entry.Entry) error
	Get(id string) (entry.Entry, error)
	List(opts ListOptions) ([]entry.Entry, error)
	ListDays(opts ListDaysOptions) ([]DaySummary, error)
	Update(id string, content string) (entry.Entry, error)
	Delete(id string) error
	Close() error

	// Template methods
	CreateTemplate(t Template) error
	GetTemplate(id string) (Template, error)
	GetTemplateByName(name string) (Template, error)
	ListTemplates() ([]Template, error)
	UpdateTemplate(id string, name string, content string) (Template, error)
	DeleteTemplate(id string) error
}
```

**Step 4: Add template name validation to entry package**

Add to `internal/entry/entry.go`:

```go
var templateNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func ValidateTemplateName(name string) error {
	if !templateNamePattern.MatchString(name) {
		return fmt.Errorf("invalid template name %q: must be lowercase alphanumeric, hyphens, underscores", name)
	}
	return nil
}
```

**Step 5: Verify the project compiles (it won't yet — backends don't implement new methods)**

Run: `go vet ./internal/storage/ ./internal/entry/`
Expected: Should compile these packages in isolation. Full build will fail until backends implement the interface.

**Step 6: Commit**

```bash
git add internal/storage/storage.go internal/entry/entry.go
git commit -m "feat: add Template, TemplateRef types and template methods to Storage interface"
```

---

### Task 2: Add template contract tests

**Files:**
- Modify: `internal/storage/contract_test.go`

**Step 1: Write failing contract tests for template CRUD**

Add a new function `runTemplateContractTests` to `contract_test.go`. Follow the same pattern as `runContractTests` — it will be called by both `TestMarkdownStorage` and `TestSQLiteStorage`.

```go
func makeTemplate(name, content string) storage.Template {
	return storage.Template{
		ID:        entry.NewID(),
		Name:      name,
		Content:   content,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func runTemplateContractTests(t *testing.T, prefix string, factory storageFactory) {
	t.Run(prefix+" Template CRUD", func(t *testing.T) {

		t.Run("CreateTemplate and GetTemplate", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			tmpl := makeTemplate("daily", "# Daily Entry\n\n")
			err := s.CreateTemplate(tmpl)
			if err != nil {
				t.Fatalf("CreateTemplate: %v", err)
			}
			got, err := s.GetTemplate(tmpl.ID)
			if err != nil {
				t.Fatalf("GetTemplate: %v", err)
			}
			if got.Name != "daily" || got.Content != "# Daily Entry\n\n" {
				t.Errorf("got name=%q content=%q", got.Name, got.Content)
			}
		})

		t.Run("GetTemplateByName", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			tmpl := makeTemplate("prompts", "## Prompts\n\n- What happened today?\n")
			_ = s.CreateTemplate(tmpl)
			got, err := s.GetTemplateByName("prompts")
			if err != nil {
				t.Fatalf("GetTemplateByName: %v", err)
			}
			if got.ID != tmpl.ID {
				t.Errorf("got ID=%q want %q", got.ID, tmpl.ID)
			}
		})

		t.Run("GetTemplateByName not found", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			_, err := s.GetTemplateByName("nonexistent")
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})

		t.Run("CreateTemplate duplicate name", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			tmpl1 := makeTemplate("daily", "content1")
			tmpl2 := makeTemplate("daily", "content2")
			_ = s.CreateTemplate(tmpl1)
			err := s.CreateTemplate(tmpl2)
			if !errors.Is(err, storage.ErrConflict) {
				t.Errorf("expected ErrConflict, got %v", err)
			}
		})

		t.Run("ListTemplates", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			_ = s.CreateTemplate(makeTemplate("alpha", "a"))
			_ = s.CreateTemplate(makeTemplate("beta", "b"))
			_ = s.CreateTemplate(makeTemplate("gamma", "c"))
			list, err := s.ListTemplates()
			if err != nil {
				t.Fatalf("ListTemplates: %v", err)
			}
			if len(list) != 3 {
				t.Errorf("got %d templates, want 3", len(list))
			}
		})

		t.Run("UpdateTemplate", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			tmpl := makeTemplate("daily", "old content")
			_ = s.CreateTemplate(tmpl)
			updated, err := s.UpdateTemplate(tmpl.ID, "daily-v2", "new content")
			if err != nil {
				t.Fatalf("UpdateTemplate: %v", err)
			}
			if updated.Name != "daily-v2" || updated.Content != "new content" {
				t.Errorf("got name=%q content=%q", updated.Name, updated.Content)
			}
			if !updated.UpdatedAt.After(tmpl.UpdatedAt) || updated.UpdatedAt.Equal(tmpl.UpdatedAt) {
				// UpdatedAt should advance (or at least not go backwards)
			}
		})

		t.Run("UpdateTemplate not found", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			_, err := s.UpdateTemplate("nonexist", "name", "content")
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})

		t.Run("DeleteTemplate", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			tmpl := makeTemplate("daily", "content")
			_ = s.CreateTemplate(tmpl)
			err := s.DeleteTemplate(tmpl.ID)
			if err != nil {
				t.Fatalf("DeleteTemplate: %v", err)
			}
			_, err = s.GetTemplate(tmpl.ID)
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound after delete, got %v", err)
			}
		})

		t.Run("DeleteTemplate not found", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			err := s.DeleteTemplate("nonexist")
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})
	})
}
```

**Step 2: Add attribution contract tests**

```go
func runAttributionContractTests(t *testing.T, prefix string, factory storageFactory) {
	t.Run(prefix+" Template Attribution", func(t *testing.T) {

		t.Run("Create entry with template refs", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			tmpl := makeTemplate("daily", "# Daily\n")
			_ = s.CreateTemplate(tmpl)

			e := makeEntry("hello world")
			e.Templates = []entry.TemplateRef{
				{TemplateID: tmpl.ID, TemplateName: tmpl.Name},
			}
			err := s.Create(e)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			got, err := s.Get(e.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if len(got.Templates) != 1 || got.Templates[0].TemplateName != "daily" {
				t.Errorf("expected 1 template ref 'daily', got %v", got.Templates)
			}
		})

		t.Run("Create entry without template refs", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			e := makeEntry("no templates")
			err := s.Create(e)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			got, err := s.Get(e.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if len(got.Templates) != 0 {
				t.Errorf("expected 0 template refs, got %v", got.Templates)
			}
		})

		t.Run("List entries filtered by template name", func(t *testing.T) {
			s := factory(t)
			defer s.Close()
			tmpl := makeTemplate("daily", "# Daily\n")
			_ = s.CreateTemplate(tmpl)

			e1 := makeEntry("with template")
			e1.Templates = []entry.TemplateRef{
				{TemplateID: tmpl.ID, TemplateName: tmpl.Name},
			}
			_ = s.Create(e1)

			e2 := makeEntry("without template")
			_ = s.Create(e2)

			// This tests the TemplateName filter on ListOptions
			opts := storage.ListOptions{TemplateName: "daily"}
			results, err := s.List(opts)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(results) != 1 || results[0].ID != e1.ID {
				t.Errorf("expected 1 entry with template 'daily', got %d", len(results))
			}
		})
	})
}
```

**Step 3: Wire tests into TestMarkdownStorage and TestSQLiteStorage**

Add calls to `runTemplateContractTests` and `runAttributionContractTests` in both test functions.

**Step 4: Run tests to verify they fail**

Run: `go test ./internal/storage/ -v -run "Template" -count=1`
Expected: Compilation failure — backends don't implement new interface methods yet.

**Step 5: Commit**

```bash
git add internal/storage/contract_test.go
git commit -m "test: add template CRUD and attribution contract tests"
```

---

### Task 3: Add TemplateName filter to ListOptions

**Files:**
- Modify: `internal/storage/storage.go`

**Step 1: Add TemplateName to ListOptions**

```go
type ListOptions struct {
	Date         *time.Time
	StartDate    *time.Time
	EndDate      *time.Time
	TemplateName string // filter entries by template attribution
	OrderBy      string
	Limit        int
	Offset       int
}
```

**Step 2: Commit**

```bash
git add internal/storage/storage.go
git commit -m "feat: add TemplateName filter to ListOptions"
```

---

### Task 4: Implement template methods in markdown backend

**Files:**
- Modify: `internal/storage/markdown/markdown.go`

**Step 1: Implement template storage in markdown backend**

Templates are stored as files in `{baseDir}/../templates/` (sibling to `entries/`). File format:

```
---
id: <ID>
name: <name>
created_at: <RFC3339>
updated_at: <RFC3339>
---

<CONTENT>
```

Implement all six template methods:
- `CreateTemplate`: validate name, check for duplicate name, write file as `{templatesDir}/{name}.md`
- `GetTemplate`: find template by ID (scan files)
- `GetTemplateByName`: read `{templatesDir}/{name}.md` directly
- `ListTemplates`: scan all `.md` files in templates dir
- `UpdateTemplate`: find by ID, update name/content/updatedAt, rename file if name changed
- `DeleteTemplate`: find by ID, remove file

Update `New()` to also create the templates directory.

**Step 2: Update Entry marshal/unmarshal to handle TemplateRef**

Update the front-matter format for entries to include templates:

```
---
id: <ID>
created_at: <RFC3339>
updated_at: <RFC3339>
templates:
  - template_id: <ID>
    template_name: <NAME>
---
```

Update `marshal()` and `unmarshal()` to handle the `Templates` field. When `Templates` is empty, omit it from front-matter for backward compatibility with existing entries.

**Step 3: Update List to filter by TemplateName**

In the `List` method, after existing date filters, add filtering by `opts.TemplateName`: only include entries that have a matching `TemplateRef.TemplateName`.

**Step 4: Run contract tests**

Run: `go test ./internal/storage/ -v -run "Markdown" -count=1`
Expected: All markdown tests pass (template + attribution + existing).

**Step 5: Commit**

```bash
git add internal/storage/markdown/markdown.go
git commit -m "feat: implement template storage in markdown backend"
```

---

### Task 5: Implement template methods in SQLite backend

**Files:**
- Modify: `internal/storage/sqlite/sqlite.go`

**Step 1: Add template and attribution tables to schema**

Add to `createSchema()`:

```sql
CREATE TABLE IF NOT EXISTS templates (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS entry_templates (
    entry_id    TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    template_id TEXT NOT NULL,
    template_name TEXT NOT NULL,
    PRIMARY KEY (entry_id, template_id)
);
```

**Step 2: Implement six template methods**

- `CreateTemplate`: INSERT into templates, check UNIQUE constraint → ErrConflict
- `GetTemplate`: SELECT by id
- `GetTemplateByName`: SELECT by name
- `ListTemplates`: SELECT all, ORDER BY name
- `UpdateTemplate`: transaction — check exists, UPDATE name/content/updated_at, return updated
- `DeleteTemplate`: DELETE by id, check RowsAffected

**Step 3: Update Create to persist TemplateRefs**

After inserting the entry, insert rows into `entry_templates` for each `entry.TemplateRef`.

**Step 4: Update Get to load TemplateRefs**

After fetching the entry, SELECT from `entry_templates WHERE entry_id = ?` and populate `entry.Templates`.

**Step 5: Update List to join on entry_templates when TemplateName filter is set**

When `opts.TemplateName` is non-empty, add:
```sql
JOIN entry_templates et ON et.entry_id = entries.id
WHERE et.template_name = ?
```

Also update List to load template refs for each entry (batch query or per-entry).

**Step 6: Run contract tests**

Run: `go test ./internal/storage/ -v -run "SQLite" -count=1`
Expected: All SQLite tests pass.

**Step 7: Run all tests**

Run: `go test ./... -count=1`
Expected: All tests pass.

**Step 8: Commit**

```bash
git add internal/storage/sqlite/sqlite.go
git commit -m "feat: implement template storage in SQLite backend"
```

---

### Task 6: Add template composition package

**Files:**
- Create: `internal/template/template.go`
- Create: `internal/template/template_test.go`

**Step 1: Write failing tests for Compose**

```go
package template

import (
	"testing"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// mockStorage implements just the template methods needed for testing
type mockStorage struct {
	templates map[string]storage.Template // keyed by name
}

func (m *mockStorage) GetTemplateByName(name string) (storage.Template, error) {
	t, ok := m.templates[name]
	if !ok {
		return storage.Template{}, storage.ErrNotFound
	}
	return t, nil
}
// ... stub other Storage methods to satisfy interface ...

func TestComposeSingle(t *testing.T) {
	ms := &mockStorage{templates: map[string]storage.Template{
		"daily": {ID: "abc12345", Name: "daily", Content: "# Daily Entry\n"},
	}}
	content, refs, err := Compose(ms, []string{"daily"})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if content != "# Daily Entry\n" {
		t.Errorf("got content=%q", content)
	}
	if len(refs) != 1 || refs[0].TemplateName != "daily" {
		t.Errorf("got refs=%v", refs)
	}
}

func TestComposeMultiple(t *testing.T) {
	ms := &mockStorage{templates: map[string]storage.Template{
		"daily":   {ID: "abc12345", Name: "daily", Content: "# Daily\n"},
		"prompts": {ID: "def67890", Name: "prompts", Content: "## Prompts\n- Q1?\n"},
	}}
	content, refs, err := Compose(ms, []string{"daily", "prompts"})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	expected := "# Daily\n\n## Prompts\n- Q1?\n"
	if content != expected {
		t.Errorf("got content=%q, want %q", content, expected)
	}
	if len(refs) != 2 {
		t.Errorf("got %d refs, want 2", len(refs))
	}
}

func TestComposeNotFound(t *testing.T) {
	ms := &mockStorage{templates: map[string]storage.Template{}}
	_, _, err := Compose(ms, []string{"missing"})
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestComposeEmpty(t *testing.T) {
	ms := &mockStorage{templates: map[string]storage.Template{}}
	content, refs, err := Compose(ms, []string{})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if content != "" || len(refs) != 0 {
		t.Errorf("expected empty result, got content=%q refs=%v", content, refs)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/template/ -v -count=1`
Expected: Compilation failure (package doesn't exist).

**Step 3: Implement template.go**

```go
package template

import (
	"fmt"
	"strings"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// TemplateLoader is the subset of storage.Storage needed for template composition.
type TemplateLoader interface {
	GetTemplateByName(name string) (storage.Template, error)
}

// ParseNames splits a comma-separated template names string into a slice.
func ParseNames(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			names = append(names, p)
		}
	}
	return names
}

// Compose loads the named templates and concatenates their content.
// Returns the combined content and a slice of TemplateRefs for attribution.
// If names is empty, returns ("", nil, nil).
// If any template is not found, returns an error immediately (fail fast).
func Compose(loader TemplateLoader, names []string) (string, []entry.TemplateRef, error) {
	if len(names) == 0 {
		return "", nil, nil
	}

	var parts []string
	var refs []entry.TemplateRef

	for _, name := range names {
		tmpl, err := loader.GetTemplateByName(name)
		if err != nil {
			return "", nil, fmt.Errorf("template %q: %w", name, err)
		}
		parts = append(parts, tmpl.Content)
		refs = append(refs, entry.TemplateRef{
			TemplateID:   tmpl.ID,
			TemplateName: tmpl.Name,
		})
	}

	return strings.Join(parts, "\n"), refs, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/template/ -v -count=1`
Expected: All pass.

**Step 5: Commit**

```bash
git add internal/template/
git commit -m "feat: add template composition package"
```

---

### Task 7: Add DefaultTemplate to config

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add DefaultTemplate field**

```go
type Config struct {
	Storage         string `mapstructure:"storage"`
	DataDir         string `mapstructure:"data_dir"`
	Editor          string `mapstructure:"editor"`
	DefaultTemplate string `mapstructure:"default_template"`
}
```

**Step 2: Add default (empty string) in Load()**

Add `viper.SetDefault("default_template", "")` alongside existing defaults.

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add default_template config option"
```

---

### Task 8: Add `template` command group (list, show, create, edit, delete)

**Files:**
- Create: `cmd/template.go`
- Create: `cmd/template_test.go`

**Step 1: Write tests for template commands**

Follow the pattern in existing `cmd/*_test.go` files. Use `setupTestEnv(t)` to get a store. Test:
- `template list` returns empty list
- `template create` + `template list` returns the template
- `template show` returns content
- `template edit` updates content
- `template delete` removes the template
- `template show` of nonexistent returns error

**Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -v -run "Template" -count=1`
Expected: Compilation failure.

**Step 3: Implement template.go**

Create the `template` parent command and five subcommands:

```go
// templateCmd is the parent: diaryctl template
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage templates",
}

// templateListCmd: diaryctl template list
// templateShowCmd: diaryctl template show <name>
// templateCreateCmd: diaryctl template create <name> (opens editor, or reads stdin with -)
// templateEditCmd: diaryctl template edit <name> (opens editor with existing content)
// templateDeleteCmd: diaryctl template delete <name> (with --force flag)
```

Register `templateCmd` in `root.go`'s `init()`.

The create subcommand follows the same editor pattern as `cmd/create.go`:
- If arg is `-`, read from stdin
- Otherwise open editor with empty buffer
- Validate non-empty content
- Generate ID, create Template, call `store.CreateTemplate()`

The edit subcommand follows `cmd/edit.go` pattern:
- Fetch template by name via `store.GetTemplateByName()`
- Open editor with existing content
- If changed, call `store.UpdateTemplate()`

The delete subcommand follows `cmd/delete.go` pattern with `--force` flag.

**Step 4: Run tests**

Run: `go test ./cmd/ -v -run "Template" -count=1`
Expected: All pass.

**Step 5: Run full test suite**

Run: `go test ./... -count=1`
Expected: All pass.

**Step 6: Commit**

```bash
git add cmd/template.go cmd/template_test.go cmd/root.go
git commit -m "feat: add template list/show/create/edit/delete commands"
```

---

### Task 9: Add --template flag to `create` command

**Files:**
- Modify: `cmd/create.go`
- Modify: `cmd/create_test.go`

**Step 1: Write failing tests**

Test cases:
- `create --template daily` with editor: editor buffer pre-filled with template content, entry has template attribution
- `create --template daily,prompts`: composed content in editor buffer
- `create --no-template` with default_template configured: editor buffer is empty
- `create --template x --no-template`: error
- `create --template nonexistent`: error with clear message
- `create` with `default_template` in config: editor buffer pre-filled
- `create` with inline content + `--template`: template NOT applied (args bypass templates)

**Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -v -run "TestCreate" -count=1`
Expected: Fail.

**Step 3: Implement --template and --no-template flags on createCmd**

Add flags:
```go
createCmd.Flags().String("template", "", "template(s) to use (comma-separated)")
createCmd.Flags().Bool("no-template", false, "skip default template")
```

In RunE, before the editor path:
1. Check `--template` and `--no-template` conflict → error
2. Resolve template names: explicit flag > config default > none
3. If `--no-template`, skip
4. If names resolved, call `template.Compose(store, names)` to get content and refs
5. Pass template content as initial editor buffer: `editor.Edit(editorCmd, templateContent)`
6. Set `e.Templates = refs` on the created entry

**Step 4: Run tests**

Run: `go test ./cmd/ -v -run "TestCreate" -count=1`
Expected: All pass.

**Step 5: Commit**

```bash
git add cmd/create.go cmd/create_test.go
git commit -m "feat: add --template and --no-template flags to create command"
```

---

### Task 10: Add --template flag to `edit` command

**Files:**
- Modify: `cmd/edit.go`
- Modify: `cmd/edit_test.go`

**Step 1: Write failing tests**

Test cases:
- `edit <id> --template prompts`: existing content + template content in editor, template refs appended (no duplicates)
- `edit <id> --template prompts` where entry already has `prompts` ref: no duplicate in refs

**Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -v -run "TestEdit" -count=1`
Expected: Fail.

**Step 3: Implement --template flag on editCmd**

Add flag:
```go
editCmd.Flags().String("template", "", "template(s) to append (comma-separated)")
```

In RunE, after fetching the entry:
1. If `--template` is set, compose template content and refs
2. Append template content to existing content (with newline separator)
3. Open editor with combined content
4. On save, merge new template refs with existing `e.Templates` (deduplicate by TemplateID)
5. Need to extend `store.Update()` to accept template refs — or add a separate method

Note: The current `Update(id, content)` signature doesn't support updating template refs. Options:
- Add an `UpdateEntry(id string, content string, templates []entry.TemplateRef) (entry.Entry, error)` method
- Or extend the existing Update signature

Simpler approach: add `templates` parameter to Update. This changes the interface, so both backends need updating. The contract tests from Task 2 should cover this.

Update the `Storage` interface:
```go
Update(id string, content string, templates []entry.TemplateRef) (entry.Entry, error)
```

Update both backends and existing callers (cmd/update.go, cmd/edit.go) to pass the templates parameter. For the existing `update` command (which doesn't deal with templates), pass `nil` to preserve current behavior.

**Step 4: Run tests**

Run: `go test ./... -count=1`
Expected: All pass.

**Step 5: Commit**

```bash
git add cmd/edit.go cmd/edit_test.go internal/storage/storage.go internal/storage/markdown/markdown.go internal/storage/sqlite/sqlite.go cmd/update.go
git commit -m "feat: add --template flag to edit command with attribution tracking"
```

---

### Task 11: Add --template filter to `list` and `daily` commands

**Files:**
- Modify: `cmd/list.go`
- Modify: `cmd/list_test.go`
- Modify: `cmd/daily.go`
- Modify: `cmd/daily_test.go`

**Step 1: Write failing tests**

Test cases for list:
- `list --template daily`: only returns entries with "daily" template attribution

Test cases for daily:
- `daily --template daily`: only shows days that have entries with "daily" template

**Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -v -run "TestList|TestDaily" -count=1`
Expected: Fail.

**Step 3: Implement --template flag on listCmd and dailyCmd**

Add `--template` string flag to both commands. When set, pass `TemplateName` in `ListOptions` / filter results accordingly.

For `daily`, this requires also adding `TemplateName` to `ListDaysOptions` and implementing the filter in both backends.

**Step 4: Run tests**

Run: `go test ./... -count=1`
Expected: All pass.

**Step 5: Commit**

```bash
git add cmd/list.go cmd/list_test.go cmd/daily.go cmd/daily_test.go internal/storage/storage.go internal/storage/markdown/markdown.go internal/storage/sqlite/sqlite.go
git commit -m "feat: add --template filter to list and daily commands"
```

---

### Task 12: Display template attribution in `show` command

**Files:**
- Modify: `cmd/show.go`
- Modify: `cmd/show_test.go`
- Modify: `internal/ui/output.go`

**Step 1: Write failing test**

Test that `show <id>` for an entry with template refs displays them in the output (e.g., `Templates: daily, prompts`).

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -v -run "TestShow" -count=1`

**Step 3: Update FormatEntryFull in ui/output.go**

Add a `Templates:` line when `entry.Templates` is non-empty. Update `show.go` if needed.

**Step 4: Run tests**

Run: `go test ./... -count=1`
Expected: All pass.

**Step 5: Commit**

```bash
git add cmd/show.go cmd/show_test.go internal/ui/output.go
git commit -m "feat: display template attribution in show command"
```

---

### Task 13: Handle default_template misconfiguration gracefully

**Files:**
- Modify: `cmd/create.go`

**Step 1: Write failing test**

Test that `create` with `default_template = "nonexistent"` in config warns to stderr but still opens the editor with an empty buffer (doesn't block entry creation).

**Step 2: Implement graceful fallback**

In the create command's template resolution:
- If the default template fails to compose, print warning to stderr: `Warning: default template "nonexistent" not found, skipping`
- Continue with empty editor buffer
- This only applies to the config default — explicit `--template` flag still fails fast

**Step 3: Run tests**

Run: `go test ./... -count=1`
Expected: All pass.

**Step 4: Commit**

```bash
git add cmd/create.go cmd/create_test.go
git commit -m "feat: graceful fallback for misconfigured default_template"
```

---

### Task 14: Run gofmt and full test suite

**Step 1: Run gofmt**

Run: `gofmt -w .`

**Step 2: Run go vet**

Run: `go vet ./...`
Expected: No issues.

**Step 3: Run full test suite**

Run: `go test ./... -v -count=1`
Expected: All pass.

**Step 4: Commit any formatting fixes**

```bash
git add -A
git commit -m "style: apply gofmt formatting"
```

(Only if there are changes.)
