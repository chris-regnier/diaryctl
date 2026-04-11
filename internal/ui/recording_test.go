package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRecordingModel_Init(t *testing.T) {
	m := recordingModel{theme: testTheme(), recording: true}
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected Init to return a tick command")
	}
}

func TestRecordingModel_View(t *testing.T) {
	m := recordingModel{theme: testTheme(), recording: true}
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view during recording")
	}
}

func TestRecordingModel_View_Done(t *testing.T) {
	m := recordingModel{theme: testTheme(), recording: false, done: true}
	view := m.View()
	if view != "" {
		t.Errorf("expected empty view when done, got %q", view)
	}
}

func TestRecordingModel_StopOnEnter(t *testing.T) {
	m := recordingModel{theme: testTheme(), recording: true}
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, cmd := m.Update(msg)
	rm := result.(recordingModel)

	if rm.recording {
		t.Error("expected recording to stop")
	}
	if rm.cancelled {
		t.Error("expected not cancelled on Enter")
	}
	if !rm.done {
		t.Error("expected done to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestRecordingModel_CancelOnCtrlC(t *testing.T) {
	m := recordingModel{theme: testTheme(), recording: true}
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	result, _ := m.Update(msg)
	rm := result.(recordingModel)

	if rm.recording {
		t.Error("expected recording to stop")
	}
	if !rm.cancelled {
		t.Error("expected cancelled on Ctrl+C")
	}
}

func TestRecordingModel_CancelOnEsc(t *testing.T) {
	m := recordingModel{theme: testTheme(), recording: true}
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	result, _ := m.Update(msg)
	rm := result.(recordingModel)

	if !rm.cancelled {
		t.Error("expected cancelled on Esc")
	}
}

func TestRecordingModel_Tick(t *testing.T) {
	m := recordingModel{theme: testTheme(), recording: true}
	msg := tickMsg(time.Now())
	result, cmd := m.Update(msg)
	rm := result.(recordingModel)

	if rm.elapsed != time.Second {
		t.Errorf("expected elapsed 1s, got %v", rm.elapsed)
	}
	if cmd == nil {
		t.Error("expected tick command to continue")
	}
}

func TestRecordingModel_View_ContainsRecording(t *testing.T) {
	m := recordingModel{theme: testTheme(), recording: true}
	view := stripANSI(m.View())
	if !strings.Contains(view, "Recording") {
		t.Errorf("expected view to contain 'Recording', got %q", view)
	}
}

func testTheme() Theme {
	return Theme{
		Primary:   "#FFFFFF",
		Secondary: "#888888",
		Accent:    "#00FF00",
		Muted:     "#666666",
		Danger:    "#FF0000",
	}
}
