package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Claude Code CLI-inspired color palette
var (
	// Primary colors
	clrDim     = color.New(color.FgHiBlack)
	clrSubtle  = color.New(color.FgWhite)
	clrBold    = color.New(color.FgWhite, color.Bold)
	
	// Accent colors (Claude-style)
	clrPrimary  = color.New(color.FgMagenta, color.Bold)
	clrSecondary = color.New(color.FgCyan)
	clrAccent   = color.New(color.FgCyan, color.Bold)
	
	// Status colors
	clrSuccess = color.New(color.FgGreen)
	clrError   = color.New(color.FgRed)
	clrWarning = color.New(color.FgYellow)
	clrInfo    = color.New(color.FgBlue)
	
	// Badge styles
	badgePrimary = color.New(color.BgMagenta, color.FgWhite, color.Bold)
	badgeSuccess = color.New(color.BgGreen, color.FgBlack)
	badgeError   = color.New(color.BgRed, color.FgWhite)
	badgeInfo    = color.New(color.BgBlue, color.FgWhite)
)

// Box-drawing characters for Claude-style UI
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

// PrintBanner displays a Claude Code CLI-style header
func PrintBanner() {
	fmt.Println()
	
	// Product badge with version-style formatting
	badge := badgePrimary.Sprint(" ◆ SIGNAL ")
	version := clrDim.Sprint("v1.0.0")
	
	// Top border
	topBorder := clrDim.Sprint(boxTopLeft + strings.Repeat(boxHorizontal, 60) + boxTopRight)
	fmt.Println(topBorder)
	
	// Title line with padding
	titleLine := fmt.Sprintf("%s  %s %s  %s",
		clrDim.Sprint(boxVertical),
		badge,
		version,
		clrDim.Sprint(strings.Repeat(" ", 36) + boxVertical))
	fmt.Println(titleLine)
	
	// Subtitle
	subtitle := clrSubtle.Sprint("Trusted Proxy Service")
	subtitleLine := fmt.Sprintf("%s  %s%s",
		clrDim.Sprint(boxVertical),
		subtitle,
		clrDim.Sprint(strings.Repeat(" ", 38) + boxVertical))
	fmt.Println(subtitleLine)
	
	// Bottom border
	bottomBorder := clrDim.Sprint(boxBottomLeft + strings.Repeat(boxHorizontal, 60) + boxBottomRight)
	fmt.Println(bottomBorder)
	fmt.Println()
}

// LogThinking displays a Claude-style thinking/processing message
func LogThinking(message string) {
	ts := clrDim.Sprint(time.Now().Format("15:04:05"))
	spinner := clrPrimary.Sprint("◐")
	fmt.Printf("%s  %s  %s\n", ts, spinner, clrSubtle.Sprint(message))
}

// LogStatus displays a status message with appropriate styling
func LogStatus(category, message string) {
	ts := clrDim.Sprint(time.Now().Format("15:04:05"))
	
	var icon string
	var styledMsg string
	
	switch category {
	case "success":
		icon = clrSuccess.Sprint("✔")
		styledMsg = clrSuccess.Sprint(message)
	case "error":
		icon = clrError.Sprint("✖")
		styledMsg = clrError.Sprint(message)
	case "warning":
		icon = clrWarning.Sprint("⚠")
		styledMsg = clrWarning.Sprint(message)
	case "info":
		icon = clrInfo.Sprint("ℹ")
		styledMsg = clrSubtle.Sprint(message)
	default:
		icon = clrDim.Sprint("●")
		styledMsg = clrSubtle.Sprint(message)
	}
	
	fmt.Printf("%s  %s  %s\n", ts, icon, styledMsg)
}

// LogSection creates a Claude-style section header
func LogSection(title string) {
	fmt.Println()
	header := fmt.Sprintf("%s %s %s",
		clrDim.Sprint("──"),
		clrAccent.Sprint(title),
		clrDim.Sprint(strings.Repeat("─", 50-len(title))))
	fmt.Println(header)
}

// LogGroup starts a grouped block of messages (Claude-style box)
func LogGroup(title string) {
	fmt.Println()
	top := clrDim.Sprintf("%s%s %s %s%s",
		boxTopLeft,
		strings.Repeat(boxHorizontal, 2),
		clrPrimary.Sprint(title),
		clrDim.Sprint(strings.Repeat(boxHorizontal, 50-len(title))),
		boxTopRight)
	fmt.Println(top)
}

// LogGroupEnd closes a grouped block
func LogGroupEnd() {
	bottom := clrDim.Sprint(boxBottomLeft + strings.Repeat(boxHorizontal, 56) + boxBottomRight)
	fmt.Println(bottom)
	fmt.Println()
}

// LogGroupItem logs an item within a group
func LogGroupItem(label, value string) {
	line := fmt.Sprintf("%s  %s %s",
		clrDim.Sprint(boxVertical),
		clrDim.Sprint(label+":"),
		clrAccent.Sprint(value))
	fmt.Println(line)
}

// LogRelay displays relay connection info in Claude-style format
func LogRelay(sni, clientIP string, up, down int64) {
	ts := clrDim.Sprint(time.Now().Format("15:04:05"))
	
	// Clean, aligned output with Claude-style formatting
	fmt.Printf("%s  %s  %s  %s  %s %s  %s %s\n",
		ts,
		clrSuccess.Sprint("→"),
		clrAccent.Sprintf("%-28s", sni),
		clrDim.Sprintf("%-16s", clientIP),
		clrDim.Sprint("↑"), clrSubtle.Sprintf("%-8s", formatBytes(up)),
		clrDim.Sprint("↓"), clrSubtle.Sprintf("%-8s", formatBytes(down)))
}

// LogConnection shows a new connection event
func LogConnection(event, target string) {
	ts := clrDim.Sprint(time.Now().Format("15:04:05"))
	
	var icon string
	switch event {
	case "connect":
		icon = clrPrimary.Sprint("◆")
	case "disconnect":
		icon = clrDim.Sprint("◇")
	default:
		icon = clrDim.Sprint("●")
	}
	
	fmt.Printf("%s  %s  %s\n", ts, icon, clrSecondary.Sprint(target))
}

// LogMetric displays a metric value
func LogMetric(name string, value interface{}, unit string) {
	ts := clrDim.Sprint(time.Now().Format("15:04:05"))
	fmt.Printf("%s  %s  %s: %s %s\n",
		ts,
		clrDim.Sprint("◈"),
		clrSubtle.Sprint(name),
		clrAccent.Sprintf("%v", value),
		clrDim.Sprint(unit))
}

// formatBytes converts bytes to human-readable format
func formatBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%dB", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(b)/1024)
	}
	if b < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	}
	return fmt.Sprintf("%.1fGB", float64(b)/(1024*1024*1024))
}

// PrintSeparator prints a subtle horizontal separator
func PrintSeparator() {
	fmt.Println(clrDim.Sprint("  " + strings.Repeat("─", 56)))
}

// PrintFooter displays a Claude-style footer message
func PrintFooter(message string) {
	fmt.Println()
	fmt.Printf("  %s %s\n", clrDim.Sprint("▸"), clrDim.Sprint(message))
}
