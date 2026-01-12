package ui

import (
	"fmt"
	"os"
	"strings"
)

// DOCS_ROOT is the base URL for documentation links
const DOCS_ROOT = "https://signal.org/docs"

// SupportsHyperlinks checks if the terminal supports OSC-8 hyperlinks
func SupportsHyperlinks() bool {
	// Check for known terminals that support OSC-8
	termProgram := os.Getenv("TERM_PROGRAM")
	term := os.Getenv("TERM")
	wtSession := os.Getenv("WT_SESSION") // Windows Terminal

	// Known supporting terminals
	if strings.Contains(termProgram, "iTerm") ||
		strings.Contains(termProgram, "WezTerm") ||
		strings.Contains(termProgram, "vscode") ||
		strings.Contains(termProgram, "Hyper") ||
		wtSession != "" {
		return true
	}

	// xterm-256color often supports it in modern terminals
	if strings.Contains(term, "xterm-256color") {
		return true
	}

	return false
}

// FormatTerminalLink creates an OSC-8 hyperlink if supported
// Falls back to "label (url)" format if not supported
func FormatTerminalLink(label, url string) string {
	if !SupportsHyperlinks() {
		return fmt.Sprintf("%s (%s)", label, url)
	}

	// OSC-8 format: ESC ] 8 ; ; URL ST text ESC ] 8 ; ; ST
	// ST (String Terminator) = ESC \
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, label)
}

// FormatDocsLink creates a documentation link
func FormatDocsLink(path, label string) string {
	url := path
	if !strings.HasPrefix(path, "http") {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		url = DOCS_ROOT + path
	}

	if label == "" {
		label = url
	}

	return FormatTerminalLink(label, url)
}

// FormatURLWithStyle creates a colored link
func FormatURLWithStyle(label, url string) string {
	link := FormatTerminalLink(label, url)
	if IsRich() {
		return Secondary(link)
	}
	return link
}

// FormatEmail creates an email link
func FormatEmail(email string) string {
	return FormatTerminalLink(email, "mailto:"+email)
}
