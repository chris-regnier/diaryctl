package ui

import (
	"regexp"
)

// stripANSI removes ANSI escape sequences from a string.
// Handles SGR codes (\x1b[...m), erase codes (\x1b[K), and other CSI sequences.
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(s, "")
}
