package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

type recordingModel struct {
	theme     Theme
	elapsed   time.Duration
	recording bool
	cancelled bool
	done      bool
}

func (m recordingModel) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m recordingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "q":
			m.recording = false
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.recording = false
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		}
	case tickMsg:
		m.elapsed += time.Second
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}
	return m, nil
}

func (m recordingModel) View() string {
	if m.done {
		return ""
	}

	// Pulsing dot animation based on elapsed seconds.
	dots := []string{"   ", ".  ", ".. ", "..."}
	dot := dots[(int(m.elapsed.Seconds()))%len(dots)]

	minutes := int(m.elapsed.Minutes())
	seconds := int(m.elapsed.Seconds()) % 60

	recordStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Danger)
	timeStyle := lipgloss.NewStyle().Foreground(m.theme.Primary)
	helpStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)

	return fmt.Sprintf("  %s  %s %s  %s",
		recordStyle.Render(fmt.Sprintf("Recording%s", dot)),
		timeStyle.Render(fmt.Sprintf("%d:%02d", minutes, seconds)),
		"",
		helpStyle.Render("Press Enter to stop, Esc to cancel"),
	)
}

// RecordingResult holds the outcome of a recording session.
type RecordingResult struct {
	Cancelled bool
	Elapsed   time.Duration
}

// RunRecordingIndicator shows an interactive recording indicator in the terminal.
// It returns when the user presses Enter (stop) or Ctrl+C/Esc (cancel).
func RunRecordingIndicator(theme Theme) (RecordingResult, error) {
	m := recordingModel{
		theme:     theme,
		recording: true,
	}
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return RecordingResult{}, err
	}

	rm := result.(recordingModel)
	return RecordingResult{
		Cancelled: rm.cancelled,
		Elapsed:   rm.elapsed,
	}, nil
}
