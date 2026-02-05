# Fix Code Review Findings Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Address critical and important issues found in TUI code review to ensure production readiness

**Architecture:** Fix navigation bugs, improve error handling, and add test coverage for interactive actions in the Bubble Tea TUI implementation

**Tech Stack:** Go 1.24, Bubble Tea, existing test infrastructure

---

## Task 1: Fix Context Panel ESC Navigation Bug (CRITICAL)

**Files:**
- Modify: `internal/ui/picker.go:859`
- Test: `internal/ui/picker_test.go`

**Step 1: Write failing test for context panel navigation**

Add to `internal/ui/picker_test.go`:

```go
func TestContextPanelEscReturnsToCorrectScreen(t *testing.T) {
	tests := []struct {
		name       string
		fromScreen screenType
	}{
		{"from today screen", screenToday},
		{"from date list", screenDateList},
		{"from day detail", screenDayDetail},
		{"from entry detail", screenEntryDetail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStorage()
			m := newModel(store)
			m.screen = screenContext
			m.prevScreen = tt.fromScreen

			// Simulate ESC key
			msg := tea.KeyMsg{Type: tea.KeyEsc}
			updatedModel, _ := m.Update(msg)
			m = updatedModel.(model)

			if m.screen != tt.fromScreen {
				t.Errorf("ESC from context panel should return to %v, got %v", tt.fromScreen, m.screen)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui -run TestContextPanelEscReturnsToCorrectScreen -v`
Expected: FAIL - test demonstrates the bug where all cases return to screenToday

**Step 3: Implement screen-aware navigation fix**

In `internal/ui/picker.go`, replace lines 858-860:

```go
case "esc":
	m.screen = m.prevScreen
	switch m.prevScreen {
	case screenToday:
		return m, m.loadTodayCmd
	case screenDateList:
		return m, nil // already loaded
	case screenDayDetail:
		return m, nil // already loaded
	case screenEntryDetail:
		return m, nil // already loaded
	default:
		return m, nil
	}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui -run TestContextPanelEscReturnsToCorrectScreen -v`
Expected: PASS - all screen transitions work correctly

**Step 5: Manual verification**

Run: `go run . daily` (in TTY mode)
Test sequence:
1. From today screen: press 'x' → ESC → should return to today
2. Navigate to date list → select day → press 'x' → ESC → should return to day detail
3. From date list → press 'x' → ESC → should return to date list

**Step 6: Commit**

```bash
git add internal/ui/picker.go internal/ui/picker_test.go
git commit -m "fix(tui): correct context panel ESC navigation to respect previous screen

Previously ESC from context panel always returned to today screen
regardless of which screen the user came from. Now properly returns
to the previous screen (today, date list, day detail, or entry detail).

Fixes critical navigation bug identified in code review."
```

---

## Task 2: Add Error Handling for Context Auto-Attach

**Files:**
- Modify: `internal/ui/picker.go:923`
- Test: `internal/ui/picker_test.go`

**Step 1: Write failing test for attach error handling**

Add to `internal/ui/picker_test.go`:

```go
func TestContextAutoAttachErrorHandling(t *testing.T) {
	store := newMockStorage()
	// Make AttachContext return an error
	store.attachError = fmt.Errorf("database connection failed")

	m := newModel(store)
	m.screen = screenContext
	m.contextEntryID = "test123"
	m.contextInput.SetValue("work")

	// Simulate creating a new context
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(model)

	// Execute the command to trigger auto-attach
	if cmd != nil {
		result := cmd()
		if msg, ok := result.(contextCreatedMsg); ok {
			// The error should be captured, not ignored
			if msg.err == nil {
				t.Error("Expected error from failed auto-attach, got nil")
			}
		}
	}
}
```

**Step 2: Update mockStorage to support error injection**

In `internal/ui/picker_test.go`, add to mockStorage:

```go
type mockStorage struct {
	// ... existing fields ...
	attachError error  // Add this field
}

func (m *mockStorage) AttachContext(entryID, contextName string) error {
	if m.attachError != nil {
		return m.attachError
	}
	// ... existing implementation ...
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/ui -run TestContextAutoAttachErrorHandling -v`
Expected: FAIL - error is currently ignored

**Step 4: Implement error handling in createContext**

In `internal/ui/picker.go`, modify the `createContext` function around line 906-924:

```go
func (m model) createContext(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.store.CreateContext(name)
		if err != nil {
			return contextCreatedMsg{err: err}
		}

		// Auto-attach if we have an entry selected
		var attachErr error
		if m.contextEntryID != "" {
			attachErr = m.store.AttachContext(m.contextEntryID, name)
			if attachErr != nil {
				// Context was created but attach failed
				return contextCreatedMsg{err: fmt.Errorf("context created but failed to attach: %w", attachErr)}
			}
		}

		return contextCreatedMsg{}
	}
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/ui -run TestContextAutoAttachErrorHandling -v`
Expected: PASS - error is properly captured and returned

**Step 6: Commit**

```bash
git add internal/ui/picker.go internal/ui/picker_test.go
git commit -m "fix(tui): handle errors in context auto-attach operation

Previously, errors from AttachContext were silently ignored during
auto-attach after creating a new context. Now errors are properly
captured and reported to the user.

Addresses important issue from code review."
```

---

## Task 3: Improve File Descriptor Cleanup in Editor Actions

**Files:**
- Modify: `internal/ui/picker.go:1000-1006,1048-1055`

**Step 1: Write test for temp file cleanup**

Add to `internal/ui/picker_test.go`:

```go
func TestEditorTempFileCleanup(t *testing.T) {
	// This test verifies that temp file Close() errors are handled
	// We can't easily test the actual cleanup, but we can verify the pattern
	store := newMockStorage()
	m := newModel(store)

	// Verify startCreate and startEdit don't panic with basic operations
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Editor action panicked: %v", r)
		}
	}()

	// Note: Full integration test would require mocking os.CreateTemp
	// which is beyond scope. This verifies the functions are callable.
	_ = m.startCreate()

	m.entries = []storage.Entry{{ID: "test", Content: "test content"}}
	m.selectedIndex = 0
	_ = m.startEdit()
}
```

**Step 2: Run test to verify baseline**

Run: `go test ./internal/ui -run TestEditorTempFileCleanup -v`
Expected: PASS (establishes baseline, will catch panics if introduced)

**Step 3: Fix startCreate temp file cleanup**

In `internal/ui/picker.go`, replace lines 1000-1006:

```go
tmpFile, err := os.CreateTemp("", "diaryctl-*.md")
if err != nil {
	m.err = err
	return m, tea.Quit
}
tmpName := tmpFile.Name()
if err := tmpFile.Close(); err != nil {
	os.Remove(tmpName)
	m.err = fmt.Errorf("failed to prepare temp file: %w", err)
	return m, tea.Quit
}
```

**Step 4: Fix startEdit temp file cleanup**

In `internal/ui/picker.go`, replace lines 1048-1055:

```go
tmpFile, err := os.CreateTemp("", "diaryctl-*.md")
if err != nil {
	m.err = err
	return m, tea.Quit
}
tmpName := tmpFile.Name()

// Write current content
if _, err := tmpFile.WriteString(e.Content); err != nil {
	tmpFile.Close()
	os.Remove(tmpName)
	m.err = fmt.Errorf("failed to write to temp file: %w", err)
	return m, tea.Quit
}

if err := tmpFile.Close(); err != nil {
	os.Remove(tmpName)
	m.err = fmt.Errorf("failed to prepare temp file: %w", err)
	return m, tea.Quit
}
```

**Step 5: Run test to verify no regressions**

Run: `go test ./internal/ui -run TestEditorTempFileCleanup -v`
Expected: PASS - no panics or errors

**Step 6: Run all tests**

Run: `go test ./internal/ui -v`
Expected: All tests PASS

**Step 7: Commit**

```bash
git add internal/ui/picker.go internal/ui/picker_test.go
git commit -m "fix(tui): improve temp file cleanup in editor actions

Add explicit error checking for tmpFile.Close() operations in both
startCreate and startEdit. Ensures temp files are properly cleaned
up even when Close() fails, preventing potential file descriptor leaks.

Addresses important issue from code review."
```

---

## Task 4: Add Named Constants for Magic Numbers

**Files:**
- Modify: `internal/ui/picker.go:105,885`

**Step 1: Define focus constants**

In `internal/ui/picker.go`, add near the screen constants (after line 29):

```go
// Focus states for today screen
const (
	focusDailyViewport = 0
	focusEntryList     = 1
)

// Input validation limits
const (
	maxContextNameLength = 100
	maxJotInputLength    = 200
)
```

**Step 2: Replace todayFocus magic numbers**

In `internal/ui/picker.go`, search and replace:
- Line 569: `m.todayFocus == 0` → `m.todayFocus == focusDailyViewport`
- Line 571: `m.todayFocus == 1` → `m.todayFocus == focusEntryList`
- Line 358: `m.todayFocus = 0` → `m.todayFocus = focusDailyViewport`
- Line 360: `m.todayFocus = 1` → `m.todayFocus = focusEntryList`

**Step 3: Replace context name length limit**

In `internal/ui/picker.go`, line 885:
- `m.contextInput.CharLimit = 100` → `m.contextInput.CharLimit = maxContextNameLength`

**Step 4: Replace jot input length limit**

In `internal/ui/picker.go`, line 756:
- `m.jotInput.CharLimit = 200` → `m.jotInput.CharLimit = maxJotInputLength`

**Step 5: Run all tests**

Run: `go test ./internal/ui -v`
Expected: All tests PASS - behavior unchanged, just clearer code

**Step 6: Commit**

```bash
git add internal/ui/picker.go
git commit -m "refactor(tui): replace magic numbers with named constants

Add named constants for:
- Today screen focus states (viewport vs entry list)
- Input validation limits (context name, jot input)

Improves code readability and maintainability.

Addresses code review suggestion."
```

---

## Task 5: Add Tests for Interactive Actions

**Files:**
- Test: `internal/ui/picker_test.go`

**Step 1: Write test for jot action with new entry**

Add to `internal/ui/picker_test.go`:

```go
func TestJotActionCreatesNewEntry(t *testing.T) {
	store := newMockStorage()
	m := newModel(store)
	m.screen = screenToday

	// Activate jot mode
	m.jotActive = true
	m.jotInput.SetValue("Meeting notes at 2pm")

	// Simulate Enter key to submit
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(model)

	// Execute the jot command
	if cmd != nil {
		result := cmd()
		if jotMsg, ok := result.(jotFinishedMsg); ok {
			if jotMsg.err != nil {
				t.Fatalf("Jot failed: %v", jotMsg.err)
			}
		}
	}

	// Verify entry was created
	if len(store.entries) != 1 {
		t.Errorf("Expected 1 entry after jot, got %d", len(store.entries))
	}

	if !strings.Contains(store.entries[0].Content, "Meeting notes at 2pm") {
		t.Errorf("Entry content doesn't contain jot text: %s", store.entries[0].Content)
	}
}
```

**Step 2: Write test for jot action appending to existing**

```go
func TestJotActionAppendsToExistingEntry(t *testing.T) {
	store := newMockStorage()
	// Pre-populate with today's daily entry
	today := time.Now().Truncate(24 * time.Hour)
	existingEntry := storage.Entry{
		ID:        "daily01",
		Content:   "# Daily Entry\n\nInitial content",
		CreatedAt: today,
		UpdatedAt: today,
	}
	store.entries = append(store.entries, existingEntry)

	m := newModel(store)
	m.screen = screenToday
	m.jotActive = true
	m.jotInput.SetValue("Additional note")

	// Submit jot
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(model)

	if cmd != nil {
		result := cmd()
		if jotMsg, ok := result.(jotFinishedMsg); ok {
			if jotMsg.err != nil {
				t.Fatalf("Jot failed: %v", jotMsg.err)
			}
		}
	}

	// Should still have 1 entry (updated, not created)
	if len(store.entries) != 1 {
		t.Errorf("Expected 1 entry after jot append, got %d", len(store.entries))
	}

	// Should contain both original and new content
	content := store.entries[0].Content
	if !strings.Contains(content, "Initial content") {
		t.Error("Original content lost during jot append")
	}
	if !strings.Contains(content, "Additional note") {
		t.Error("Jot content not appended")
	}
}
```

**Step 3: Write test for delete action confirmation**

```go
func TestDeleteActionConfirmation(t *testing.T) {
	store := newMockStorage()
	store.entries = []storage.Entry{
		{ID: "entry01", Content: "Test entry", CreatedAt: time.Now()},
	}

	m := newModel(store)
	m.screen = screenEntryDetail
	m.entries = store.entries
	m.selectedIndex = 0

	// Activate delete mode
	m.deleteActive = true

	// Simulate 'y' to confirm
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(model)

	if cmd != nil {
		result := cmd()
		if delMsg, ok := result.(deleteFinishedMsg); ok {
			if delMsg.err != nil {
				t.Fatalf("Delete failed: %v", delMsg.err)
			}
		}
	}

	// Entry should be deleted
	if len(store.entries) != 0 {
		t.Errorf("Expected 0 entries after delete, got %d", len(store.entries))
	}
}
```

**Step 4: Write test for delete action cancellation**

```go
func TestDeleteActionCancellation(t *testing.T) {
	store := newMockStorage()
	store.entries = []storage.Entry{
		{ID: "entry01", Content: "Test entry", CreatedAt: time.Now()},
	}

	m := newModel(store)
	m.screen = screenEntryDetail
	m.entries = store.entries
	m.selectedIndex = 0
	m.deleteActive = true

	// Simulate 'n' to cancel
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(model)

	// Entry should NOT be deleted
	if len(store.entries) != 1 {
		t.Errorf("Expected 1 entry after cancel, got %d", len(store.entries))
	}

	// Delete mode should be deactivated
	if m.deleteActive {
		t.Error("Delete mode should be deactivated after cancellation")
	}
}
```

**Step 5: Write test for help overlay toggle**

```go
func TestHelpOverlayToggle(t *testing.T) {
	store := newMockStorage()
	m := newModel(store)
	m.screen = screenToday

	// Initially help should be inactive
	if m.helpActive {
		t.Error("Help should be inactive initially")
	}

	// Press '?' to show help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(model)

	if !m.helpActive {
		t.Error("Help should be active after pressing '?'")
	}

	// Press '?' again to hide help
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(model)

	if m.helpActive {
		t.Error("Help should be inactive after toggling off")
	}
}
```

**Step 6: Run new tests**

Run: `go test ./internal/ui -run "TestJot|TestDelete|TestHelp" -v`
Expected: All new tests PASS

**Step 7: Run all tests**

Run: `go test ./internal/ui -v`
Expected: All tests PASS

**Step 8: Commit**

```bash
git add internal/ui/picker_test.go
git commit -m "test(tui): add comprehensive tests for interactive actions

Add test coverage for:
- Jot action (create new entry and append to existing)
- Delete action (confirmation and cancellation flows)
- Help overlay toggle

Improves test coverage for user-facing interactive features.

Addresses code review recommendation."
```

---

## Task 6: Add Tests for Context Panel Actions

**Files:**
- Test: `internal/ui/picker_test.go`

**Step 1: Write test for context attach action**

Add to `internal/ui/picker_test.go`:

```go
func TestContextPanelAttach(t *testing.T) {
	store := newMockStorage()
	store.contexts = []string{"work", "personal"}
	store.entries = []storage.Entry{
		{ID: "entry01", Content: "Test", CreatedAt: time.Now()},
	}

	m := newModel(store)
	m.screen = screenContext
	m.contextEntryID = "entry01"
	m.contextList.Select(0) // Select "work"

	// Simulate Enter to attach
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(model)

	if cmd != nil {
		result := cmd()
		if attachMsg, ok := result.(contextAttachedMsg); ok {
			if attachMsg.err != nil {
				t.Fatalf("Attach failed: %v", attachMsg.err)
			}
		}
	}

	// Verify context was attached
	attached := store.entryContexts["entry01"]
	found := false
	for _, ctx := range attached {
		if ctx == "work" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Context 'work' was not attached to entry")
	}
}
```

**Step 2: Write test for context detach action**

```go
func TestContextPanelDetach(t *testing.T) {
	store := newMockStorage()
	store.contexts = []string{"work", "personal"}
	store.entries = []storage.Entry{
		{ID: "entry01", Content: "Test", CreatedAt: time.Now()},
	}
	// Pre-attach "work" context
	store.entryContexts = map[string][]string{
		"entry01": {"work"},
	}

	m := newModel(store)
	m.screen = screenContext
	m.contextEntryID = "entry01"
	m.contextList.Select(0) // Select "work" (already attached)

	// Simulate Enter to detach
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(model)

	if cmd != nil {
		result := cmd()
		if detachMsg, ok := result.(contextDetachedMsg); ok {
			if detachMsg.err != nil {
				t.Fatalf("Detach failed: %v", detachMsg.err)
			}
		}
	}

	// Verify context was detached
	attached := store.entryContexts["entry01"]
	for _, ctx := range attached {
		if ctx == "work" {
			t.Error("Context 'work' should have been detached")
		}
	}
}
```

**Step 3: Write test for context creation**

```go
func TestContextPanelCreate(t *testing.T) {
	store := newMockStorage()

	m := newModel(store)
	m.screen = screenContext
	m.contextInput.SetValue("newcontext")

	// Simulate Ctrl+N to initiate creation
	msg := tea.KeyMsg{Type: tea.KeyCtrlN}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(model)

	// Should be in context creation mode
	// (implementation detail: check if input is focused)

	// Simulate Enter to confirm creation
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(model)

	if cmd != nil {
		result := cmd()
		if createMsg, ok := result.(contextCreatedMsg); ok {
			if createMsg.err != nil {
				t.Fatalf("Create failed: %v", createMsg.err)
			}
		}
	}

	// Verify context was created
	found := false
	for _, ctx := range store.contexts {
		if ctx == "newcontext" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Context 'newcontext' was not created")
	}
}
```

**Step 4: Update mockStorage to track attach/detach**

In `internal/ui/picker_test.go`, enhance mockStorage:

```go
type mockStorage struct {
	// ... existing fields ...
	entryContexts map[string][]string  // Add this field
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		entries:       []storage.Entry{},
		contexts:      []string{},
		entryContexts: make(map[string][]string),
	}
}

func (m *mockStorage) AttachContext(entryID, contextName string) error {
	if m.attachError != nil {
		return m.attachError
	}
	if m.entryContexts[entryID] == nil {
		m.entryContexts[entryID] = []string{}
	}
	m.entryContexts[entryID] = append(m.entryContexts[entryID], contextName)
	return nil
}

func (m *mockStorage) DetachContext(entryID, contextName string) error {
	contexts := m.entryContexts[entryID]
	filtered := []string{}
	for _, ctx := range contexts {
		if ctx != contextName {
			filtered = append(filtered, ctx)
		}
	}
	m.entryContexts[entryID] = filtered
	return nil
}
```

**Step 5: Run new tests**

Run: `go test ./internal/ui -run TestContextPanel -v`
Expected: All new tests PASS

**Step 6: Run all tests**

Run: `go test ./internal/ui -v`
Expected: All tests PASS

**Step 7: Commit**

```bash
git add internal/ui/picker_test.go
git commit -m "test(tui): add comprehensive tests for context panel

Add test coverage for:
- Context attach action
- Context detach action
- Context creation with auto-attach

Enhances mockStorage to track context attachments for testing.

Addresses code review recommendation for test coverage."
```

---

## Task 7: Run Full Test Suite and Format

**Files:**
- All Go files

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS across all packages

**Step 2: Run gofmt**

Run: `gofmt -w .`
Expected: All files formatted consistently

**Step 3: Run go vet**

Run: `go vet ./...`
Expected: No issues reported

**Step 4: Manual TUI testing**

Run: `go run . daily`

Test all fixed functionality:
1. **Context panel navigation**: Open from different screens, ESC returns correctly
2. **Jot action**: Create new entry and append to existing
3. **Delete action**: Confirm and cancel flows
4. **Context operations**: Attach, detach, create with auto-attach
5. **Help overlay**: Toggle on/off

**Step 5: Commit if formatting changes**

```bash
git add .
git commit -m "chore: apply gofmt and final cleanup

Run gofmt and go vet across all files to ensure code quality
after implementing code review fixes."
```

---

## Summary

This plan addresses:

**Critical Issues (MUST FIX):**
- ✅ Context panel ESC navigation bug (Task 1)

**Important Issues (SHOULD FIX):**
- ✅ Context auto-attach error handling (Task 2)
- ✅ File descriptor cleanup in editor actions (Task 3)

**Code Quality Improvements:**
- ✅ Named constants for magic numbers (Task 4)
- ✅ Test coverage for interactive actions (Task 5)
- ✅ Test coverage for context panel (Task 6)
- ✅ Full test suite verification (Task 7)

**Testing Strategy:**
- TDD approach: Write failing test first, implement fix, verify pass
- Manual verification for UI/UX flows
- Full regression testing after each task

**Commit Strategy:**
- One commit per task
- Descriptive commit messages following conventional commits
- Include "Addresses code review" in commit messages

**Estimated Time:**
- Task 1 (Critical bug): 15-20 minutes
- Task 2 (Error handling): 15-20 minutes
- Task 3 (File cleanup): 10-15 minutes
- Task 4 (Constants): 5-10 minutes
- Task 5 (Action tests): 20-25 minutes
- Task 6 (Context tests): 20-25 minutes
- Task 7 (Full suite): 10-15 minutes
- **Total: ~2 hours**
