package ui

import (
	"math/rand"
	"strings"
	"time"
)

// Default tagline fallback
const defaultTagline = "Trusted Proxy Service"

// Tagline pool with personality
var taglines = []string{
	"Trusted Proxy Service",
	"Your Signal, Relayed Securely",
	"Privacy-first proxy routing",
	"Connecting you to Signal, anywhere",
	"Secure relay for the modern era",
	"Routing trust, one packet at a time",
	"Making censorship obsolete since 2024",
	"Where privacy meets performance",
	"Silent guardian of your Signal",
	"Tunneling through barriers",
}

// Holiday-specific taglines
var holidayTaglines = map[string][]taglineRule{
	"christmas": {
		{month: 12, day: 25, tagline: "🎄 Ho ho ho—relaying holiday cheer!"},
		{month: 12, day: 24, tagline: "🎄 Santa's favorite proxy service"},
	},
	"halloween": {
		{month: 10, day: 31, tagline: "🎃 Boo! Your packets are haunted"},
		{month: 10, day: 30, tagline: "🎃 Spooky secure connections"},
	},
	"valentine": {
		{month: 2, day: 14, tagline: "💘 Sending love through encrypted channels"},
	},
	"newyear": {
		{month: 1, day: 1, tagline: "🎉 Happy New Year! Fresh connections await"},
	},
}

type taglineRule struct {
	month   int
	day     int
	tagline string
}

// PickTagline returns a random tagline, considering holidays
func PickTagline() string {
	now := time.Now()
	month := int(now.Month())
	day := now.Day()

	// Check for holiday-specific taglines
	for _, rules := range holidayTaglines {
		for _, rule := range rules {
			if rule.month == month && rule.day == day {
				return rule.tagline
			}
		}
	}

	// Random selection from pool
	if len(taglines) == 0 {
		return defaultTagline
	}

	// Use current time for seed variation
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return taglines[r.Intn(len(taglines))]
}

// GetAllTaglines returns all available taglines (for testing/display)
func GetAllTaglines() []string {
	return append([]string{}, taglines...)
}

// FormatTagline wraps a tagline with optional styling
func FormatTagline(tagline string) string {
	if !IsRich() {
		return tagline
	}
	// Highlight emojis differently
	if strings.HasPrefix(tagline, "🎄") ||
		strings.HasPrefix(tagline, "🎃") ||
		strings.HasPrefix(tagline, "💘") ||
		strings.HasPrefix(tagline, "🎉") {
		return tagline // Keep emojis as-is
	}
	return AccentDim(tagline)
}
