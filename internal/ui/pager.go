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
	maxWidth int // maximum viewport width (0 = no limit)
	width    int // terminal width
	height   int // terminal height
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
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(m.contentWidth(), msg.Height-1)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = m.contentWidth()
			m.viewport.Height = msg.Height - 1
			m.viewport.SetContent(m.content)
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// contentWidth returns the effective content width, respecting maxWidth configuration.
func (m *pagerModel) contentWidth() int {
	if m.maxWidth > 0 && m.width > m.maxWidth {
		return m.maxWidth
	}
	return m.width
}

// centerContent centers the given content string horizontally if width > maxWidth.
func (m *pagerModel) centerContent(content string) string {
	if m.maxWidth <= 0 || m.width <= m.maxWidth {
		return content
	}

	contentWidth := m.maxWidth
	leftPadding := (m.width - contentWidth) / 2

	if leftPadding <= 0 {
		return content
	}

	padding := strings.Repeat(" ", leftPadding)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = padding + line
	}
	return strings.Join(lines, "\n")
}

func (m pagerModel) View() string {
	if !m.ready {
		return m.centerContent("Loading...")
	}
	footer := helpStyle.Render("↑/↓ scroll • q quit")
	return m.centerContent(m.viewport.View() + "\n" + footer)
}

// PageOutput displays content through a Bubble Tea pager when running in a TTY
// and the content exceeds terminal height. Otherwise writes directly to stdout.
// Uses a default max width of 100 characters.
func PageOutput(content string) error {
	return PageOutputWithMaxWidth(content, 100)
}

// PageOutputWithMaxWidth displays content with a custom max width constraint.
func PageOutputWithMaxWidth(content string, maxWidth int) error {
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
	p := tea.NewProgram(pagerModel{content: content, maxWidth: maxWidth}, tea.WithAltScreen())
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
