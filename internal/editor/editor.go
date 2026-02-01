package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ResolveEditor determines which editor to use based on config, env vars, and fallback.
func ResolveEditor(configEditor string) string {
	if configEditor != "" {
		return configEditor
	}
	if ed := os.Getenv("EDITOR"); ed != "" {
		return ed
	}
	if ed := os.Getenv("VISUAL"); ed != "" {
		return ed
	}
	return "vi"
}

// Edit opens the given content in an editor and returns the edited content.
// If the user saves unchanged content or an empty file, it returns the original
// content and changed=false.
func Edit(editorCmd string, initialContent string) (content string, changed bool, err error) {
	// Create temp file
	tmp, err := os.CreateTemp("", "diaryctl-*.md")
	if err != nil {
		return "", false, fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	// Write initial content
	if _, err := tmp.WriteString(initialContent); err != nil {
		tmp.Close()
		return "", false, fmt.Errorf("writing temp file: %w", err)
	}
	tmp.Close()

	// Launch editor
	parts := strings.Fields(editorCmd)
	if len(parts) == 0 {
		return "", false, fmt.Errorf("empty editor command")
	}

	cmdArgs := append(parts[1:], tmpName)
	cmd := exec.Command(parts[0], cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", false, fmt.Errorf("editor exited with error: %w", err)
	}

	// Read result
	data, err := os.ReadFile(tmpName)
	if err != nil {
		return "", false, fmt.Errorf("reading edited file: %w", err)
	}

	result := string(data)

	// Detect if content is empty or unchanged
	if strings.TrimSpace(result) == "" {
		return "", false, nil
	}

	if strings.TrimSpace(result) == strings.TrimSpace(initialContent) {
		return initialContent, false, nil
	}

	return result, true, nil
}
