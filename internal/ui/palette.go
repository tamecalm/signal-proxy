package ui

var CLI_PALETTE = struct {
	// Primary accent colors
	Accent       string // #FF5A2D - Primary brand color
	AccentBright string // #FF7A3D - Highlighted/active state
	AccentDim    string // #D14A22 - Muted accent

	// Semantic colors
	Info    string // #FF8A5B - Informational messages
	Success string // #2FBF71 - Success/completion
	Warn    string // #FFB020 - Warnings
	Error   string // #E23D2D - Errors

	// Neutral
	Muted string // #8B7F77 - Secondary text, hints, metadata
}{
	Accent:       "#FF5A2D",
	AccentBright: "#FF7A3D",
	AccentDim:    "#D14A22",
	Info:         "#FF8A5B",
	Success:      "#2FBF71",
	Warn:         "#FFB020",
	Error:        "#E23D2D",
	Muted:        "#8B7F77",
}
