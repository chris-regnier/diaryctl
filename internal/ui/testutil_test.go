package ui

import (
	"regexp"
)

// stripANSI removes ANSI escape sequences from a string.
// This is a test utility function used to strip color codes for assertion testing.
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}
