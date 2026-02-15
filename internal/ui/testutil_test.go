package ui

import (
	"regexp"
	"strings"
)

// stripANSI removes ANSI escape sequences from a string.
// This is a test utility function used to strip color codes for assertion testing.
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

// countLines returns the number of lines in the given string.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}
