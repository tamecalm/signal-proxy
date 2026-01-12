package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// ASCII art banner for Signal Proxy
var asciiBanner = []string{
	"░██████╗██╗░██████╗░███╗░░██╗░█████╗░██╗░░░░░",
	"██╔════╝██║██╔════╝░████╗░██║██╔══██╗██║░░░░░",
	"╚█████╗░██║██║░░██╗░██╔██╗██║███████║██║░░░░░",
	"░╚═══██╗██║██║░░╚██╗██║╚████║██╔══██║██║░░░░░",
	"██████╔╝██║╚██████╔╝██║░╚███║██║░░██║███████╗",
	"╚═════╝░╚═╝░╚═════╝░╚═╝░░╚══╝╚═╝░░╚═╝╚══════╝",
}

// Simpler fallback banner
var simpleBanner = []string{
	"███████╗██╗ ██████╗ ███╗   ██╗ █████╗ ██╗     ",
	"██╔════╝██║██╔════╝ ████╗  ██║██╔══██╗██║     ",
	"███████╗██║██║  ███╗██╔██╗ ██║███████║██║     ",
	"╚════██║██║██║   ██║██║╚██╗██║██╔══██║██║     ",
	"███████║██║╚██████╔╝██║ ╚████║██║  ██║███████╗",
	"╚══════╝╚═╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝╚══════╝",
}

var bannerEmitted = false

// FormatBannerArt returns the ASCII banner with gradient coloring
func FormatBannerArt() string {
	rich := IsRich()
	if !rich {
		return strings.Join(simpleBanner, "\n")
	}

	// Color styles
	accent := color.New(color.FgHiRed, color.Bold)
	accentDim := color.New(color.FgRed)

	var lines []string
	for _, line := range simpleBanner {
		var coloredLine strings.Builder
		for _, ch := range line {
			switch ch {
			case '█', '╗', '╔', '╚', '╝', '║':
				coloredLine.WriteString(accent.Sprint(string(ch)))
			case '░', '═', '╣', '╠':
				coloredLine.WriteString(accentDim.Sprint(string(ch)))
			default:
				coloredLine.WriteString(Muted("%c", ch))
			}
		}
		lines = append(lines, coloredLine.String())
	}
	return strings.Join(lines, "\n")
}

// FormatBannerLine returns the version/tagline line
func FormatBannerLine(version, tagline string) string {
	rich := IsRich()
	title := "◆ SIGNAL PROXY"

	if rich {
		return fmt.Sprintf("%s %s %s %s",
			Heading(title),
			Info(version),
			Muted("—"),
			AccentDim(tagline))
	}
	return fmt.Sprintf("%s %s — %s", title, version, tagline)
}

// EmitBanner displays the banner once, respecting TTY and flags
func EmitBanner(version, tagline string) {
	if bannerEmitted {
		return
	}
	if !isTTY() {
		return
	}
	// Skip for --json or --version flags
	for _, arg := range os.Args {
		if arg == "--json" || arg == "--version" || arg == "-v" {
			return
		}
	}

	fmt.Println()
	fmt.Println(FormatBannerArt())
	fmt.Println()
	fmt.Println(FormatBannerLine(version, tagline))
	fmt.Println()
	bannerEmitted = true
}

// EmitSimpleBanner displays a simpler boxed banner (current style)
func EmitSimpleBanner(version, tagline string) {
	if bannerEmitted {
		return
	}
	if !isTTY() {
		return
	}

	fmt.Println()

	// Product badge
	badge := color.New(color.BgMagenta, color.FgWhite, color.Bold).Sprint(" ◆ SIGNAL ")
	ver := Muted(version)

	// Top border
	topBorder := Muted(boxTopLeft + strings.Repeat(boxHorizontal, 60) + boxTopRight)
	fmt.Println(topBorder)

	// Title line
	titleLine := fmt.Sprintf("%s  %s %s  %s",
		Muted(boxVertical),
		badge,
		ver,
		Muted(strings.Repeat(" ", 36)+boxVertical))
	fmt.Println(titleLine)

	// Subtitle
	subtitle := Subtle(tagline)
	subtitleLine := fmt.Sprintf("%s  %s%s",
		Muted(boxVertical),
		subtitle,
		Muted(strings.Repeat(" ", 60-2-len(tagline))+boxVertical))
	fmt.Println(subtitleLine)

	// Bottom border
	bottomBorder := Muted(boxBottomLeft + strings.Repeat(boxHorizontal, 60) + boxBottomRight)
	fmt.Println(bottomBorder)
	fmt.Println()

	bannerEmitted = true
}

// isTTY checks if stdout is a terminal
func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ResetBanner allows banner to be shown again (for testing)
func ResetBanner() {
	bannerEmitted = false
}
