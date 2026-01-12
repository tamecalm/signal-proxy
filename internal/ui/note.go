package ui

import (
	"fmt"
	"os"
	"strings"
)

// Note displays a boxed message with optional title
func Note(message string, title string) {
	wrapped := WrapNoteMessage(message, 80)
	lines := strings.Split(wrapped, "\n")

	fmt.Println()

	// Calculate box width
	maxWidth := 0
	for _, line := range lines {
		w := VisibleWidth(line)
		if w > maxWidth {
			maxWidth = w
		}
	}
	boxWidth := maxWidth + 4 // padding

	// Top border with title
	if title != "" {
		styledTitle := title
		if IsRich() {
			styledTitle = Heading(title)
		}
		top := fmt.Sprintf("%s%s %s %s%s",
			Muted(boxTopLeft),
			Muted(strings.Repeat(boxHorizontal, 2)),
			styledTitle,
			Muted(strings.Repeat(boxHorizontal, boxWidth-4-VisibleWidth(title))),
			Muted(boxTopRight))
		fmt.Println(top)
	} else {
		fmt.Println(Muted(boxTopLeft + strings.Repeat(boxHorizontal, boxWidth) + boxTopRight))
	}

	// Content lines
	for _, line := range lines {
		padding := boxWidth - VisibleWidth(line) - 2
		if padding < 0 {
			padding = 0
		}
		fmt.Printf("%s %s%s %s\n",
			Muted(boxVertical),
			line,
			spaces(padding),
			Muted(boxVertical))
	}

	// Bottom border
	fmt.Println(Muted(boxBottomLeft + strings.Repeat(boxHorizontal, boxWidth) + boxBottomRight))
	fmt.Println()
}

// WrapNoteMessage wraps text to fit within terminal width
func WrapNoteMessage(message string, maxWidth int) string {
	// Get terminal width
	columns := 80
	if term, ok := os.LookupEnv("COLUMNS"); ok {
		if n := parseIntOr(term, 80); n > 0 {
			columns = n
		}
	}

	// Cap width
	width := columns - 10
	if width > maxWidth {
		width = maxWidth
	}
	if width < 40 {
		width = 40
	}

	// Wrap each line
	inputLines := strings.Split(message, "\n")
	var outputLines []string

	for _, line := range inputLines {
		wrapped := wrapLine(line, width)
		outputLines = append(outputLines, wrapped...)
	}

	return strings.Join(outputLines, "\n")
}

// wrapLine wraps a single line to width
func wrapLine(line string, maxWidth int) []string {
	if strings.TrimSpace(line) == "" {
		return []string{line}
	}

	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	current := ""

	for _, word := range words {
		candidate := current
		if current != "" {
			candidate += " "
		}
		candidate += word

		if VisibleWidth(candidate) <= maxWidth {
			current = candidate
		} else {
			if current != "" {
				lines = append(lines, current)
			}
			current = word
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

// parseIntOr parses an int or returns default
func parseIntOr(s string, def int) int {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return def
	}
	return n
}

// InfoNote displays an info-styled note
func InfoNote(message string) {
	Note(message, "ℹ Info")
}

// WarningNote displays a warning-styled note
func WarningNote(message string) {
	Note(message, "⚠ Warning")
}

// ErrorNote displays an error-styled note
func ErrorNote(message string) {
	Note(message, "✗ Error")
}

// SuccessNote displays a success-styled note
func SuccessNote(message string) {
	Note(message, "✓ Success")
}
