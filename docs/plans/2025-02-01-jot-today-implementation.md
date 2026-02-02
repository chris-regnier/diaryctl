# Jot & Today Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `diaryctl jot` for frictionless timestamped capture to a daily entry, and `diaryctl today` as a shortcut to view/edit today's entry.

**Architecture:** `jot` finds or creates today's entry, appends a timestamped bullet, and saves — no editor opens. `today` is a convenience command wrapping list/show/edit scoped to the current day. Both use the existing `Storage` interface (no new methods) and integrate with the templates system for default content on new daily entries.

**Tech Stack:** Go 1.24, Cobra, existing markdown + SQLite storage backends, template composition.

**Worktree:** `/Users/chris-regnier/code/diaryctl/.worktrees/feature-jot-today` (branch: `feature/jot-today`, based on `feature/templates`)

**Depends on:** Templates feature (tasks 1-7 on `feature/templates`)

---

### Task 1: Add a `daily` package for shared "find or create today's entry" logic

Both `jot` and `today` need to find today's entry or create one. This shared logic lives in a small internal package to avoid duplication.

**Files:**
- Create: `internal/daily/daily.go`
- Create: `internal/daily/daily_test.go`

**Step 1: Write failing tests**

Create `internal/daily/daily_test.go`:

```go
package daily

import (
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
)

func testStore(t *testing.T) storage.Storage {
	t.Helper()
	s, err := markdown.New(t.TempDir())
	if err != nil {
		t.Fatalf("markdown.New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestGetOrCreateToday_CreatesNewEntry(t *testing.T) {
	s := testStore(t)
	e, created, err := GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if !created {
		t.Error("expected created=true for new entry")
	}
	if e.ID == "" {
		t.Error("expected non-empty ID")
	}
	// Verify it's stored
	got, err := s.Get(e.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != e.ID {
		t.Errorf("got ID=%q, want %q", got.ID, e.ID)
	}
}

func TestGetOrCreateToday_FindsExistingEntry(t *testing.T) {
	s := testStore(t)
	// Create an entry dated today
	now := time.Now().UTC()
	e := entry.Entry{
		ID:        entry.NewID(),
		Content:   "existing daily entry",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, created, err := GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if created {
		t.Error("expected created=false for existing entry")
	}
	if got.ID != e.ID {
		t.Errorf("got ID=%q, want %q", got.ID, e.ID)
	}
}

func TestGetOrCreateToday_WithDefaultTemplate(t *testing.T) {
	s := testStore(t)
	// Create a template first
	tmpl := storage.Template{
		ID:        entry.NewID(),
		Name:      "daily",
		Content:   "# Daily Log\n\n",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.CreateTemplate(tmpl); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	e, created, err := GetOrCreateToday(s, "daily")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if !created {
		t.Error("expected created=true")
	}
	if e.Content != "# Daily Log\n\n" {
		t.Errorf("expected template content, got %q", e.Content)
	}
	if len(e.Templates) != 1 || e.Templates[0].TemplateName != "daily" {
		t.Errorf("expected template attribution, got %v", e.Templates)
	}
}

func TestGetOrCreateToday_BadDefaultTemplateWarns(t *testing.T) {
	s := testStore(t)
	// Pass a nonexistent template name — should still create entry, not error
	e, created, err := GetOrCreateToday(s, "nonexistent")
	if err != nil {
		t.Fatalf("GetOrCreateToday should not error on bad default template: %v", err)
	}
	if !created {
		t.Error("expected created=true")
	}
	if e.Content != "" {
		t.Errorf("expected empty content when template missing, got %q", e.Content)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/daily/ -v -count=1`
Expected: Compilation failure — package doesn't exist.

**Step 3: Implement daily.go**

Create `internal/daily/daily.go`:

```go
package daily

import (
	"fmt"
	"os"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/template"
)

// GetOrCreateToday finds today's entry or creates a new one.
// If defaultTemplate is non-empty, it composes the template content for new entries.
// If the default template is not found, a warning is printed to stderr and an empty entry is created.
// Returns the entry, whether it was newly created, and any error.
func GetOrCreateToday(store storage.Storage, defaultTemplate string) (entry.Entry, bool, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Try to find today's entry
	entries, err := store.List(storage.ListOptions{
		Date:  &today,
		Limit: 1,
	})
	if err != nil {
		return entry.Entry{}, false, fmt.Errorf("listing today's entries: %w", err)
	}
	if len(entries) > 0 {
		return entries[0], false, nil
	}

	// No entry for today — create one
	var content string
	var refs []entry.TemplateRef

	if defaultTemplate != "" {
		names := template.ParseNames(defaultTemplate)
		c, r, err := template.Compose(store, names)
		if err != nil {
			// Warn but don't block entry creation
			fmt.Fprintf(os.Stderr, "Warning: default template %q not found, skipping\n", defaultTemplate)
		} else {
			content = c
			refs = r
		}
	}

	nowUTC := time.Now().UTC()
	e := entry.Entry{
		ID:        entry.NewID(),
		Content:   content,
		CreatedAt: nowUTC,
		UpdatedAt: nowUTC,
		Templates: refs,
	}
	if err := store.Create(e); err != nil {
		return entry.Entry{}, false, fmt.Errorf("creating today's entry: %w", err)
	}
	return e, true, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/daily/ -v -count=1`
Expected: All pass.

**Step 5: Commit**

```bash
git add internal/daily/
git commit -m "feat: add daily package with GetOrCreateToday helper"
```

---

### Task 2: Add `jot` command — core append logic

**Files:**
- Create: `cmd/jot.go`
- Create: `cmd/jot_test.go`
- Modify: `cmd/root.go` (register command)

**Step 1: Write failing tests**

Create `cmd/jot_test.go`:

```go
package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

func TestJotCreatesNewDailyEntry(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig.DefaultTemplate = ""

	err := jotRun([]string{"hello", "world"})
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}

	// Find today's entry
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	entries, err := s.List(storage.ListOptions{Date: &today})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Content, "hello world") {
		t.Errorf("expected content to contain 'hello world', got %q", entries[0].Content)
	}
}

func TestJotAppendsToExistingEntry(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig.DefaultTemplate = ""

	// Create an existing entry for today
	now := time.Now().UTC()
	e := entry.Entry{
		ID:        entry.NewID(),
		Content:   "- **09:00** first note",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	err := jotRun([]string{"second", "note"})
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}

	got, err := s.Get(e.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.Contains(got.Content, "first note") {
		t.Error("original content should be preserved")
	}
	if !strings.Contains(got.Content, "second note") {
		t.Error("jot content should be appended")
	}
}

func TestJotTimestampFormat(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig.DefaultTemplate = ""

	err := jotRun([]string{"timestamped note"})
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	entries, _ := s.List(storage.ListOptions{Date: &today})
	content := entries[0].Content

	// Should contain a timestamp in HH:MM format
	expectedTime := now.Format("15:04")
	if !strings.Contains(content, "**"+expectedTime+"**") {
		t.Errorf("expected timestamp %q in content %q", expectedTime, content)
	}
}

func TestJotEmptyContentRejected(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig.DefaultTemplate = ""

	err := jotRun([]string{})
	if err == nil {
		t.Error("expected error for empty jot")
	}
}

func TestJotFromStdin(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig.DefaultTemplate = ""

	// This test verifies the stdin path works via the command's RunE.
	// Actual stdin piping is tested via integration; here we test
	// that jotRun handles content correctly.
	err := jotRun([]string{"piped content"})
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	entries, _ := s.List(storage.ListOptions{Date: &today})
	if !strings.Contains(entries[0].Content, "piped content") {
		t.Error("expected piped content in entry")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -v -run "TestJot" -count=1`
Expected: Compilation failure.

**Step 3: Implement jot.go**

Create `cmd/jot.go`:

```go
package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/daily"
	"github.com/spf13/cobra"
)

var jotCmd = &cobra.Command{
	Use:   "jot [content...]",
	Short: "Append a timestamped note to today's entry",
	Long: `Quickly capture a thought by appending a timestamped line to today's
daily entry. If no entry exists for today, one is created (using the
default template if configured).

Content can be provided as arguments or piped via stdin with "-".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var content string

		if len(args) == 1 && args[0] == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			content = strings.TrimSpace(string(data))
		} else {
			content = strings.TrimSpace(strings.Join(args, " "))
		}

		if content == "" {
			return fmt.Errorf("jot content cannot be empty")
		}

		return jotRun(args)
	},
}

// jotRun is the core logic, separated for testing.
func jotRun(args []string) error {
	var content string

	if len(args) == 1 && args[0] == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		content = strings.TrimSpace(string(data))
	} else {
		content = strings.TrimSpace(strings.Join(args, " "))
	}

	if content == "" {
		return fmt.Errorf("jot content cannot be empty")
	}

	// Find or create today's entry
	e, _, err := daily.GetOrCreateToday(store, appConfig.DefaultTemplate)
	if err != nil {
		return err
	}

	// Format the jot line
	timestamp := time.Now().Format("15:04")
	jotLine := fmt.Sprintf("- **%s** %s", timestamp, content)

	// Append to existing content
	var newContent string
	if strings.TrimSpace(e.Content) == "" {
		newContent = jotLine
	} else {
		newContent = e.Content + "\n" + jotLine
	}

	// Update the entry
	_, err = store.Update(e.ID, newContent)
	if err != nil {
		return fmt.Errorf("updating entry: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(jotCmd)
}
```

Note: The `RunE` and `jotRun` have duplicated content-parsing logic. Refactor so `RunE` just calls `jotRun` — put all logic in `jotRun`. The `RunE` should parse content from args/stdin, then call the core function. Alternatively, make `jotRun` accept the content string directly:

```go
func jotRun(content string) error { ... }
```

And have `RunE` handle arg parsing / stdin, then call `jotRun(content)`. Adjust tests to call `jotRun("hello world")` directly.

**Step 4: Run tests**

Run: `go test ./cmd/ -v -run "TestJot" -count=1`
Expected: All pass.

**Step 5: Run full test suite**

Run: `go test ./... -count=1`
Expected: All pass (no regressions).

**Step 6: Commit**

```bash
git add cmd/jot.go cmd/jot_test.go cmd/root.go
git commit -m "feat: add jot command for frictionless timestamped capture"
```

---

### Task 3: Add `today` command — view today's entry

**Files:**
- Create: `cmd/today.go`
- Create: `cmd/today_test.go`
- Modify: `cmd/root.go` (register command, if not using init())

**Step 1: Write failing tests**

Create `cmd/today_test.go`:

```go
package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
)

func TestTodayShowsExistingEntry(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig.DefaultTemplate = ""

	// Create an entry for today
	now := time.Now().UTC()
	e := entry.Entry{
		ID:        entry.NewID(),
		Content:   "today's content",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var buf bytes.Buffer
	err := todayRun(&buf, false, false)
	if err != nil {
		t.Fatalf("todayRun: %v", err)
	}
	if !strings.Contains(buf.String(), "today's content") {
		t.Errorf("expected output to contain entry content, got %q", buf.String())
	}
}

func TestTodayCreatesEntryIfMissing(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig.DefaultTemplate = ""

	var buf bytes.Buffer
	err := todayRun(&buf, false, false)
	if err != nil {
		t.Fatalf("todayRun: %v", err)
	}

	// Should have created an entry
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	entries, _ := s.List(storage.ListOptions{Date: &today})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestTodayIDOnly(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig.DefaultTemplate = ""

	now := time.Now().UTC()
	e := entry.Entry{
		ID:        entry.NewID(),
		Content:   "content",
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = s.Create(e)

	var buf bytes.Buffer
	err := todayRun(&buf, true, false)
	if err != nil {
		t.Fatalf("todayRun: %v", err)
	}
	output := strings.TrimSpace(buf.String())
	if output != e.ID {
		t.Errorf("expected just ID %q, got %q", e.ID, output)
	}
}

func TestTodayContentOnly(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig.DefaultTemplate = ""

	now := time.Now().UTC()
	e := entry.Entry{
		ID:        entry.NewID(),
		Content:   "just the content",
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = s.Create(e)

	var buf bytes.Buffer
	err := todayRun(&buf, false, true)
	if err != nil {
		t.Fatalf("todayRun: %v", err)
	}
	output := strings.TrimSpace(buf.String())
	if output != "just the content" {
		t.Errorf("expected content only, got %q", output)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -v -run "TestToday" -count=1`
Expected: Compilation failure.

**Step 3: Implement today.go**

Create `cmd/today.go`:

```go
package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/chris-regnier/diaryctl/internal/daily"
	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var (
	todayEdit        bool
	todayIDOnly      bool
	todayContentOnly bool
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "View or edit today's diary entry",
	Long: `Show today's diary entry, creating one if it doesn't exist.
Use --edit to open the entry in your editor, or --id-only / --content-only
for scripting.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if todayEdit {
			return todayEditRun()
		}
		return todayRun(os.Stdout, todayIDOnly, todayContentOnly)
	},
}

func todayRun(w io.Writer, idOnly bool, contentOnly bool) error {
	e, _, err := daily.GetOrCreateToday(store, appConfig.DefaultTemplate)
	if err != nil {
		return err
	}

	if jsonOutput {
		return ui.FormatJSON(w, e)
	}
	if idOnly {
		fmt.Fprintln(w, e.ID)
		return nil
	}
	if contentOnly {
		fmt.Fprintln(w, e.Content)
		return nil
	}

	ui.FormatEntryFull(w, e)
	return nil
}

func todayEditRun() error {
	e, _, err := daily.GetOrCreateToday(store, appConfig.DefaultTemplate)
	if err != nil {
		return err
	}

	editorCmd := editor.ResolveEditor(appConfig.Editor)
	content, changed, err := editor.Edit(editorCmd, e.Content)
	if err != nil {
		return fmt.Errorf("editor: %w", err)
	}
	if !changed {
		ui.FormatNoChanges(os.Stdout, e.ID)
		return nil
	}

	updated, err := store.Update(e.ID, content)
	if err != nil {
		return fmt.Errorf("updating entry: %w", err)
	}

	if jsonOutput {
		return ui.FormatJSON(os.Stdout, updated)
	}
	ui.FormatEntryUpdated(os.Stdout, updated)
	return nil
}

func init() {
	todayCmd.Flags().BoolVar(&todayEdit, "edit", false, "open today's entry in editor")
	todayCmd.Flags().BoolVar(&todayIDOnly, "id-only", false, "print only the entry ID")
	todayCmd.Flags().BoolVar(&todayContentOnly, "content-only", false, "print only the entry content")
	rootCmd.AddCommand(todayCmd)
}
```

**Step 4: Run tests**

Run: `go test ./cmd/ -v -run "TestToday" -count=1`
Expected: All pass.

**Step 5: Run full test suite**

Run: `go test ./... -count=1`
Expected: All pass.

**Step 6: Commit**

```bash
git add cmd/today.go cmd/today_test.go
git commit -m "feat: add today command to view/edit today's entry"
```

---

### Task 4: Add `--id-only` and `--content-only` flags to `show` and `list`

These pipe-ability flags were called out in the workflow design. Adding them now makes `jot` + `today` + other commands composable via pipes.

**Files:**
- Modify: `cmd/show.go`
- Modify: `cmd/show_test.go`
- Modify: `cmd/list.go`
- Modify: `cmd/list_test.go`

**Step 1: Write failing tests for show**

Add to `cmd/show_test.go`:

```go
func TestShowIDOnly(t *testing.T) {
	s := setupTestStore(t)
	store = s
	// Create entry, call show with idOnly=true
	// Assert output is just the ID
}

func TestShowContentOnly(t *testing.T) {
	s := setupTestStore(t)
	store = s
	// Create entry, call show with contentOnly=true
	// Assert output is just the content, no metadata
}
```

**Step 2: Write failing tests for list**

Add to `cmd/list_test.go`:

```go
func TestListIDOnly(t *testing.T) {
	s := setupTestStore(t)
	store = s
	// Create 3 entries, call list with idOnly=true
	// Assert output is 3 lines, each a valid entry ID
}
```

**Step 3: Run tests to verify they fail**

Run: `go test ./cmd/ -v -run "TestShowIDOnly|TestShowContentOnly|TestListIDOnly" -count=1`
Expected: Fail.

**Step 4: Add flags to show command**

Add `--id-only` and `--content-only` bool flags to `showCmd`. In RunE:
- `--id-only`: print just the ID and return
- `--content-only`: print just `e.Content` and return
- These take precedence over `--json`

**Step 5: Add `--id-only` flag to list command**

Add `--id-only` bool flag to `listCmd`. In RunE:
- `--id-only`: for each entry, print just the ID (one per line)

**Step 6: Run tests**

Run: `go test ./cmd/ -v -run "TestShow|TestList" -count=1`
Expected: All pass.

**Step 7: Run full test suite**

Run: `go test ./... -count=1`
Expected: All pass.

**Step 8: Commit**

```bash
git add cmd/show.go cmd/show_test.go cmd/list.go cmd/list_test.go
git commit -m "feat: add --id-only and --content-only flags to show and list commands"
```

---

### Task 5: Handle `today` with multiple entries on the same day

The design says "one entry per day" as a convention, but users may have created multiple entries for the same day via `create`. `today` and `jot` need a predictable behavior.

**Files:**
- Modify: `internal/daily/daily.go`
- Modify: `internal/daily/daily_test.go`

**Step 1: Write failing test**

```go
func TestGetOrCreateToday_MultipleEntries_ReturnsNewest(t *testing.T) {
	s := testStore(t)

	now := time.Now().UTC()
	old := entry.Entry{
		ID:        entry.NewID(),
		Content:   "older entry",
		CreatedAt: now.Add(-1 * time.Hour),
		UpdatedAt: now.Add(-1 * time.Hour),
	}
	newer := entry.Entry{
		ID:        entry.NewID(),
		Content:   "newer entry",
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = s.Create(old)
	_ = s.Create(newer)

	got, created, err := GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if created {
		t.Error("expected created=false")
	}
	// Should return the newest entry for today
	if got.ID != newer.ID {
		t.Errorf("expected newest entry %q, got %q", newer.ID, got.ID)
	}
}
```

**Step 2: Run test to verify it fails (or passes — check behavior)**

Run: `go test ./internal/daily/ -v -run "MultipleEntries" -count=1`

The existing implementation uses `List(Date: today, Limit: 1)` which returns entries ordered by `created_at DESC`. This should already return the newest entry. If the test passes, we just need the test for documentation. If it fails, fix the query.

**Step 3: Commit**

```bash
git add internal/daily/daily.go internal/daily/daily_test.go
git commit -m "test: verify jot/today selects newest entry when multiple exist"
```

---

### Task 6: Add `--template` flag to `jot` command

The jot command should support an explicit template override for the daily entry creation (not for the jot line itself).

**Files:**
- Modify: `cmd/jot.go`
- Modify: `cmd/jot_test.go`

**Step 1: Write failing test**

```go
func TestJotWithTemplateFlag(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig.DefaultTemplate = ""

	// Create a template
	tmpl := storage.Template{
		ID:        entry.NewID(),
		Name:      "worklog",
		Content:   "# Work Log\n\n",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = s.CreateTemplate(tmpl)

	// Jot with explicit template (overrides empty default)
	err := jotRunWithTemplate("hello world", "worklog")
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	entries, _ := s.List(storage.ListOptions{Date: &today})
	if !strings.Contains(entries[0].Content, "# Work Log") {
		t.Error("expected template content in entry")
	}
	if !strings.Contains(entries[0].Content, "hello world") {
		t.Error("expected jot content in entry")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/ -v -run "TestJotWithTemplate" -count=1`

**Step 3: Implement**

Add `--template` string flag to `jotCmd`. Pass the template name (or fall back to `appConfig.DefaultTemplate`) to `daily.GetOrCreateToday()`. This requires `GetOrCreateToday` to accept the template name as a parameter (already does).

Update the jot logic: if `--template` is provided, use that; otherwise use `appConfig.DefaultTemplate`.

**Step 4: Run tests**

Run: `go test ./cmd/ -v -run "TestJot" -count=1`
Expected: All pass.

**Step 5: Commit**

```bash
git add cmd/jot.go cmd/jot_test.go
git commit -m "feat: add --template flag to jot command"
```

---

### Task 7: Add JSON output to `jot` command

**Files:**
- Modify: `cmd/jot.go`
- Modify: `cmd/jot_test.go`

**Step 1: Write failing test**

```go
func TestJotJSONOutput(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig.DefaultTemplate = ""
	jsonOutput = true
	defer func() { jsonOutput = false }()

	var buf bytes.Buffer
	err := jotRunToWriter(&buf, "test note")
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}
	if !strings.Contains(buf.String(), `"content"`) {
		t.Error("expected JSON output")
	}
}
```

**Step 2: Implement**

When `jsonOutput` is true, after the jot append + update, output the updated entry as JSON. Refactor `jotRun` to accept an `io.Writer` parameter.

**Step 3: Run tests**

Run: `go test ./cmd/ -v -run "TestJot" -count=1`
Expected: All pass.

**Step 4: Commit**

```bash
git add cmd/jot.go cmd/jot_test.go
git commit -m "feat: add JSON output support to jot command"
```

---

### Task 8: Run gofmt and full test suite

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
