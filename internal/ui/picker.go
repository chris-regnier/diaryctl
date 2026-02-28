package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/template"
)

// pickerScreen represents the current screen state.
type pickerScreen int

const (
	screenToday pickerScreen = iota
	screenDateList
	screenDayDetail
	screenEntryDetail
	screenContextPanel
)

// Focus states for today screen
const (
	focusDailyViewport = 0
	focusEntryList     = 1
)

// Input validation limits
const (
	maxContextNameLength = 100
	maxJotInputLength    = 500
)

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

	// Template
	ListTemplates() ([]storage.Template, error)
	GetTemplateByName(name string) (storage.Template, error)
}

// dateItem implements list.Item for DaySummary.
type dateItem struct {
	summary storage.DaySummary
}

func (d dateItem) Title() string {
	label := "entries"
	if d.summary.Count == 1 {
		label = "entry"
	}
	return fmt.Sprintf("%s  (%d %s)", d.summary.Date.Format("2006-01-02"), d.summary.Count, label)
}

func (d dateItem) Description() string { return d.summary.Preview }
func (d dateItem) FilterValue() string { return d.summary.Date.Format("2006-01-02") }

// entryItem implements list.Item for entry.Entry.
type entryItem struct {
	entry entry.Entry
}

func (e entryItem) Title() string {
	return fmt.Sprintf("%s  %s", e.entry.ID, e.entry.CreatedAt.Local().Format("15:04"))
}

func (e entryItem) Description() string { return e.entry.Preview(80) }
func (e entryItem) FilterValue() string { return e.entry.ID }

// contextItem implements list.Item for storage.Context.
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

// templateItem implements list.Item for storage.Template.
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
	lines := strings.SplitN(t.tmpl.Content, "\n", 2)
	preview := lines[0]
	if len(preview) > 60 {
		preview = preview[:57] + "..."
	}
	return preview
}

func (t templateItem) FilterValue() string { return t.tmpl.Name }

// templateCallbackFunc is called after template picker selection.
// names is nil if cancelled, empty slice if skipped, or selected template names.
type templateCallbackFunc func(m *pickerModel, names []string) tea.Cmd

// pickerModel is the main Bubble Tea model for the daily picker.
type pickerModel struct {
	store  StorageProvider
	cfg    TUIConfig
	screen pickerScreen
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
	// Jot mode
	jotInput  textarea.Model
	jotActive bool
	jotTarget *entry.Entry // entry to append jot to (nil = create new daily)
	// Delete confirmation mode
	deleteActive bool
	deleteEntry  entry.Entry
	// Context panel
	contextList     list.Model
	contextEntryID  string // entry being context-managed (empty = browse mode)
	contextItems    []storage.Context
	contextAttached map[string]bool // contextID -> attached to current entry
	prevScreen      pickerScreen    // screen to return to on esc
	contextInput    textinput.Model
	contextCreating bool
	// Help overlay
	helpActive bool
	// Template picker
	templateList         list.Model
	templatePickerActive bool
	templateSelected     map[string]bool    // name -> selected
	templateItems        []storage.Template // cached templates
	templateCallback     templateCallbackFunc
	templateTargetEntry  *entry.Entry // entry being edited with template append
	// Common
	width  int
	height int
	ready  bool
	err    error
}

func newPickerModel(store StorageProvider, days []storage.DaySummary, theme Theme) pickerModel {
	// Build date list items
	items := make([]list.Item, len(days))
	for i, d := range days {
		items[i] = dateItem{summary: d}
	}

	dateList := theme.NewList(items, 0, 0)
	dateList.Title = "Daily View"
	dateList.SetShowHelp(false)

	return pickerModel{
		store:    store,
		cfg:     TUIConfig{Theme: theme},
		screen:   screenDateList,
		days:     days,
		dateList: dateList,
	}
}

func (m pickerModel) Init() tea.Cmd {
	if m.screen == screenToday {
		return m.loadTodayCmd
	}
	return nil
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case jotCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		// Refresh current screen
		switch m.screen {
		case screenToday:
			return m, m.loadTodayCmd
		case screenDateList:
			// Reload date list by reinitializing on current screen
			return m.loadDateList()
		case screenDayDetail:
			// Reload day detail by reinitializing on current screen
			return m.loadDayDetail()
		default:
			return m, nil
		}

	case editorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		// Refresh current screen
		switch m.screen {
		case screenToday:
			return m, m.loadTodayCmd
		case screenDateList:
			return m.loadDateList()
		case screenDayDetail:
			return m.loadDayDetail()
		default:
			return m, nil
		}

	case deleteCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		// Go back to parent screen and refresh
		if m.screen == screenEntryDetail {
			m.screen = screenDayDetail
		}
		// Refresh current screen
		switch m.screen {
		case screenToday:
			return m, m.loadTodayCmd
		case screenDateList:
			return m.loadDateList()
		case screenDayDetail:
			return m.loadDayDetail()
		default:
			return m, nil
		}

	case contextCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		// Context created successfully, reload the context list
		return m, m.loadContexts

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
		m.contextList = m.cfg.Theme.NewList(items, m.contentWidth()-4, m.height-6)
		m.contextList.Title = title
		m.contextList.SetShowHelp(false)
		m.screen = screenContextPanel
		return m, nil

	case templatesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		if len(msg.templates) == 0 {
			// No templates available, proceed without
			if m.templateCallback != nil {
				cmd := m.templateCallback(&m, nil)
				m.templateCallback = nil
				return m, cmd
			}
			return m, nil
		}
		m.templateItems = msg.templates
		m.templateSelected = make(map[string]bool)

		items := make([]list.Item, len(msg.templates))
		for i, t := range msg.templates {
			items[i] = templateItem{tmpl: t, selected: false}
		}

		m.templateList = m.cfg.Theme.NewList(items, m.contentWidth()-4, m.height/2)
		m.templateList.Title = "Select Template(s)"
		m.templateList.SetShowHelp(false)
		m.templatePickerActive = true
		return m, nil

	case openEditorForCreateMsg:
		return m.doCreateWithEditor(msg.content, msg.refs)

	case openEditorForEditMsg:
		return m.doEditWithEditor(msg.entryID, msg.content, msg.refs)

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
		m.todayList = m.cfg.Theme.NewList(items, 0, 0)
		m.todayList.Title = ""
		m.todayList.SetShowHelp(false)
		// Build daily viewport
		if msg.daily != nil {
			m.dailyViewport = viewport.New(m.contentWidth(), m.dailyViewportHeight())
			// Render markdown content as rich text
			rendered := RenderMarkdownWithStyle(msg.daily.Content, m.contentWidth(), m.cfg.Theme.MarkdownStyle)
			m.dailyViewport.SetContent(rendered)
		}
		if m.ready {
			m.layoutToday()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		headerHeight := 4 // title + spacing
		footerHeight := 2 // help hint + spacing

		switch m.screen {
		case screenToday:
			m.layoutToday()
		case screenDateList:
			m.dateList.SetSize(m.contentWidth(), msg.Height-footerHeight)
		case screenDayDetail:
			m.dayList.SetSize(m.contentWidth(), msg.Height-footerHeight)
		case screenEntryDetail:
			m.viewport.Width = m.contentWidth()
			m.viewport.Height = msg.Height - headerHeight - footerHeight
			m.viewport.SetContent(m.formatEntry())
		}
		return m, nil

	case tea.KeyMsg:
		// Help overlay — intercept all keys when active
		if m.helpActive {
			switch msg.String() {
			case "?", "esc":
				m.helpActive = false
				return m, nil
			}
			// Swallow all other keys while help is shown
			return m, nil
		}

		// Template picker mode — intercept all keys
		if m.templatePickerActive {
			return m.updateTemplatePicker(msg)
		}

		// Jot input mode — intercept all keys
		if m.jotActive {
			return m.updateJotInput(msg)
		}

		// Delete confirmation mode — intercept all keys
		if m.deleteActive {
			return m.updateDeleteConfirm(msg)
		}

		// Global keys (work from any screen when not in input mode)
		switch msg.String() {
		case "j":
			return m.startJot()
		case "c":
			return m.startCreate()
		case "?":
			m.helpActive = true
			return m, nil
		case "x":
			return m.openContextPanel()
		}

		// Screen-specific handling
		switch m.screen {
		case screenToday:
			return m.updateToday(msg)
		case screenDateList:
			return m.updateDateList(msg)
		case screenDayDetail:
			return m.updateDayDetail(msg)
		case screenEntryDetail:
			return m.updateEntryDetail(msg)
		case screenContextPanel:
			return m.updateContextPanel(msg)
		}
	}

	// Pass through to active sub-model
	var cmd tea.Cmd
	switch m.screen {
	case screenDateList:
		m.dateList, cmd = m.dateList.Update(msg)
	case screenDayDetail:
		m.dayList, cmd = m.dayList.Update(msg)
	case screenEntryDetail:
		m.viewport, cmd = m.viewport.Update(msg)
	}
	return m, cmd
}

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
		if m.todayFocus == focusDailyViewport && m.dailyEntry != nil {
			// Edit daily entry in $EDITOR
			return m.startEdit(*m.dailyEntry)
		}
		if m.todayFocus == focusEntryList {
			if item, ok := m.todayList.SelectedItem().(entryItem); ok {
				return m.loadEntryDetail(item.entry.ID)
			}
		}
		return m, nil
	case "t", "T":
		// Append template to selected entry
		var targetEntry *entry.Entry
		if m.todayFocus == focusDailyViewport && m.dailyEntry != nil {
			targetEntry = m.dailyEntry
		} else if m.todayFocus == focusEntryList {
			if item, ok := m.todayList.SelectedItem().(entryItem); ok {
				e := item.entry
				targetEntry = &e
			}
		}
		if targetEntry != nil {
			m.templateTargetEntry = targetEntry
			return m.openTemplatePicker(appendTemplatesCallback)
		}
		return m, nil
	}

	// Pass to focused component
	var cmd tea.Cmd
	if m.todayFocus == focusDailyViewport && m.dailyEntry != nil {
		m.dailyViewport, cmd = m.dailyViewport.Update(msg)
	} else if len(m.todayEntries) > 0 {
		m.todayList, cmd = m.todayList.Update(msg)
	}
	return m, cmd
}

func (m pickerModel) updateDateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace":
		m.screen = screenToday
		return m, m.loadTodayCmd
	case "enter":
		if item, ok := m.dateList.SelectedItem().(dateItem); ok {
			// Find index of this day
			for i, d := range m.days {
				if d.Date.Equal(item.summary.Date) {
					m.dayIdx = i
					break
				}
			}
			return m.loadDayDetail()
		}
	}

	var cmd tea.Cmd
	m.dateList, cmd = m.dateList.Update(msg)
	return m, cmd
}

func (m pickerModel) updateDayDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace":
		m.screen = screenDateList
		if m.ready {
			m.dateList.SetSize(m.contentWidth(), m.height-2)
		}
		return m, nil
	case "enter":
		if item, ok := m.dayList.SelectedItem().(entryItem); ok {
			return m.loadEntryDetail(item.entry.ID)
		}
	case "e":
		if item, ok := m.dayList.SelectedItem().(entryItem); ok {
			return m.startEdit(item.entry)
		}
	case "d":
		if item, ok := m.dayList.SelectedItem().(entryItem); ok {
			m.deleteActive = true
			m.deleteEntry = item.entry
			return m, nil
		}
	case "t", "T":
		if item, ok := m.dayList.SelectedItem().(entryItem); ok {
			e := item.entry
			m.templateTargetEntry = &e
			return m.openTemplatePicker(appendTemplatesCallback)
		}
	case "left", "p":
		// Navigate to previous (earlier) day
		if m.dayIdx < len(m.days)-1 {
			m.dayIdx++
			return m.loadDayDetail()
		}
		return m, nil
	case "right", "n":
		// Navigate to next (later) day
		if m.dayIdx > 0 {
			m.dayIdx--
			return m.loadDayDetail()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.dayList, cmd = m.dayList.Update(msg)
	return m, cmd
}

func (m pickerModel) updateEntryDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace":
		m.screen = screenDayDetail
		if m.ready {
			m.dayList.SetSize(m.contentWidth(), m.height-2)
		}
		return m, nil
	case "e":
		return m.startEdit(m.entry)
	case "d":
		m.deleteActive = true
		m.deleteEntry = m.entry
		return m, nil
	case "t", "T":
		m.templateTargetEntry = &m.entry
		return m.openTemplatePicker(appendTemplatesCallback)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m pickerModel) loadDayDetail() (tea.Model, tea.Cmd) {
	day := m.days[m.dayIdx]
	date := day.Date
	entries, err := m.store.List(storage.ListOptions{Date: &date})
	if err != nil {
		m.err = err
		return m, tea.Quit
	}

	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = entryItem{entry: e}
	}

	label := "entries"
	if day.Count == 1 {
		label = "entry"
	}
	m.dayList = m.cfg.Theme.NewList(items, 0, 0)
	m.dayList.Title = fmt.Sprintf("%s (%d %s)", date.Format("2006-01-02"), day.Count, label)
	m.dayList.SetShowHelp(false)
	if m.ready {
		m.dayList.SetSize(m.contentWidth(), m.height-2)
	}
	m.screen = screenDayDetail
	return m, nil
}

func (m pickerModel) loadEntryDetail(id string) (tea.Model, tea.Cmd) {
	e, err := m.store.Get(id)
	if err != nil {
		m.err = err
		return m, tea.Quit
	}

	m.entry = e
	headerHeight := 4
	footerHeight := 2
	vpHeight := max(m.height-headerHeight-footerHeight, 1)
	m.viewport = viewport.New(m.contentWidth(), vpHeight)
	m.viewport.SetContent(m.formatEntry())
	m.screen = screenEntryDetail
	return m, nil
}

func (m pickerModel) formatEntry() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Entry: %s\n", m.entry.ID)
	fmt.Fprintf(&b, "Created: %s\n", m.entry.CreatedAt.Local().Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "Modified: %s\n", m.entry.UpdatedAt.Local().Format("2006-01-02 15:04"))
	fmt.Fprintln(&b)

	// Render markdown content as rich text
	rendered := RenderMarkdownWithStyle(m.entry.Content, m.viewport.Width, m.cfg.Theme.MarkdownStyle)
	fmt.Fprintln(&b, rendered)
	return b.String()
}

func (m pickerModel) dailyViewportHeight() int {
	maxHeight := m.height * 6 / 10 // 60% of terminal
	return maxHeight
}

func (m *pickerModel) layoutToday() {
	// Guard: only layout if we have loaded today's data
	if m.todayList.Items() == nil {
		return
	}

	headerHeight := 2 // header + blank line
	footerHeight := 2 // help + blank line

	if m.dailyEntry != nil {
		vpHeight := m.dailyViewportHeight()
		m.dailyViewport.Width = m.contentWidth()
		m.dailyViewport.Height = vpHeight
		// Render markdown content as rich text
		rendered := RenderMarkdownWithStyle(m.dailyEntry.Content, m.contentWidth(), m.cfg.Theme.MarkdownStyle)
		m.dailyViewport.SetContent(rendered)

		listHeight := max(m.height-headerHeight-vpHeight-footerHeight-1, 3) // 1 for separator
		m.todayList.SetSize(m.contentWidth(), listHeight)
	} else {
		m.todayList.SetSize(m.contentWidth(), m.height-headerHeight-footerHeight)
	}
}

// contentWidth returns the effective content width, respecting MaxWidth configuration.
func (m *pickerModel) contentWidth() int {
	if m.cfg.MaxWidth > 0 && m.width > m.cfg.MaxWidth {
		return m.cfg.MaxWidth
	}
	return m.width
}


type todayLoadedMsg struct {
	daily   *entry.Entry
	entries []entry.Entry
	err     error
}

type jotCompleteMsg struct {
	err error
}

type editorFinishedMsg struct {
	err error
}

type deleteCompleteMsg struct {
	err error
}

type contextsLoadedMsg struct {
	contexts []storage.Context
	attached map[string]bool
	err      error
}

type contextCreatedMsg struct {
	err error
}

type templatesLoadedMsg struct {
	templates []storage.Template
	err       error
}

type openEditorForCreateMsg struct {
	content string
	refs    []entry.TemplateRef
}

type openEditorForEditMsg struct {
	entryID string
	content string
	refs    []entry.TemplateRef
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
	m.dateList = m.cfg.Theme.NewList(items, 0, 0)
	m.dateList.Title = "Daily View"
	m.dateList.SetShowHelp(false)
	if m.ready {
		m.dateList.SetSize(m.contentWidth(), m.height-2)
	}
	m.screen = screenDateList
	return m, nil
}

func (m pickerModel) View() string {
	if !m.ready {
		// No PaintScreen here: dimensions are unknown until the first WindowSizeMsg.
		return "Loading..."
	}

	// Help and template overlays handle their own positioning via lipgloss.Place
	// with WithWhitespaceBackground. ClearLineEnds adds \x1b[K to each line to
	// guarantee the background fills to the right terminal edge.
	if m.helpActive {
		return m.cfg.Theme.ClearLineEnds(m.helpOverlay())
	}
	if m.templatePickerActive {
		picker := m.templatePickerView()
		placed := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, picker,
			lipgloss.WithWhitespaceBackground(m.cfg.Theme.Background))
		return m.cfg.Theme.ClearLineEnds(placed)
	}

	cw := m.contentWidth()
	var result string

	switch m.screen {
	case screenToday:
		if m.dailyEntry == nil && len(m.todayEntries) == 0 {
			// Empty state
			header := m.cfg.Theme.HeaderStyle().Width(cw).Render(
				fmt.Sprintf("Today — %s", time.Now().Format("2006-01-02")))
			empty := m.cfg.Theme.ViewPaneStyle().Width(cw).Render(
				"Nothing yet today.\n\n  j  jot a quick note\n  c  create a new entry")
			footer := m.cfg.Theme.HelpStyle().Width(cw).Render("j jot  c create  b browse  x ctx  ? help")
			result = header + "\n" + empty + "\n" + footer
		} else {
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
			header := m.cfg.Theme.HeaderStyle().Width(cw).Render(
				fmt.Sprintf("Today — %s    %d %s", time.Now().Format("2006-01-02"), count, label))
			sections = append(sections, header)

			// Daily entry viewport
			if m.dailyEntry != nil {
				paneStyle := m.cfg.Theme.ViewPaneStyle().Width(cw)
				sections = append(sections, paneStyle.Render(m.dailyViewport.View()))
			}

			// Other entries list
			if len(m.todayEntries) > 0 {
				sections = append(sections, m.todayList.View())
			}

			// Footer
			footer := m.cfg.Theme.HelpStyle().Width(cw).Render("j jot  c create  e edit  b browse  x ctx  ? help")
			sections = append(sections, footer)

			result = strings.Join(sections, "\n")
		}
	case screenDateList:
		footer := m.cfg.Theme.HelpStyle().Width(cw).Render("↑/↓ navigate • enter select • q quit")
		result = m.dateList.View() + "\n" + footer
	case screenDayDetail:
		footer := m.cfg.Theme.HelpStyle().Width(cw).Render("↑/↓ navigate • enter select • ←/p prev day • →/n next day • esc back • q quit")
		result = m.dayList.View() + "\n" + footer
	case screenEntryDetail:
		header := m.cfg.Theme.HeaderStyle().Width(cw).Render(fmt.Sprintf("Entry: %s", m.entry.ID))
		meta := m.cfg.Theme.HelpStyle().Width(cw).Render(fmt.Sprintf("Created: %s  Modified: %s",
			m.entry.CreatedAt.Local().Format("2006-01-02 15:04"),
			m.entry.UpdatedAt.Local().Format("2006-01-02 15:04")))
		footer := m.cfg.Theme.HelpStyle().Width(cw).Render("↑/↓ scroll • esc back • q quit")
		paneStyle := m.cfg.Theme.ViewPaneStyle().Width(cw)
		result = header + "\n" + meta + "\n\n" + paneStyle.Render(m.viewport.View()) + "\n" + footer
	case screenContextPanel:
		var b strings.Builder
		b.WriteString(m.contextList.View())
		if m.contextCreating {
			b.WriteString("\n" + m.contextInput.View())
		} else {
			hint := "enter toggle  n new  / filter  esc close"
			if m.contextEntryID == "" {
				hint = "n new  / filter  esc close"
			}
			b.WriteString("\n" + m.cfg.Theme.HelpStyle().Width(cw).Render(hint))
		}
		result = b.String()
	}

	if m.deleteActive {
		prompt := fmt.Sprintf("Delete entry %s? [y/N] ", m.deleteEntry.ID)
		result = result + "\n" + m.cfg.Theme.DangerStyle().Width(cw).Render(prompt)
	} else if m.jotActive {
		result = result + "\n" + m.jotInput.View()
	}

	return m.cfg.Theme.PaintScreen(result, m.width, m.height, cw)
}

func (m pickerModel) helpOverlay() string {
	help := m.cfg.Theme.BorderStyle().
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
  j          jot a note (^J for newline)
  c          create new entry (with templates)
  e          edit selected entry
  t          append template to entry
  d          delete selected entry
  /          search / filter
  x          context panel

  q          quit     ? close help`)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, help,
		lipgloss.WithWhitespaceBackground(m.cfg.Theme.Background))
}

func (m pickerModel) startJot() (tea.Model, tea.Cmd) {
	m.jotTarget = m.resolveJotTarget()

	ta := textarea.New()
	ta.Placeholder = "jot (^J=newline ↵=submit)..."
	ta.Focus()
	ta.CharLimit = maxJotInputLength
	ta.SetWidth(m.contentWidth() - 4)
	// Dynamic height: use 25% of screen height or max 5 lines
	height := max(min(m.height/4, 5), 3)
	ta.SetHeight(height)
	ta.ShowLineNumbers = false
	m.jotInput = ta
	m.jotActive = true
	return m, textarea.Blink
}

// resolveJotTarget determines which entry to jot into based on the current screen
// and selection state. Returns nil if no target exists (will create new daily entry).
func (m pickerModel) resolveJotTarget() *entry.Entry {
	switch m.screen {
	case screenToday:
		if m.todayFocus == focusEntryList {
			if item, ok := m.todayList.SelectedItem().(entryItem); ok {
				e := item.entry
				return &e
			}
		}
		return m.dailyEntry
	case screenDayDetail:
		if item, ok := m.dayList.SelectedItem().(entryItem); ok {
			e := item.entry
			return &e
		}
		return nil
	case screenEntryDetail:
		return &m.entry
	default:
		return nil
	}
}

func (m pickerModel) updateJotInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+j":
		// Insert a newline
		m.jotInput.InsertString("\n")
		return m, nil
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

func (m pickerModel) openContextPanel() (tea.Model, tea.Cmd) {
	m.prevScreen = m.screen

	// Determine if we have a selected entry
	switch m.screen {
	case screenToday:
		if m.todayFocus == focusDailyViewport && m.dailyEntry != nil {
			m.contextEntryID = m.dailyEntry.ID
		} else if m.todayFocus == focusEntryList {
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

func (m pickerModel) updateContextPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.contextCreating {
		return m.updateContextCreate(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
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
	case "enter":
		if m.contextEntryID != "" {
			// Toggle context attachment
			if item, ok := m.contextList.SelectedItem().(contextItem); ok {
				return m, func() tea.Msg {
					if item.attached {
						err := m.store.DetachContext(m.contextEntryID, item.ctx.ID)
						if err != nil {
							return contextsLoadedMsg{err: err}
						}
					} else {
						err := m.store.AttachContext(m.contextEntryID, item.ctx.ID)
						if err != nil {
							return contextsLoadedMsg{err: err}
						}
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
		ti.CharLimit = maxContextNameLength
		ti.Width = m.contentWidth() - 8
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
				return contextCreatedMsg{err: err}
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
				return contextCreatedMsg{err: err}
			}

			// Auto-attach if we have an entry selected
			var attachErr error
			if m.contextEntryID != "" {
				attachErr = m.store.AttachContext(m.contextEntryID, id)
				if attachErr != nil {
					// Context was created but attach failed
					return contextCreatedMsg{err: fmt.Errorf("context created but failed to attach: %w", attachErr)}
				}
			}

			return contextCreatedMsg{}
		}
	case "esc":
		m.contextCreating = false
		return m, nil
	}

	var cmd tea.Cmd
	m.contextInput, cmd = m.contextInput.Update(msg)
	return m, cmd
}

// --- Template Picker Methods ---

func (m pickerModel) loadTemplatesCmd() tea.Msg {
	templates, err := m.store.ListTemplates()
	if err != nil {
		return templatesLoadedMsg{err: err}
	}
	return templatesLoadedMsg{templates: templates}
}

func (m pickerModel) openTemplatePicker(callback templateCallbackFunc) (tea.Model, tea.Cmd) {
	m.templateCallback = callback
	return m, m.loadTemplatesCmd
}

func (m pickerModel) updateTemplatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.templatePickerActive = false
		if m.templateCallback != nil {
			cmd := m.templateCallback(&m, nil)
			m.templateCallback = nil
			return m, cmd
		}
		return m, nil

	case " ":
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
		// Collect selected names in order
		var names []string
		for _, t := range m.templateItems {
			if m.templateSelected[t.Name] {
				names = append(names, t.Name)
			}
		}
		if m.templateCallback != nil {
			cmd := m.templateCallback(&m, names)
			m.templateCallback = nil
			return m, cmd
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.templateList, cmd = m.templateList.Update(msg)
	return m, cmd
}

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
	b.WriteString(m.cfg.Theme.HelpStyle().Render("space toggle  enter confirm  esc skip"))

	return m.cfg.Theme.BorderStyle().
		Padding(1).
		Render(b.String())
}

// createWithTemplatesCallback is called after template selection for create action.
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

		return openEditorForCreateMsg{content: content, refs: refs}
	}
}

// appendTemplatesCallback is called after template selection for append action.
func appendTemplatesCallback(m *pickerModel, names []string) tea.Cmd {
	if len(names) == 0 || m.templateTargetEntry == nil {
		return nil
	}

	return func() tea.Msg {
		c, refs, err := template.Compose(m.store, names)
		if err != nil {
			return editorFinishedMsg{err: err}
		}

		// Append to existing content
		newContent := m.templateTargetEntry.Content
		if newContent != "" && !strings.HasSuffix(newContent, "\n") {
			newContent += "\n"
		}
		newContent += "\n" + c

		// Merge refs (deduplicate by ID)
		existingRefs := make(map[string]bool)
		for _, r := range m.templateTargetEntry.Templates {
			existingRefs[r.TemplateID] = true
		}
		mergedRefs := append([]entry.TemplateRef{}, m.templateTargetEntry.Templates...)
		for _, r := range refs {
			if !existingRefs[r.TemplateID] {
				mergedRefs = append(mergedRefs, r)
			}
		}

		return openEditorForEditMsg{
			entryID: m.templateTargetEntry.ID,
			content: newContent,
			refs:    mergedRefs,
		}
	}
}

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

	if initialContent != "" {
		if _, err := tmpFile.WriteString(initialContent); err != nil {
			tmpFile.Close()
			os.Remove(tmpName)
			m.err = fmt.Errorf("failed to write to temp file: %w", err)
			return m, tea.Quit
		}
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpName)
		m.err = fmt.Errorf("failed to prepare temp file: %w", err)
		return m, tea.Quit
	}

	cmdArgs := append(parts[1:], tmpName)
	c := exec.Command(parts[0], cmdArgs...)
	templateRefs := refs

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

func (m pickerModel) doEditWithEditor(entryID string, content string, refs []entry.TemplateRef) (tea.Model, tea.Cmd) {
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

	if _, err := tmpFile.WriteString(content); err != nil {
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

	cmdArgs := append(parts[1:], tmpName)
	c := exec.Command(parts[0], cmdArgs...)
	originalContent := content
	templateRefs := refs

	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpName)
		if err != nil {
			return editorFinishedMsg{err: err}
		}
		data, err := os.ReadFile(tmpName)
		if err != nil {
			return editorFinishedMsg{err: err}
		}
		newContent := strings.TrimSpace(string(data))
		if newContent == "" || newContent == strings.TrimSpace(originalContent) {
			return editorFinishedMsg{} // no change
		}
		if _, err := m.store.Update(entryID, newContent, templateRefs); err != nil {
			return editorFinishedMsg{err: err}
		}
		return editorFinishedMsg{}
	})
}

func (m pickerModel) doJot(content string) tea.Msg {
	now := time.Now()
	timestamp := now.Format("15:04")
	jotLine := fmt.Sprintf("- **%s** %s", timestamp, content)

	if m.jotTarget != nil {
		// Append to the targeted entry
		target, err := m.store.Get(m.jotTarget.ID)
		if err != nil {
			return jotCompleteMsg{err: fmt.Errorf("jot target not found: %w", err)}
		}

		var newContent string
		if strings.TrimSpace(target.Content) == "" {
			newContent = jotLine
		} else {
			newContent = target.Content + "\n" + jotLine
		}
		_, err = m.store.Update(target.ID, newContent, nil)
		if err != nil {
			return jotCompleteMsg{err: err}
		}
	} else {
		// No target — create new daily entry (screenToday with no entries)
		id, err := entry.NewID()
		if err != nil {
			return jotCompleteMsg{err: err}
		}
		nowUTC := now.UTC()

		// Use default template if configured
		var templateRefs []entry.TemplateRef
		if m.cfg.DefaultTemplate != "" {
			names := template.ParseNames(m.cfg.DefaultTemplate)
			_, refs, err := template.Compose(m.store, names)
			if err != nil {
				// Continue without template refs (match CLI behavior)
			} else {
				templateRefs = refs
			}
		}

		e := entry.Entry{
			ID:        id,
			Content:   fmt.Sprintf("# %s\n\n%s", now.Format("2006-01-02"), jotLine),
			Templates: templateRefs,
			CreatedAt: nowUTC,
			UpdatedAt: nowUTC,
		}
		if err := m.store.Create(e); err != nil {
			return jotCompleteMsg{err: err}
		}
	}

	return jotCompleteMsg{}
}

func (m pickerModel) startCreate() (tea.Model, tea.Cmd) {
	// Open template picker for create flow
	return m.openTemplatePicker(createWithTemplatesCallback)
}

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

// TUIConfig holds configuration needed by the TUI.
type TUIConfig struct {
	Editor          string // resolved editor command
	DefaultTemplate string // default template name
	MaxWidth        int    // maximum viewport width (0 = no limit)
	Theme           Theme  // resolved theme
}

// newTUIModel creates a new TUI model starting at the today screen.
func newTUIModel(store StorageProvider, cfg TUIConfig) pickerModel {
	m := pickerModel{
		store:  store,
		cfg:    cfg,
		screen: screenToday,
	}
	return m
}

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

// RunPicker launches the interactive daily picker.
func RunPicker(store StorageProvider, opts storage.ListDaysOptions, theme Theme) error {
	days, err := store.ListDays(opts)
	if err != nil {
		return err
	}

	if len(days) == 0 {
		fmt.Println("No diary entries found.")
		return nil
	}

	m := newPickerModel(store, days, theme)
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
