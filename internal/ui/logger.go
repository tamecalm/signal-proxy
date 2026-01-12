package ui

import (
	"fmt"
	"strings"
	"time"
)

// Box-drawing characters for UI elements
const (
	boxTopLeft     = "╭"
	boxTopRight    = "╮"
	boxBottomLeft  = "╰"
	boxBottomRight = "╯"
	boxHorizontal  = "─"
	boxVertical    = "│"
	boxTeeRight    = "├"
	boxTeeLeft     = "┤"
)

// VERSION is the application version
const VERSION = "v1.0.0"

// PrintBanner displays the CLI header with version and tagline
func PrintBanner() {
	tagline := PickTagline()
	EmitSimpleBanner(VERSION, tagline)
}

// LogThinking displays a processing message with spinner icon
func LogThinking(message string) {
	ts := Muted(time.Now().Format("15:04:05"))
	spinner := Primary("◐")
	fmt.Printf("%s  %s  %s\n", ts, spinner, Subtle(message))
}

// LogStatus displays a status message with semantic styling
func LogStatus(category, message string) {
	ts := Muted(time.Now().Format("15:04:05"))

	var icon string
	var styledMsg string

	switch category {
	case "success":
		icon = Success("✓")
		styledMsg = Success(message)
	case "error":
		icon = Error("✗")
		styledMsg = Error(message)
	case "warning", "warn":
		icon = Warn("⚠")
		styledMsg = Warn(message)
	case "info":
		icon = Info("ℹ")
		styledMsg = Subtle(message)
	default:
		icon = Muted("●")
		styledMsg = Subtle(message)
	}

	fmt.Printf("%s  %s  %s\n", ts, icon, styledMsg)
}

// LogSection creates a styled section header
func LogSection(title string) {
	fmt.Println()
	header := fmt.Sprintf("%s %s %s",
		Muted("──"),
		Heading(title),
		Muted(strings.Repeat("─", 50-len(title))))
	fmt.Println(header)
}

// LogGroup starts a grouped block of messages
func LogGroup(title string) {
	fmt.Println()
	top := fmt.Sprintf("%s%s %s %s%s",
		Muted(boxTopLeft),
		Muted(strings.Repeat(boxHorizontal, 2)),
		Primary(title),
		Muted(strings.Repeat(boxHorizontal, 50-len(title))),
		Muted(boxTopRight))
	fmt.Println(top)
}

// LogGroupEnd closes a grouped block
func LogGroupEnd() {
	bottom := Muted(boxBottomLeft + strings.Repeat(boxHorizontal, 56) + boxBottomRight)
	fmt.Println(bottom)
	fmt.Println()
}

// LogGroupItem logs an item within a group
func LogGroupItem(label, value string) {
	line := fmt.Sprintf("%s  %s %s",
		Muted(boxVertical),
		Muted(label+":"),
		Secondary(value))
	fmt.Println(line)
}

// LogRelay displays relay connection info
func LogRelay(sni, clientIP string, up, down int64) {
	ts := Muted(time.Now().Format("15:04:05"))

	fmt.Printf("%s  %s  %s  %s  %s %s  %s %s\n",
		ts,
		Success("→"),
		Secondary(fmt.Sprintf("%-28s", sni)),
		Muted(fmt.Sprintf("%-16s", clientIP)),
		Muted("↑"), Subtle(fmt.Sprintf("%-8s", formatBytes(up))),
		Muted("↓"), Subtle(fmt.Sprintf("%-8s", formatBytes(down))))
}

// LogConnection shows a connection event
func LogConnection(event, target string) {
	ts := Muted(time.Now().Format("15:04:05"))

	var icon string
	switch event {
	case "connect":
		icon = Primary("◆")
	case "disconnect":
		icon = Muted("◇")
	default:
		icon = Muted("●")
	}

	fmt.Printf("%s  %s  %s\n", ts, icon, Secondary(target))
}

// LogMetric displays a metric value
func LogMetric(name string, value interface{}, unit string) {
	ts := Muted(time.Now().Format("15:04:05"))
	fmt.Printf("%s  %s  %s: %s %s\n",
		ts,
		Muted("◈"),
		Subtle(name),
		AccentBright(fmt.Sprintf("%v", value)),
		Muted(unit))
}

// formatBytes converts bytes to human-readable format
func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case b < KB:
		return fmt.Sprintf("%dB", b)
	case b < MB:
		return fmt.Sprintf("%.1fKB", float64(b)/KB)
	case b < GB:
		return fmt.Sprintf("%.1fMB", float64(b)/MB)
	default:
		return fmt.Sprintf("%.1fGB", float64(b)/GB)
	}
}

// PrintSeparator prints a horizontal separator
func PrintSeparator() {
	fmt.Println(Muted("  " + strings.Repeat("─", 56)))
}

// PrintFooter displays a footer message
func PrintFooter(message string) {
	fmt.Println()
	fmt.Printf("  %s %s\n", Muted("▸"), Muted(message))
}

// FormatError returns a rich error message with context
func FormatError(err error, solutions ...string) string {
	lines := []string{
		Error("✗ Error: %s", err.Error()),
		"",
	}

	if len(solutions) > 0 {
		lines = append(lines, Muted("Possible solutions:"))
		for _, s := range solutions {
			lines = append(lines, Muted("  • "+s))
		}
		lines = append(lines, "")
	}

	lines = append(lines, Muted("Docs: "+FormatDocsLink("/troubleshooting", "signal.org/docs")))

	return strings.Join(lines, "\n")
}

// LogGracefulShutdown logs the shutdown message
func LogGracefulShutdown() {
	fmt.Println()
	fmt.Printf("  %s %s\n", Muted("✋"), Muted("Cancelled"))
}
