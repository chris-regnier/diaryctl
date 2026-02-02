package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// pickerScreen represents the current screen state.
type pickerScreen int

const (
	screenDateList pickerScreen = iota
	screenDayDetail
	screenEntryDetail
)

// StorageProvider abstracts storage operations for the picker.
type StorageProvider interface {
	ListDays(opts storage.ListDaysOptions) ([]storage.DaySummary, error)
	List(opts storage.ListOptions) ([]entry.Entry, error)
	Get(id string) (entry.Entry, error)
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

// pickerModel is the main Bubble Tea model for the daily picker.
type pickerModel struct {
	store    StorageProvider
	screen   pickerScreen
	days     []storage.DaySummary
	dayIdx   int
	dateList list.Model
	dayList  list.Model
	viewport viewport.Model
	entry    entry.Entry
	width    int
	height   int
	ready    bool
	err      error
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
	return nil
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		headerHeight := 4 // title + spacing
		footerHeight := 2 // help hint + spacing

		switch m.screen {
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
		switch m.screen {
		case screenDateList:
			return m.updateDateList(msg)
		case screenDayDetail:
			return m.updateDayDetail(msg)
		case screenEntryDetail:
			return m.updateEntryDetail(msg)
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

func (m pickerModel) updateDateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
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

func (m pickerModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	switch m.screen {
	case screenDateList:
		footer := helpStyle.Render("↑/↓ navigate • enter select • q quit")
		return m.dateList.View() + "\n" + footer
	case screenDayDetail:
		footer := helpStyle.Render("↑/↓ navigate • enter select • ←/p prev day • →/n next day • esc back • q quit")
		return m.dayList.View() + "\n" + footer
	case screenEntryDetail:
		header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Entry: %s", m.entry.ID))
		meta := helpStyle.Render(fmt.Sprintf("Created: %s  Modified: %s",
			m.entry.CreatedAt.Local().Format("2006-01-02 15:04"),
			m.entry.UpdatedAt.Local().Format("2006-01-02 15:04")))
		footer := helpStyle.Render("↑/↓ scroll • esc back • q quit")
		return header + "\n" + meta + "\n\n" + m.viewport.View() + "\n" + footer
	}
	return ""
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
