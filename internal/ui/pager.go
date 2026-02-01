package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

type pagerModel struct {
	viewport viewport.Model
	content  string
	ready    bool
}

func (m pagerModel) Init() tea.Cmd {
	return nil
}

func (m pagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-1)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 1
			m.viewport.SetContent(m.content)
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m pagerModel) View() string {
	if !m.ready {
		return "Loading..."
	}
	footer := helpStyle.Render("↑/↓ scroll • q quit")
	return m.viewport.View() + "\n" + footer
}

// PageOutput displays content through a Bubble Tea pager when running in a TTY
// and the content exceeds terminal height. Otherwise writes directly to stdout.
func PageOutput(content string) error {
	// If not a TTY, write directly
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Print(content)
		return nil
	}

	// Check if content fits in terminal
	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		fmt.Print(content)
		return nil
	}

	lineCount := strings.Count(content, "\n") + 1
	if lineCount <= height-2 {
		fmt.Print(content)
		return nil
	}

	// Use Bubble Tea pager
	p := tea.NewProgram(pagerModel{content: content}, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// OutputOrPage writes content to the writer, using the pager if appropriate.
// When jsonOutput is true, always writes directly (no paging).
func OutputOrPage(w io.Writer, content string, jsonOutput bool) error {
	if jsonOutput {
		fmt.Fprint(w, content)
		return nil
	}
	if w == os.Stdout {
		return PageOutput(content)
	}
	fmt.Fprint(w, content)
	return nil
}
