# TUI Features Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enhance the existing daily picker into a full-featured TUI with today-first home screen, write actions (jot, create, edit, delete), context management, and help overlay — becoming the default `diaryctl` behavior.

**Architecture:** Progressive enhancement of `internal/ui/picker.go`. The `pickerModel` gains new screen states, the `StorageProvider` interface expands for write ops, and the root command launches the TUI when in a TTY. Editor actions suspend the TUI via `tea.ExecProcess`. All state changes trigger data refresh commands.

**Tech Stack:** Go 1.24, Bubble Tea + Bubbles (list, viewport, textinput) + Lipgloss, Cobra

**Design doc:** `docs/plans/2026-02-04-tui-features-design.md`

---

### Task 1: Expand StorageProvider Interface

**Files:**
- Modify: `internal/ui/picker.go` (lines 25-31, the `StorageProvider` interface)

**Step 1: Expand the interface**

Replace the existing `StorageProvider` interface with:

```go
// StorageProvider abstracts storage operations for the TUI.
type StorageProvider interface {
	// Read
	ListDays(opts storage.ListDaysOptions) ([]storage.DaySummary, error)
	List(opts storage.ListOptions) ([]entry.Entry, error)
	Get(id string) (entry.Entry, error)

	// Write
	Create(e entry.Entry) error
	Update(id string, content string, templates []entry.TemplateRef) (entry.Entry, error)
	Delete(id string) error

	// Context
	ListContexts() ([]storage.Context, error)
	CreateContext(c storage.Context) error
	AttachContext(entryID string, contextID string) error
	DetachContext(entryID string, contextID string) error
}
```

**Step 2: Verify it compiles**

Run: `go build ./...`
Expected: PASS (storage.Storage already satisfies this — it's a subset)

**Step 3: Commit**

```bash
git add internal/ui/picker.go
git commit -m "feat(tui): expand StorageProvider interface for write and context ops"
```

---

### Task 2: Add TUI Config Struct and RunTUI Entry Point

The TUI needs config values (editor command, default template) that the picker currently doesn't have access to. Add a config struct passed to the TUI launcher.

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Add TUIConfig struct and update RunPicker**

Add above `RunPicker`:

```go
// TUIConfig holds configuration needed by the TUI.
type TUIConfig struct {
	Editor          string // resolved editor command
	DefaultTemplate string // default template name
}
```

Add a new `RunTUI` function that launches with the today screen:

```go
// RunTUI launches the full interactive TUI starting at the today screen.
func RunTUI(store StorageProvider, cfg TUIConfig) error {
	m := newTUIModel(store, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return err
	}
	if pm, ok := result.(pickerModel); ok && pm.err != nil {
		return pm.err
	}
	return nil
}
```

Don't implement `newTUIModel` yet — just have it call `newPickerModel` with empty days for now. We'll build it out in Task 3.

**Step 2: Verify it compiles**

Run: `go build ./...`

**Step 3: Commit**

```bash
git add internal/ui/picker.go
git commit -m "feat(tui): add TUIConfig and RunTUI entry point"
```

---

### Task 3: Today Screen — Model and Data Loading

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Add screen constant and model fields**

Add `screenToday` to the `pickerScreen` enum (as value 0, shifting others):

```go
const (
	screenToday pickerScreen = iota
	screenDateList
	screenDayDetail
	screenEntryDetail
)
```

Add fields to `pickerModel`:

```go
type pickerModel struct {
	store    StorageProvider
	cfg      TUIConfig
	screen   pickerScreen
	// Today screen
	dailyEntry    *entry.Entry   // today's daily entry (nil if none)
	todayEntries  []entry.Entry  // other entries today (excluding daily)
	todayList     list.Model     // list for other today entries
	dailyViewport viewport.Model // viewport for daily entry content
	todayFocus    int            // 0=daily viewport, 1=entry list
	// Browse screens (existing)
	days     []storage.DaySummary
	dayIdx   int
	dateList list.Model
	dayList  list.Model
	viewport viewport.Model
	entry    entry.Entry
	// Common
	width    int
	height   int
	ready    bool
	err      error
}
```

**Step 2: Implement `newTUIModel`**

```go
func newTUIModel(store StorageProvider, cfg TUIConfig) pickerModel {
	m := pickerModel{
		store:  store,
		cfg:    cfg,
		screen: screenToday,
	}
	return m
}
```

**Step 3: Add `loadToday` method**

This loads today's data. It reuses the same logic as `daily.GetOrCreateToday` but without creating — just reads.

```go
type todayLoadedMsg struct {
	daily   *entry.Entry
	entries []entry.Entry
	err     error
}

func (m pickerModel) loadTodayCmd() tea.Msg {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	entries, err := m.store.List(storage.ListOptions{Date: &today})
	if err != nil {
		return todayLoadedMsg{err: err}
	}
	if len(entries) == 0 {
		return todayLoadedMsg{}
	}
	// First entry is the daily entry (oldest, shown inline)
	daily := entries[len(entries)-1] // oldest (list is newest-first)
	var others []entry.Entry
	if len(entries) > 1 {
		others = entries[:len(entries)-1]
	}
	return todayLoadedMsg{daily: &daily, entries: others}
}
```

**Step 4: Handle `todayLoadedMsg` in `Update`**

In the main `Update` method, add a case:

```go
case todayLoadedMsg:
	if msg.err != nil {
		m.err = msg.err
		return m, tea.Quit
	}
	m.dailyEntry = msg.daily
	m.todayEntries = msg.entries
	// Build today list
	items := make([]list.Item, len(msg.entries))
	for i, e := range msg.entries {
		items[i] = entryItem{entry: e}
	}
	m.todayList = list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.todayList.Title = ""
	m.todayList.SetShowHelp(false)
	// Build daily viewport
	if msg.daily != nil {
		m.dailyViewport = viewport.New(m.width, m.dailyViewportHeight())
		m.dailyViewport.SetContent(msg.daily.Content)
	}
	if m.ready {
		m.layoutToday()
	}
	return m, nil
```

**Step 5: Update `Init` to load today**

```go
func (m pickerModel) Init() tea.Cmd {
	if m.screen == screenToday {
		return m.loadTodayCmd
	}
	return nil
}
```

**Step 6: Verify it compiles**

Run: `go build ./...`

**Step 7: Commit**

```bash
git add internal/ui/picker.go
git commit -m "feat(tui): add today screen model and data loading"
```

---

### Task 4: Today Screen — View and Layout

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Add layout helpers**

```go
func (m pickerModel) dailyViewportHeight() int {
	maxHeight := m.height * 6 / 10 // 60% of terminal
	return maxHeight
}

func (m *pickerModel) layoutToday() {
	headerHeight := 2  // header + blank line
	footerHeight := 2  // help + blank line

	if m.dailyEntry != nil {
		vpHeight := m.dailyViewportHeight()
		m.dailyViewport.Width = m.width
		m.dailyViewport.Height = vpHeight
		m.dailyViewport.SetContent(m.dailyEntry.Content)

		listHeight := m.height - headerHeight - vpHeight - footerHeight - 1 // 1 for separator
		if listHeight < 3 {
			listHeight = 3
		}
		m.todayList.SetSize(m.width, listHeight)
	} else {
		m.todayList.SetSize(m.width, m.height-headerHeight-footerHeight)
	}
}
```

**Step 2: Add today view rendering**

In the `View()` method, add the `screenToday` case:

```go
case screenToday:
	if m.dailyEntry == nil && len(m.todayEntries) == 0 {
		// Empty state
		header := lipgloss.NewStyle().Bold(true).Render(
			fmt.Sprintf("Today — %s", time.Now().Format("2006-01-02")))
		empty := "\nNothing yet today.\n\n  j  jot a quick note\n  c  create a new entry\n"
		footer := helpStyle.Render("j jot  c create  b browse  x ctx  ? help")
		return header + empty + "\n" + footer
	}

	var sections []string

	// Header
	count := len(m.todayEntries)
	if m.dailyEntry != nil {
		count++
	}
	label := "entries"
	if count == 1 {
		label = "entry"
	}
	header := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("Today — %s    %d %s", time.Now().Format("2006-01-02"), count, label))
	sections = append(sections, header)

	// Daily entry viewport
	if m.dailyEntry != nil {
		sections = append(sections, m.dailyViewport.View())
	}

	// Other entries list
	if len(m.todayEntries) > 0 {
		sections = append(sections, m.todayList.View())
	}

	// Footer
	footer := helpStyle.Render("j jot  c create  e edit  b browse  x ctx  ? help")
	sections = append(sections, footer)

	return strings.Join(sections, "\n")
```

**Step 3: Handle WindowSizeMsg for today screen**

In the `tea.WindowSizeMsg` handler, add:

```go
case screenToday:
	m.layoutToday()
```

**Step 4: Verify it compiles**

Run: `go build ./...`

**Step 5: Commit**

```bash
git add internal/ui/picker.go
git commit -m "feat(tui): add today screen view and layout"
```

---

### Task 5: Today Screen — Navigation and Key Handling

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Add `updateToday` method**

```go
func (m pickerModel) updateToday(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "b":
		// Switch to browse (date list)
		return m.loadDateList()
	case "tab":
		// Toggle focus between daily viewport and entry list
		if m.dailyEntry != nil && len(m.todayEntries) > 0 {
			m.todayFocus = (m.todayFocus + 1) % 2
		}
		return m, nil
	case "enter":
		if m.todayFocus == 0 && m.dailyEntry != nil {
			// Edit daily entry in $EDITOR — handled in Task 7
			return m, nil
		}
		if m.todayFocus == 1 {
			if item, ok := m.todayList.SelectedItem().(entryItem); ok {
				return m.loadEntryDetail(item.entry.ID)
			}
		}
		return m, nil
	}

	// Pass to focused component
	var cmd tea.Cmd
	if m.todayFocus == 0 && m.dailyEntry != nil {
		m.dailyViewport, cmd = m.dailyViewport.Update(msg)
	} else if len(m.todayEntries) > 0 {
		m.todayList, cmd = m.todayList.Update(msg)
	}
	return m, cmd
}
```

**Step 2: Wire into Update's KeyMsg handler**

In the `tea.KeyMsg` switch, add:

```go
case screenToday:
	return m.updateToday(msg)
```

**Step 3: Add `loadDateList` method**

```go
func (m pickerModel) loadDateList() (tea.Model, tea.Cmd) {
	days, err := m.store.ListDays(storage.ListDaysOptions{})
	if err != nil {
		m.err = err
		return m, tea.Quit
	}
	m.days = days
	items := make([]list.Item, len(days))
	for i, d := range days {
		items[i] = dateItem{summary: d}
	}
	m.dateList = list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.dateList.Title = "Daily View"
	m.dateList.SetShowHelp(false)
	if m.ready {
		m.dateList.SetSize(m.width, m.height-2)
	}
	m.screen = screenDateList
	return m, nil
}
```

**Step 4: Update `esc` on date list to return to today**

In `updateDateList`, change the `esc`/`backspace` behavior. Currently date list has no back behavior. Add:

```go
case "esc", "backspace":
	m.screen = screenToday
	return m, m.loadTodayCmd
```

**Step 5: Verify it compiles, run tests**

Run: `go build ./... && go test ./...`

**Step 6: Commit**

```bash
git add internal/ui/picker.go
git commit -m "feat(tui): add today screen navigation and key handling"
```

---

### Task 6: Jot Action — Inline Text Input

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Add jot mode fields and imports**

Add to imports: `"github.com/charmbracelet/bubbles/textinput"`

Add to `pickerModel`:

```go
	// Jot mode
	jotInput  textinput.Model
	jotActive bool
```

**Step 2: Add jot message types**

```go
type jotCompleteMsg struct {
	err error
}
```

**Step 3: Handle `j` key globally**

In the `tea.KeyMsg` handler in `Update`, before the screen-specific switch, add global key handling:

```go
case tea.KeyMsg:
	// Jot input mode — intercept all keys
	if m.jotActive {
		return m.updateJotInput(msg)
	}

	// Global keys (work from any screen when not in input mode)
	switch msg.String() {
	case "j":
		return m.startJot()
	case "?":
		// Help overlay — Task 10
	case "x":
		// Context panel — Task 9
	}

	// Screen-specific handling
	switch m.screen { ... }
```

**Step 4: Implement jot methods**

```go
func (m pickerModel) startJot() (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "jot a quick note..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = m.width - 4
	m.jotInput = ti
	m.jotActive = true
	return m, textinput.Blink
}

func (m pickerModel) updateJotInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		content := strings.TrimSpace(m.jotInput.Value())
		if content == "" {
			m.jotActive = false
			return m, nil
		}
		m.jotActive = false
		return m, func() tea.Msg {
			return m.doJot(content)
		}
	case "esc":
		m.jotActive = false
		return m, nil
	}

	var cmd tea.Cmd
	m.jotInput, cmd = m.jotInput.Update(msg)
	return m, cmd
}

func (m pickerModel) doJot(content string) tea.Msg {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Get or create today's entry
	entries, err := m.store.List(storage.ListOptions{Date: &today, Limit: 1})
	if err != nil {
		return jotCompleteMsg{err: err}
	}

	timestamp := now.Format("15:04")
	jotLine := fmt.Sprintf("- **%s** %s", timestamp, content)

	if len(entries) > 0 {
		// Append to existing daily entry (oldest)
		allEntries, err := m.store.List(storage.ListOptions{Date: &today})
		if err != nil {
			return jotCompleteMsg{err: err}
		}
		daily := allEntries[len(allEntries)-1] // oldest
		var newContent string
		if strings.TrimSpace(daily.Content) == "" {
			newContent = jotLine
		} else {
			newContent = daily.Content + "\n" + jotLine
		}
		_, err = m.store.Update(daily.ID, newContent, nil)
		if err != nil {
			return jotCompleteMsg{err: err}
		}
	} else {
		// Create new daily entry
		id, err := entry.NewID()
		if err != nil {
			return jotCompleteMsg{err: err}
		}
		nowUTC := now.UTC()
		e := entry.Entry{
			ID:        id,
			Content:   fmt.Sprintf("# %s\n\n%s", now.Format("2006-01-02"), jotLine),
			CreatedAt: nowUTC,
			UpdatedAt: nowUTC,
		}
		if err := m.store.Create(e); err != nil {
			return jotCompleteMsg{err: err}
		}
	}

	return jotCompleteMsg{}
}
```

**Step 5: Handle `jotCompleteMsg` — refresh today**

```go
case jotCompleteMsg:
	if msg.err != nil {
		m.err = msg.err
		return m, tea.Quit
	}
	// Refresh current screen
	return m, m.loadTodayCmd
```

**Step 6: Render jot input in View**

In each screen's view, when `m.jotActive`, append the jot input at the bottom:

```go
// At the end of View(), before returning:
if m.jotActive {
	return result + "\n" + m.jotInput.View()
}
```

**Step 7: Verify it compiles**

Run: `go build ./...`

**Step 8: Commit**

```bash
git add internal/ui/picker.go
git commit -m "feat(tui): add inline jot action with text input"
```

---

### Task 7: Create and Edit Actions — Editor Suspension

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Add imports**

Add to imports:
```go
"os/exec"
"github.com/chris-regnier/diaryctl/internal/editor"
```

**Step 2: Add message types**

```go
type editorFinishedMsg struct {
	err error
}
```

**Step 3: Implement create action**

Handle `c` key in the global handler:

```go
case "c":
	return m.startCreate()
```

```go
func (m pickerModel) startCreate() (tea.Model, tea.Cmd) {
	editorCmd := editor.ResolveEditor(m.cfg.Editor)
	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		return m, nil
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "diaryctl-*.md")
	if err != nil {
		m.err = err
		return m, tea.Quit
	}
	tmpName := tmpFile.Name()
	tmpFile.Close()

	cmdArgs := append(parts[1:], tmpName)
	c := exec.Command(parts[0], cmdArgs...)
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpName)
		if err != nil {
			return editorFinishedMsg{err: err}
		}
		data, err := os.ReadFile(tmpName)
		if err != nil {
			return editorFinishedMsg{err: err}
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			return editorFinishedMsg{} // no-op
		}
		id, err := entry.NewID()
		if err != nil {
			return editorFinishedMsg{err: err}
		}
		now := time.Now().UTC()
		e := entry.Entry{
			ID:        id,
			Content:   content,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := m.store.Create(e); err != nil {
			return editorFinishedMsg{err: err}
		}
		return editorFinishedMsg{}
	})
}
```

**Step 4: Implement edit action**

Handle `e` key in screen-specific handlers (day detail, entry view) and `enter` on today screen daily entry:

```go
func (m pickerModel) startEdit(e entry.Entry) (tea.Model, tea.Cmd) {
	editorCmd := editor.ResolveEditor(m.cfg.Editor)
	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		return m, nil
	}

	tmpFile, err := os.CreateTemp("", "diaryctl-*.md")
	if err != nil {
		m.err = err
		return m, tea.Quit
	}
	tmpName := tmpFile.Name()
	tmpFile.WriteString(e.Content)
	tmpFile.Close()

	cmdArgs := append(parts[1:], tmpName)
	c := exec.Command(parts[0], cmdArgs...)
	entryID := e.ID
	originalContent := e.Content
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpName)
		if err != nil {
			return editorFinishedMsg{err: err}
		}
		data, err := os.ReadFile(tmpName)
		if err != nil {
			return editorFinishedMsg{err: err}
		}
		content := strings.TrimSpace(string(data))
		if content == "" || content == strings.TrimSpace(originalContent) {
			return editorFinishedMsg{} // no change
		}
		if _, err := m.store.Update(entryID, content, nil); err != nil {
			return editorFinishedMsg{err: err}
		}
		return editorFinishedMsg{}
	})
}
```

**Step 5: Wire `enter` on today screen to edit daily entry**

Update `updateToday`:

```go
case "enter":
	if m.todayFocus == 0 && m.dailyEntry != nil {
		return m.startEdit(*m.dailyEntry)
	}
	// ... existing list item selection
```

**Step 6: Wire `e` key in day detail and entry view**

In `updateDayDetail`:
```go
case "e":
	if item, ok := m.dayList.SelectedItem().(entryItem); ok {
		return m.startEdit(item.entry)
	}
```

In `updateEntryDetail`:
```go
case "e":
	return m.startEdit(m.entry)
```

**Step 7: Handle `editorFinishedMsg` — refresh**

```go
case editorFinishedMsg:
	if msg.err != nil {
		m.err = msg.err
		return m, tea.Quit
	}
	return m, m.refreshCurrentScreen
```

Add a refresh helper:

```go
func (m pickerModel) refreshCurrentScreen() tea.Msg {
	switch m.screen {
	case screenToday:
		return m.loadTodayCmd()
	case screenDayDetail:
		// Reload day detail
		day := m.days[m.dayIdx]
		date := day.Date
		entries, err := m.store.List(storage.ListOptions{Date: &date})
		if err != nil {
			return todayLoadedMsg{err: err}
		}
		// Return a message that reloads day detail
		return dayRefreshMsg{entries: entries}
	default:
		return m.loadTodayCmd()
	}
}
```

**Step 8: Verify it compiles**

Run: `go build ./...`

**Step 9: Commit**

```bash
git add internal/ui/picker.go
git commit -m "feat(tui): add create and edit actions via editor suspension"
```

---

### Task 8: Delete Action — Inline Confirmation

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Add delete mode fields**

```go
	// Delete confirmation mode
	deleteActive bool
	deleteEntry  entry.Entry
```

**Step 2: Add message type**

```go
type deleteCompleteMsg struct {
	err error
}
```

**Step 3: Handle `d` key in day detail and entry view**

In `updateDayDetail`:
```go
case "d":
	if item, ok := m.dayList.SelectedItem().(entryItem); ok {
		m.deleteActive = true
		m.deleteEntry = item.entry
		return m, nil
	}
```

In `updateEntryDetail`:
```go
case "d":
	m.deleteActive = true
	m.deleteEntry = m.entry
	return m, nil
```

**Step 4: Handle delete confirmation keys**

In the global `tea.KeyMsg` handler, after jot check:

```go
if m.deleteActive {
	return m.updateDeleteConfirm(msg)
}
```

```go
func (m pickerModel) updateDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y":
		m.deleteActive = false
		id := m.deleteEntry.ID
		return m, func() tea.Msg {
			if err := m.store.Delete(id); err != nil {
				return deleteCompleteMsg{err: err}
			}
			return deleteCompleteMsg{}
		}
	case "n", "esc":
		m.deleteActive = false
		return m, nil
	}
	return m, nil
}
```

**Step 5: Handle `deleteCompleteMsg`**

```go
case deleteCompleteMsg:
	if msg.err != nil {
		m.err = msg.err
		return m, tea.Quit
	}
	// Go back to parent screen and refresh
	if m.screen == screenEntryDetail {
		m.screen = screenDayDetail
	}
	return m, m.refreshCurrentScreen
```

**Step 6: Render delete confirmation in View**

When `m.deleteActive`, replace the footer with:

```go
if m.deleteActive {
	prompt := fmt.Sprintf("Delete entry %s? [y/N] ", m.deleteEntry.ID)
	return result + "\n" + warningStyle.Render(prompt)
}
```

**Step 7: Verify it compiles**

Run: `go build ./...`

**Step 8: Commit**

```bash
git add internal/ui/picker.go
git commit -m "feat(tui): add delete action with inline confirmation"
```

---

### Task 9: Context Panel Overlay

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Add screen constant and model fields**

Add `screenContextPanel` to the enum.

Add to `pickerModel`:

```go
	// Context panel
	contextList     list.Model
	contextEntryID  string   // entry being context-managed (empty = browse mode)
	contextItems    []storage.Context
	contextAttached map[string]bool // contextID -> attached to current entry
	prevScreen      pickerScreen    // screen to return to on esc
	contextInput    textinput.Model
	contextCreating bool
```

**Step 2: Add contextItem list adapter**

```go
type contextItem struct {
	ctx      storage.Context
	attached bool
}

func (c contextItem) Title() string {
	marker := "○"
	if c.attached {
		marker = "●"
	}
	return fmt.Sprintf("%s %s", marker, c.ctx.Name)
}

func (c contextItem) Description() string { return c.ctx.Source }
func (c contextItem) FilterValue() string { return c.ctx.Name }
```

**Step 3: Handle `x` key globally**

```go
case "x":
	return m.openContextPanel()
```

```go
func (m pickerModel) openContextPanel() (tea.Model, tea.Cmd) {
	m.prevScreen = m.screen

	// Determine if we have a selected entry
	switch m.screen {
	case screenToday:
		if m.todayFocus == 0 && m.dailyEntry != nil {
			m.contextEntryID = m.dailyEntry.ID
		} else if m.todayFocus == 1 {
			if item, ok := m.todayList.SelectedItem().(entryItem); ok {
				m.contextEntryID = item.entry.ID
			}
		}
	case screenDayDetail:
		if item, ok := m.dayList.SelectedItem().(entryItem); ok {
			m.contextEntryID = item.entry.ID
		}
	case screenEntryDetail:
		m.contextEntryID = m.entry.ID
	default:
		m.contextEntryID = ""
	}

	return m, func() tea.Msg { return m.loadContexts() }
}

type contextsLoadedMsg struct {
	contexts []storage.Context
	attached map[string]bool
	err      error
}

func (m pickerModel) loadContexts() tea.Msg {
	contexts, err := m.store.ListContexts()
	if err != nil {
		return contextsLoadedMsg{err: err}
	}

	attached := make(map[string]bool)
	if m.contextEntryID != "" {
		e, err := m.store.Get(m.contextEntryID)
		if err == nil {
			for _, ref := range e.Contexts {
				attached[ref.ContextID] = true
			}
		}
	}

	return contextsLoadedMsg{contexts: contexts, attached: attached}
}
```

**Step 4: Handle `contextsLoadedMsg`**

```go
case contextsLoadedMsg:
	if msg.err != nil {
		m.err = msg.err
		return m, tea.Quit
	}
	m.contextItems = msg.contexts
	m.contextAttached = msg.attached

	items := make([]list.Item, len(msg.contexts))
	for i, c := range msg.contexts {
		items[i] = contextItem{ctx: c, attached: msg.attached[c.ID]}
	}

	title := "Contexts"
	if m.contextEntryID != "" {
		title = fmt.Sprintf("Contexts for %s", m.contextEntryID)
	}
	m.contextList = list.New(items, list.NewDefaultDelegate(), m.width-4, m.height-6)
	m.contextList.Title = title
	m.contextList.SetShowHelp(false)
	m.screen = screenContextPanel
	return m, nil
```

**Step 5: Implement context panel key handling**

```go
func (m pickerModel) updateContextPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.contextCreating {
		return m.updateContextCreate(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = m.prevScreen
		return m, m.refreshCurrentScreen
	case "enter":
		if m.contextEntryID != "" {
			// Toggle context attachment
			if item, ok := m.contextList.SelectedItem().(contextItem); ok {
				return m, func() tea.Msg {
					if item.attached {
						err := m.store.DetachContext(m.contextEntryID, item.ctx.ID)
						return contextsLoadedMsg{err: err}
					}
					err := m.store.AttachContext(m.contextEntryID, item.ctx.ID)
					if err != nil {
						return contextsLoadedMsg{err: err}
					}
					return m.loadContexts()
				}
			}
		}
	case "n":
		// Create new context
		ti := textinput.New()
		ti.Placeholder = "context name..."
		ti.Focus()
		ti.CharLimit = 100
		ti.Width = m.width - 8
		m.contextInput = ti
		m.contextCreating = true
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	m.contextList, cmd = m.contextList.Update(msg)
	return m, cmd
}

func (m pickerModel) updateContextCreate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		name := strings.TrimSpace(m.contextInput.Value())
		m.contextCreating = false
		if name == "" {
			return m, nil
		}
		return m, func() tea.Msg {
			id, err := entry.NewID()
			if err != nil {
				return contextsLoadedMsg{err: err}
			}
			now := time.Now().UTC()
			c := storage.Context{
				ID:        id,
				Name:      name,
				Source:    "manual",
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := m.store.CreateContext(c); err != nil {
				return contextsLoadedMsg{err: err}
			}
			// Auto-attach if we have an entry selected
			if m.contextEntryID != "" {
				m.store.AttachContext(m.contextEntryID, id)
			}
			return m.loadContexts()
		}
	case "esc":
		m.contextCreating = false
		return m, nil
	}

	var cmd tea.Cmd
	m.contextInput, cmd = m.contextInput.Update(msg)
	return m, cmd
}
```

**Step 6: Add context panel View**

```go
case screenContextPanel:
	var b strings.Builder
	b.WriteString(m.contextList.View())
	if m.contextCreating {
		b.WriteString("\n" + m.contextInput.View())
	} else {
		hint := "enter toggle  n new  / filter  esc close"
		if m.contextEntryID == "" {
			hint = "enter filter  n new  / search  esc close"
		}
		b.WriteString("\n" + helpStyle.Render(hint))
	}
	return b.String()
```

**Step 7: Wire into screen switch in Update**

```go
case screenContextPanel:
	return m.updateContextPanel(msg)
```

**Step 8: Verify it compiles**

Run: `go build ./...`

**Step 9: Commit**

```bash
git add internal/ui/picker.go
git commit -m "feat(tui): add context panel overlay with attach/detach/create"
```

---

### Task 10: Help Overlay

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Add help state**

```go
	// Help overlay
	helpActive bool
```

**Step 2: Handle `?` key globally**

In the global handler:

```go
case "?":
	m.helpActive = !m.helpActive
	return m, nil
```

Also handle dismissal:

```go
if m.helpActive {
	switch msg.String() {
	case "?", "esc":
		m.helpActive = false
		return m, nil
	}
	// Swallow all other keys while help is shown
	return m, nil
}
```

This should be checked BEFORE jot/delete active checks.

**Step 3: Render help overlay in View**

At the end of `View()`, if `m.helpActive`, overlay the help text:

```go
func (m pickerModel) helpOverlay() string {
	help := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(48).
		Render(`Navigation
  ↑/↓        navigate / scroll
  enter      select / edit daily entry
  esc        go back
  b          browse date list
  ←/→ p/n    prev / next day
  tab        switch focus (today)

Actions
  j          jot a quick note
  c          create new entry
  e          edit selected entry
  d          delete selected entry
  /          search / filter
  x          context panel

  q          quit     ? close help`)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, help)
}
```

In `View()`:

```go
func (m pickerModel) View() string {
	if !m.ready {
		return "Loading..."
	}
	if m.helpActive {
		return m.helpOverlay()
	}
	// ... rest of view
}
```

**Step 4: Verify it compiles**

Run: `go build ./...`

**Step 5: Commit**

```bash
git add internal/ui/picker.go
git commit -m "feat(tui): add help overlay with keybinding reference"
```

---

### Task 11: Wire Root Command to Launch TUI

**Files:**
- Modify: `cmd/root.go`

**Step 1: Add RunE to rootCmd**

```go
var rootCmd = &cobra.Command{
	Use:   "diaryctl",
	Short: "A diary management CLI tool",
	Long:  "diaryctl is a command-line tool for managing personal diary entries with pluggable storage backends.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// ... existing config/storage init (unchanged)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			// Non-TTY: fall back to today's entry
			return todayRun(os.Stdout, false, false)
		}
		return ui.RunTUI(store, ui.TUIConfig{
			Editor:          editor.ResolveEditor(appConfig.Editor),
			DefaultTemplate: appConfig.DefaultTemplate,
		})
	},
}
```

Add imports to `cmd/root.go`:

```go
"github.com/chris-regnier/diaryctl/internal/editor"
"github.com/chris-regnier/diaryctl/internal/ui"
"golang.org/x/term"
```

**Step 2: Verify it compiles**

Run: `go build ./...`

**Step 3: Test manually**

Run: `./diaryctl` in a terminal — should launch TUI with today screen.
Run: `./diaryctl | cat` — should print today's entry as plain text.
Run: `./diaryctl daily` — should still launch the date list picker.

**Step 4: Commit**

```bash
git add cmd/root.go
git commit -m "feat(tui): wire bare diaryctl to launch TUI in TTY mode"
```

---

### Task 12: Integration Test and Polish

**Files:**
- Modify: `internal/ui/picker.go` (any fixes)

**Step 1: Run full test suite**

Run: `go test ./... -v`

Fix any compilation errors or test failures.

**Step 2: Run `go vet`**

Run: `go vet ./...`

**Step 3: Run `gofmt`**

Run: `gofmt -l .` — fix any formatting issues with `gofmt -w .`

**Step 4: Manual smoke test**

Test each action:
- `diaryctl` → today screen appears
- `j` → type note → enter → jot appears
- `c` → editor opens → save → entry created
- `b` → date list → enter → day detail → enter → entry view
- `e` → editor opens → save → entry updated
- `d` → `y` → entry deleted
- `x` → context panel → `n` → create context → `enter` → toggle
- `?` → help overlay → `?` → dismiss
- `esc` → navigates back at every level

**Step 5: Commit**

```bash
git add -A
git commit -m "feat(tui): polish and integration fixes"
```

---

### Summary of Commits

1. `feat(tui): expand StorageProvider interface for write and context ops`
2. `feat(tui): add TUIConfig and RunTUI entry point`
3. `feat(tui): add today screen model and data loading`
4. `feat(tui): add today screen view and layout`
5. `feat(tui): add today screen navigation and key handling`
6. `feat(tui): add inline jot action with text input`
7. `feat(tui): add create and edit actions via editor suspension`
8. `feat(tui): add delete action with inline confirmation`
9. `feat(tui): add context panel overlay with attach/detach/create`
10. `feat(tui): add help overlay with keybinding reference`
11. `feat(tui): wire bare diaryctl to launch TUI in TTY mode`
12. `feat(tui): polish and integration fixes`
