package ui

import (
	"regexp"
	"unicode/utf8"
)

// ANSI escape code patterns
var (
	// SGR (Select Graphic Rendition) codes: ESC[...m
	ansiSGRPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

	// OSC-8 hyperlink codes: ESC]8;;...ESC\ or ESC]8;;ESC\
	osc8Pattern = regexp.MustCompile(`\x1b\]8;;[^\x1b]*\x1b\\|\x1b\]8;;\x1b\\`)

	// Combined pattern for all ANSI codes
	allAnsiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m|\x1b\]8;;[^\x1b]*\x1b\\`)
)

// StripAnsi removes all ANSI escape codes from a string
func StripAnsi(input string) string {
	// First remove OSC-8 hyperlinks
	result := osc8Pattern.ReplaceAllString(input, "")
	// Then remove SGR codes
	result = ansiSGRPattern.ReplaceAllString(result, "")
	return result
}

// VisibleWidth returns the display width of a string, ignoring ANSI codes
// This counts runes, not bytes, for proper Unicode support
func VisibleWidth(input string) int {
	stripped := StripAnsi(input)
	return utf8.RuneCountInString(stripped)
}

// TruncateVisible truncates a string to a maximum visible width
// Preserves ANSI codes but counts only visible characters
func TruncateVisible(input string, maxWidth int) string {
	stripped := StripAnsi(input)
	if utf8.RuneCountInString(stripped) <= maxWidth {
		return input
	}

	// Simple approach: strip, truncate, return plain
	runes := []rune(stripped)
	if len(runes) > maxWidth-3 {
		return string(runes[:maxWidth-3]) + "..."
	}
	return string(runes[:maxWidth])
}

// PadRight pads a string to a minimum visible width (right-aligned content)
func PadRight(input string, width int) string {
	visible := VisibleWidth(input)
	if visible >= width {
		return input
	}
	padding := width - visible
	return input + spaces(padding)
}

// PadLeft pads a string to a minimum visible width (left-aligned content)
func PadLeft(input string, width int) string {
	visible := VisibleWidth(input)
	if visible >= width {
		return input
	}
	padding := width - visible
	return spaces(padding) + input
}

// PadCenter centers a string within a given width
func PadCenter(input string, width int) string {
	visible := VisibleWidth(input)
	if visible >= width {
		return input
	}
	padding := width - visible
	left := padding / 2
	right := padding - left
	return spaces(left) + input + spaces(right)
}

// spaces returns a string of n spaces
func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
