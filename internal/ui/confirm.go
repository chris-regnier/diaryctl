package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type confirmModel struct {
	prompt    string
	confirmed bool
	done      bool
	theme     Theme
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
	promptStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary)
	warningStyle := m.theme.DangerStyle()
	return fmt.Sprintf("%s %s",
		promptStyle.Render(m.prompt),
		warningStyle.Render("[y/N]"),
	) + " "
}

// Confirm shows an interactive confirmation prompt and returns true if the user confirms.
func Confirm(prompt string, theme Theme) (bool, error) {
	m := confirmModel{prompt: prompt, theme: theme}
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return false, err
	}
	return result.(confirmModel).confirmed, nil
}
