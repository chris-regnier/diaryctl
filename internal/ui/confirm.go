package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	promptStyle  = lipgloss.NewStyle().Bold(true)
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
)

type confirmModel struct {
	prompt    string
	confirmed bool
	done      bool
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch strings.ToLower(msg.String()) {
		case "y":
			m.confirmed = true
			m.done = true
			return m, tea.Quit
		case "n", "enter", "esc", "ctrl+c":
			m.confirmed = false
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("%s %s",
		promptStyle.Render(m.prompt),
		warningStyle.Render("[y/N]"),
	) + " "
}

// Confirm shows an interactive confirmation prompt and returns true if the user confirms.
func Confirm(prompt string) (bool, error) {
	m := confirmModel{prompt: prompt}
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return false, err
	}
	return result.(confirmModel).confirmed, nil
}
