package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
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
	jotInput  textinput.Model
	jotActive bool
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
	// Common
	width  int
	height int
	ready  bool
	err    error
}

func newPickerModel(store StorageProvider, days []storage.DaySummary) pickerModel {
	// Build date list items
	items := make([]list.Item, len(days))
	for i, d := range days {
		items[i] = dateItem{summary: d}
	}

	dateList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	dateList.Title = "Daily View"
	dateList.SetShowHelp(false)

	return pickerModel{
		store:    store,
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
			m.dateList.SetSize(msg.Width, msg.Height-footerHeight)
		case screenDayDetail:
			m.dayList.SetSize(msg.Width, msg.Height-footerHeight)
		case screenEntryDetail:
			m.viewport.Width = msg.Width
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
		if m.todayFocus == 0 && m.dailyEntry != nil {
			// Edit daily entry in $EDITOR
			return m.startEdit(*m.dailyEntry)
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
			m.dateList.SetSize(m.width, m.height-2)
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
			m.dayList.SetSize(m.width, m.height-2)
		}
		return m, nil
	case "e":
		return m.startEdit(m.entry)
	case "d":
		m.deleteActive = true
		m.deleteEntry = m.entry
		return m, nil
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
	m.dayList = list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.dayList.Title = fmt.Sprintf("%s (%d %s)", date.Format("2006-01-02"), day.Count, label)
	m.dayList.SetShowHelp(false)
	if m.ready {
		m.dayList.SetSize(m.width, m.height-2)
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
	vpHeight := m.height - headerHeight - footerHeight
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport = viewport.New(m.width, vpHeight)
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
	fmt.Fprintln(&b, m.entry.Content)
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
	m.dateList = list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.dateList.Title = "Daily View"
	m.dateList.SetShowHelp(false)
	if m.ready {
		m.dateList.SetSize(m.width, m.height-2)
	}
	m.screen = screenDateList
	return m, nil
}

func (m pickerModel) View() string {
	if !m.ready {
		return "Loading..."
	}
	if m.helpActive {
		return m.helpOverlay()
	}

	var result string

	switch m.screen {
	case screenToday:
		if m.dailyEntry == nil && len(m.todayEntries) == 0 {
			// Empty state
			header := lipgloss.NewStyle().Bold(true).Render(
				fmt.Sprintf("Today — %s", time.Now().Format("2006-01-02")))
			empty := "\nNothing yet today.\n\n  j  jot a quick note\n  c  create a new entry\n"
			footer := helpStyle.Render("j jot  c create  b browse  x ctx  ? help")
			result = header + empty + "\n" + footer
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

			result = strings.Join(sections, "\n")
		}
	case screenDateList:
		footer := helpStyle.Render("↑/↓ navigate • enter select • q quit")
		result = m.dateList.View() + "\n" + footer
	case screenDayDetail:
		footer := helpStyle.Render("↑/↓ navigate • enter select • ←/p prev day • →/n next day • esc back • q quit")
		result = m.dayList.View() + "\n" + footer
	case screenEntryDetail:
		header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Entry: %s", m.entry.ID))
		meta := helpStyle.Render(fmt.Sprintf("Created: %s  Modified: %s",
			m.entry.CreatedAt.Local().Format("2006-01-02 15:04"),
			m.entry.UpdatedAt.Local().Format("2006-01-02 15:04")))
		footer := helpStyle.Render("↑/↓ scroll • esc back • q quit")
		result = header + "\n" + meta + "\n\n" + m.viewport.View() + "\n" + footer
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
			b.WriteString("\n" + helpStyle.Render(hint))
		}
		result = b.String()
	}

	// At the end of View(), before returning:
	if m.deleteActive {
		prompt := fmt.Sprintf("Delete entry %s? [y/N] ", m.deleteEntry.ID)
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // red
		return result + "\n" + warningStyle.Render(prompt)
	}

	if m.jotActive {
		return result + "\n" + m.jotInput.View()
	}

	return result
}

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
		return m, m.loadTodayCmd
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

func (m pickerModel) doJot(content string) tea.Msg {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Get all today's entries (single fetch)
	entries, err := m.store.List(storage.ListOptions{Date: &today})
	if err != nil {
		return jotCompleteMsg{err: err}
	}

	timestamp := now.Format("15:04")
	jotLine := fmt.Sprintf("- **%s** %s", timestamp, content)

	if len(entries) > 0 {
		// Append to existing daily entry (oldest)
		daily := entries[len(entries)-1]
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
		// Create new daily entry with template
		id, err := entry.NewID()
		if err != nil {
			return jotCompleteMsg{err: err}
		}
		nowUTC := now.UTC()

		// Use default template if configured
		var templateRefs []entry.TemplateRef
		if m.cfg.DefaultTemplate != "" {
			templateRefs = []entry.TemplateRef{{TemplateName: m.cfg.DefaultTemplate}}
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

// TUIConfig holds configuration needed by the TUI.
type TUIConfig struct {
	Editor          string // resolved editor command
	DefaultTemplate string // default template name
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
func RunPicker(store StorageProvider, opts storage.ListDaysOptions) error {
	days, err := store.ListDays(opts)
	if err != nil {
		return err
	}

	if len(days) == 0 {
		fmt.Println("No diary entries found.")
		return nil
	}

	m := newPickerModel(store, days)
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
