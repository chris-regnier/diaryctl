# TUI Template Integration Plan

**Date:** 2026-02-11
**Status:** Draft

## Overview

Integrate templates into the TUI to allow users to:
1. Select templates when creating new entries
2. Append template content when editing entries
3. Use templates for "blocks on the fly" - quickly inserting structured content

## Current State

**What exists:**
- Full template CRUD via CLI (`diaryctl template {list|show|create|edit|delete}`)
- Template composition in `internal/template/` package
- `TUIConfig.DefaultTemplate` passed to TUI
- `doJot()` references `cfg.DefaultTemplate` but only stores name (no ID lookup)
- `startCreate()` ignores templates entirely

**Gaps:**
1. No template picker/selector in TUI
2. `startCreate()` doesn't pre-fill editor with template content
3. `startEdit()` has no option to append template content
4. Jot creates entries with incomplete `TemplateRef` (missing `TemplateID`)
5. No way to browse/select templates interactively

## Design

### Template Picker Component

A reusable list-based picker for selecting one or more templates:

```
┌─ Select Template(s) ────────────────────────┐
│  [ ] daily          # Daily Entry           │
│  [x] prompts        ## Reflection Prompts   │
│  [ ] standup        ## Standup Notes        │
│                                             │
│  space toggle  enter confirm  esc cancel    │
└─────────────────────────────────────────────┘
```

- Multi-select with `space` to toggle
- `enter` confirms selection and returns `[]string` names
- `esc` cancels (returns nil)
- Preview shows first line of template content

### Integration Points

#### 1. Create Action (`c` key)

Before opening the editor:
1. If `DefaultTemplate` configured → use it (current behavior, but with proper TemplateRef)
2. If no default → show template picker (optional)
3. User can skip with `esc` or select templates with `space`+`enter`
4. Compose selected templates, pre-fill editor buffer
5. Store `TemplateRef` with both ID and name

Flow:
```
c pressed → Template picker → [optional selection] → Editor opens with content → Save → Entry created with attribution
```

#### 2. Edit Action with Template (`t` key or `e` with modifier)

New action to append template content to existing entry:
- `T` (shift+t) on selected entry → Template picker → Append to content → Editor opens
- Alternatively: `e` opens editor, new `t` key in editor... (but we can't control editor)

Simpler approach:
- `t` key opens template picker for selected entry
- After selection, appends template content and opens editor
- Merges template refs (deduplicates)

#### 3. Quick Template Insert (`t` from today screen)

On today screen with daily entry focused:
- `t` opens template picker
- Selected template content appended to daily entry
- Opens editor for review/edit

#### 4. Jot with Proper Template Attribution

Fix `doJot()` to:
- Look up template by name to get ID
- Store complete `TemplateRef{TemplateID, TemplateName}`

### StorageProvider Extension

Add template read method:

```go
type StorageProvider interface {
    // ... existing methods ...
    
    // Template (read-only for TUI)
    ListTemplates() ([]storage.Template, error)
    GetTemplateByName(name string) (storage.Template, error)
}
```

### New Model Fields

```go
type pickerModel struct {
    // ... existing fields ...
    
    // Template picker
    templatePicker     list.Model
    templatePickerMode bool
    templateMultiSelect map[string]bool // name -> selected
    templateCallback   func([]string)   // called with selected names
}
```

### Key Bindings

| Key | Context | Action |
|-----|---------|--------|
| `c` | Any screen | Create with template picker (if templates exist) |
| `t` | Entry selected | Append template to entry |
| `T` | Entry selected | Same as `t` (alias) |

### Help Overlay Update

Add to help:
```
Actions
  ...
  t          append template to entry
```

---

## Implementation Tasks

### Task 1: Extend StorageProvider Interface

**Files:**
- Modify: `internal/ui/picker.go`

Add template methods to `StorageProvider`:

```go
type StorageProvider interface {
    // ... existing ...
    
    // Template
    ListTemplates() ([]storage.Template, error)
    GetTemplateByName(name string) (storage.Template, error)
}
```

This is already satisfied by `storage.Storage`.

**Commit:** `feat(tui): extend StorageProvider with template methods`

---

### Task 2: Add Template Picker Model

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Add templateItem list adapter**

```go
type templateItem struct {
    tmpl     storage.Template
    selected bool
}

func (t templateItem) Title() string {
    marker := "○"
    if t.selected {
        marker = "●"
    }
    return fmt.Sprintf("%s %s", marker, t.tmpl.Name)
}

func (t templateItem) Description() string {
    // First line preview, max 60 chars
    lines := strings.SplitN(t.tmpl.Content, "\n", 2)
    preview := lines[0]
    if len(preview) > 60 {
        preview = preview[:57] + "..."
    }
    return preview
}

func (t templateItem) FilterValue() string { return t.tmpl.Name }
```

**Step 2: Add model fields**

```go
type pickerModel struct {
    // ... existing ...
    
    // Template picker overlay
    templateList        list.Model
    templatePickerActive bool
    templateSelected    map[string]bool      // name -> selected
    templateItems       []storage.Template
    templateCallback    templateCallbackFunc // what to do after selection
}

type templateCallbackFunc func(m *pickerModel, names []string) tea.Cmd
```

**Step 3: Add message types**

```go
type templatesLoadedMsg struct {
    templates []storage.Template
    err       error
}

type templatePickerDoneMsg struct {
    names []string // nil = cancelled
}
```

**Commit:** `feat(tui): add template picker model and types`

---

### Task 3: Implement Template Picker Loading and View

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Load templates**

```go
func (m pickerModel) loadTemplatesCmd() tea.Msg {
    templates, err := m.store.ListTemplates()
    if err != nil {
        return templatesLoadedMsg{err: err}
    }
    return templatesLoadedMsg{templates: templates}
}
```

**Step 2: Handle templatesLoadedMsg**

```go
case templatesLoadedMsg:
    if msg.err != nil {
        m.err = msg.err
        return m, tea.Quit
    }
    m.templateItems = msg.templates
    m.templateSelected = make(map[string]bool)
    
    items := make([]list.Item, len(msg.templates))
    for i, t := range msg.templates {
        items[i] = templateItem{tmpl: t, selected: false}
    }
    
    m.templateList = list.New(items, list.NewDefaultDelegate(), m.contentWidth()-4, m.height/2)
    m.templateList.Title = "Select Template(s)"
    m.templateList.SetShowHelp(false)
    m.templatePickerActive = true
    return m, nil
```

**Step 3: Template picker view**

```go
func (m pickerModel) templatePickerView() string {
    // Rebuild items with current selection state
    items := make([]list.Item, len(m.templateItems))
    for i, t := range m.templateItems {
        items[i] = templateItem{tmpl: t, selected: m.templateSelected[t.Name]}
    }
    m.templateList.SetItems(items)
    
    var b strings.Builder
    b.WriteString(m.templateList.View())
    b.WriteString("\n")
    b.WriteString(helpStyle.Render("space toggle  enter confirm  esc skip"))
    
    // Wrap in a box
    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        Padding(1).
        Render(b.String())
}
```

**Step 4: Render in View()**

In `View()`, after help check but before screen rendering:

```go
if m.templatePickerActive {
    // Overlay template picker centered
    picker := m.templatePickerView()
    return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, picker)
}
```

**Commit:** `feat(tui): implement template picker loading and view`

---

### Task 4: Implement Template Picker Key Handling

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Handle keys when picker active**

In the main `Update` KeyMsg handler, add early check:

```go
case tea.KeyMsg:
    if m.templatePickerActive {
        return m.updateTemplatePicker(msg)
    }
    // ... rest of key handling
```

**Step 2: Implement updateTemplatePicker**

```go
func (m pickerModel) updateTemplatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "esc":
        m.templatePickerActive = false
        // Callback with nil = cancelled/skipped
        if m.templateCallback != nil {
            return m, m.templateCallback(&m, nil)
        }
        return m, nil
        
    case " ", "space":
        // Toggle selection
        if item, ok := m.templateList.SelectedItem().(templateItem); ok {
            name := item.tmpl.Name
            m.templateSelected[name] = !m.templateSelected[name]
            // Refresh list items
            items := make([]list.Item, len(m.templateItems))
            for i, t := range m.templateItems {
                items[i] = templateItem{tmpl: t, selected: m.templateSelected[t.Name]}
            }
            m.templateList.SetItems(items)
        }
        return m, nil
        
    case "enter":
        m.templatePickerActive = false
        // Collect selected names
        var names []string
        for name, selected := range m.templateSelected {
            if selected {
                names = append(names, name)
            }
        }
        // Sort for consistent ordering
        sort.Strings(names)
        if m.templateCallback != nil {
            return m, m.templateCallback(&m, names)
        }
        return m, nil
    }
    
    var cmd tea.Cmd
    m.templateList, cmd = m.templateList.Update(msg)
    return m, cmd
}
```

**Commit:** `feat(tui): implement template picker key handling`

---

### Task 5: Integrate Template Picker with Create Action

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Modify startCreate to use template picker**

Replace `startCreate()`:

```go
func (m pickerModel) startCreate() (tea.Model, tea.Cmd) {
    // Check if templates exist
    return m, func() tea.Msg {
        templates, err := m.store.ListTemplates()
        if err != nil || len(templates) == 0 {
            // No templates, proceed directly to editor
            return templatePickerDoneMsg{names: nil}
        }
        return templatesLoadedMsg{templates: templates}
    }
}
```

**Step 2: Set callback when templates loaded**

In `templatesLoadedMsg` handler, set the callback:

```go
case templatesLoadedMsg:
    // ... existing setup code ...
    
    // Set callback for create flow
    m.templateCallback = createWithTemplatesCallback
    return m, nil
```

**Step 3: Implement create callback**

```go
func createWithTemplatesCallback(m *pickerModel, names []string) tea.Cmd {
    return func() tea.Msg {
        var content string
        var refs []entry.TemplateRef
        
        if len(names) > 0 {
            c, r, err := template.Compose(m.store, names)
            if err != nil {
                return editorFinishedMsg{err: err}
            }
            content = c
            refs = r
        }
        
        // Now open editor with content
        return openEditorForCreate{content: content, refs: refs}
    }
}

type openEditorForCreate struct {
    content string
    refs    []entry.TemplateRef
}
```

**Step 4: Handle openEditorForCreate**

```go
case openEditorForCreate:
    return m.doCreateWithEditor(msg.content, msg.refs)

func (m pickerModel) doCreateWithEditor(initialContent string, refs []entry.TemplateRef) (tea.Model, tea.Cmd) {
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
    
    // Write initial content
    if initialContent != "" {
        tmpFile.WriteString(initialContent)
    }
    tmpFile.Close()
    
    cmdArgs := append(parts[1:], tmpName)
    c := exec.Command(parts[0], cmdArgs...)
    templateRefs := refs // capture for closure
    
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
            Templates: templateRefs,
        }
        if err := m.store.Create(e); err != nil {
            return editorFinishedMsg{err: err}
        }
        return editorFinishedMsg{}
    })
}
```

**Step 5: Handle templatePickerDoneMsg for direct create**

```go
case templatePickerDoneMsg:
    // Direct to editor (no templates selected or no templates exist)
    return m.doCreateWithEditor("", nil)
```

**Commit:** `feat(tui): integrate template picker with create action`

---

### Task 6: Add Template Append Action (`t` key)

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Handle `t` key in screen handlers**

In `updateToday`, `updateDayDetail`, `updateEntryDetail`:

```go
case "t", "T":
    // Get selected entry
    var targetEntry *entry.Entry
    // ... determine targetEntry based on screen/focus ...
    if targetEntry != nil {
        m.editTargetEntry = targetEntry
        return m, m.loadTemplatesForAppend
    }
```

**Step 2: Add editTargetEntry field**

```go
type pickerModel struct {
    // ...
    editTargetEntry *entry.Entry // entry being edited with template append
}
```

**Step 3: Implement append flow**

```go
func (m pickerModel) loadTemplatesForAppend() tea.Msg {
    templates, err := m.store.ListTemplates()
    if err != nil {
        return templatesLoadedMsg{err: err}
    }
    if len(templates) == 0 {
        return editorFinishedMsg{err: fmt.Errorf("no templates available")}
    }
    return templatesLoadedMsg{templates: templates}
}
```

When templates loaded for append, set callback:

```go
m.templateCallback = appendTemplatesCallback
```

**Step 4: Implement append callback**

```go
func appendTemplatesCallback(m *pickerModel, names []string) tea.Cmd {
    if len(names) == 0 || m.editTargetEntry == nil {
        return nil
    }
    
    return func() tea.Msg {
        c, refs, err := template.Compose(m.store, names)
        if err != nil {
            return editorFinishedMsg{err: err}
        }
        
        // Append to existing content
        newContent := m.editTargetEntry.Content
        if newContent != "" && !strings.HasSuffix(newContent, "\n") {
            newContent += "\n"
        }
        newContent += "\n" + c
        
        // Merge refs (deduplicate)
        existingRefs := make(map[string]bool)
        for _, r := range m.editTargetEntry.Templates {
            existingRefs[r.TemplateID] = true
        }
        mergedRefs := m.editTargetEntry.Templates
        for _, r := range refs {
            if !existingRefs[r.TemplateID] {
                mergedRefs = append(mergedRefs, r)
            }
        }
        
        return openEditorForEdit{
            entry:   *m.editTargetEntry,
            content: newContent,
            refs:    mergedRefs,
        }
    }
}

type openEditorForEdit struct {
    entry   entry.Entry
    content string
    refs    []entry.TemplateRef
}
```

**Step 5: Handle openEditorForEdit**

```go
case openEditorForEdit:
    return m.doEditWithEditor(msg.entry.ID, msg.content, msg.refs)

func (m pickerModel) doEditWithEditor(entryID string, content string, refs []entry.TemplateRef) (tea.Model, tea.Cmd) {
    // Similar to existing startEdit but with custom content and refs
    // ...
}
```

**Commit:** `feat(tui): add template append action (t key)`

---

### Task 7: Fix Jot Template Attribution

**Files:**
- Modify: `internal/ui/picker.go`

**Step 1: Update doJot to look up template**

In `doJot()`, when `cfg.DefaultTemplate` is set:

```go
func (m pickerModel) doJot(content string) tea.Msg {
    // ... existing code ...
    
    if len(entries) == 0 {
        // Create new daily entry
        var templateRefs []entry.TemplateRef
        
        // Look up template to get proper TemplateRef
        if m.cfg.DefaultTemplate != "" {
            names := template.ParseNames(m.cfg.DefaultTemplate)
            _, refs, err := template.Compose(m.store, names)
            if err != nil {
                // Warn but continue - match CLI behavior
                fmt.Fprintf(os.Stderr, "Warning: template %q not found\n", m.cfg.DefaultTemplate)
            } else {
                templateRefs = refs
            }
        }
        
        e := entry.Entry{
            ID:        id,
            Content:   fmt.Sprintf("# %s\n\n%s", now.Format("2006-01-02"), jotLine),
            CreatedAt: nowUTC,
            UpdatedAt: nowUTC,
            Templates: templateRefs,
        }
        // ...
    }
}
```

**Commit:** `fix(tui): store complete TemplateRef in jot entries`

---

### Task 8: Update Help Overlay

**Files:**
- Modify: `internal/ui/picker.go`

Update `helpOverlay()` to include template action:

```go
Actions
  j          jot a quick note (^J=newline)
  c          create new entry (with template picker)
  e          edit selected entry
  t          append template to entry
  d          delete selected entry
```

**Commit:** `docs(tui): add template action to help overlay`

---

### Task 9: Add Tests

**Files:**
- Create: `internal/ui/template_picker_test.go`

Test:
- Template picker shows all templates
- Space toggles selection
- Enter returns selected names
- Esc returns nil
- Create flow with templates pre-fills editor
- Append flow merges template refs

**Commit:** `test(tui): add template picker tests`

---

### Task 10: Polish and Integration Testing

1. Run full test suite: `go test ./...`
2. Run vet: `go vet ./...`
3. Run gofmt: `gofmt -w .`
4. Manual testing of all template flows

**Commit:** `chore(tui): polish template integration`

---

## Summary

This plan adds comprehensive template integration to the TUI:

1. **Template Picker** - Multi-select overlay for choosing templates
2. **Create with Templates** - `c` shows picker, pre-fills editor
3. **Append Templates** - `t` on entry appends template content
4. **Fixed Attribution** - Proper `TemplateRef` with ID and name
5. **Help Updates** - Documents new `t` keybinding

Total: 10 tasks, ~15 commits
