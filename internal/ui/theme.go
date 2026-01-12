package ui

import (
	"os"
	"strings"

	"github.com/fatih/color"
)

// Theme provides styled color functions for consistent CLI output
// Respects NO_COLOR and FORCE_COLOR environment variables

var (
	// Check color support
	noColor    = os.Getenv("NO_COLOR") != ""
	forceColor = isForceColor()
)

func isForceColor() bool {
	fc := strings.TrimSpace(os.Getenv("FORCE_COLOR"))
	return fc != "" && fc != "0"
}

// IsRich returns true if the terminal supports rich output (colors)
func IsRich() bool {
	if noColor && !forceColor {
		return false
	}
	return color.NoColor == false
}

// Theme color functions - wrapping fatih/color for consistency

// Accent returns primary brand-colored text
func Accent(format string, a ...interface{}) string {
	return color.New(color.FgHiRed).Sprintf(format, a...)
}

// AccentBright returns highlighted accent text
func AccentBright(format string, a ...interface{}) string {
	return color.New(color.FgHiRed, color.Bold).Sprintf(format, a...)
}

// AccentDim returns muted accent text
func AccentDim(format string, a ...interface{}) string {
	return color.New(color.FgRed).Sprintf(format, a...)
}

// Info returns informational styled text
func Info(format string, a ...interface{}) string {
	return color.New(color.FgHiYellow).Sprintf(format, a...)
}

// Success returns success-styled text
func Success(format string, a ...interface{}) string {
	return color.New(color.FgGreen).Sprintf(format, a...)
}

// Warn returns warning-styled text
func Warn(format string, a ...interface{}) string {
	return color.New(color.FgYellow).Sprintf(format, a...)
}

// Error returns error-styled text
func Error(format string, a ...interface{}) string {
	return color.New(color.FgRed).Sprintf(format, a...)
}

// Muted returns secondary/hint text
func Muted(format string, a ...interface{}) string {
	return color.New(color.FgHiBlack).Sprintf(format, a...)
}

// Heading returns bold accent text for section headers
func Heading(format string, a ...interface{}) string {
	return color.New(color.FgHiRed, color.Bold).Sprintf(format, a...)
}

// Command returns command/code styled text
func Command(format string, a ...interface{}) string {
	return color.New(color.FgCyan, color.Bold).Sprintf(format, a...)
}

// Option returns option/flag styled text
func Option(format string, a ...interface{}) string {
	return color.New(color.FgYellow).Sprintf(format, a...)
}

// Subtle returns subtle white text
func Subtle(format string, a ...interface{}) string {
	return color.New(color.FgWhite).Sprintf(format, a...)
}

// Bold returns bold white text
func Bold(format string, a ...interface{}) string {
	return color.New(color.FgWhite, color.Bold).Sprintf(format, a...)
}

// Primary returns primary magenta styled text
func Primary(format string, a ...interface{}) string {
	return color.New(color.FgMagenta, color.Bold).Sprintf(format, a...)
}

// Secondary returns secondary cyan styled text
func Secondary(format string, a ...interface{}) string {
	return color.New(color.FgCyan).Sprintf(format, a...)
}

// Colorize conditionally applies color based on rich mode
func Colorize(rich bool, colorFn func(string, ...interface{}) string, format string, a ...interface{}) string {
	if rich {
		return colorFn(format, a...)
	}
	if len(a) == 0 {
		return format
	}
	return color.New().Sprintf(format, a...)
}
