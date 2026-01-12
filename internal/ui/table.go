package ui

import (
	"fmt"
	"strings"
)

// Align type for table column alignment
type Align int

const (
	AlignLeft Align = iota
	AlignRight
	AlignCenter
)

// TableColumn defines a column in a table
type TableColumn struct {
	Key      string
	Header   string
	Align    Align
	MinWidth int
	MaxWidth int
}

// TableBorder style for tables
type TableBorder int

const (
	BorderUnicode TableBorder = iota
	BorderASCII
	BorderNone
)

// RenderTableOptions configures table rendering
type RenderTableOptions struct {
	Columns []TableColumn
	Rows    []map[string]string
	Border  TableBorder
	Padding int
}

// Box drawing characters
type boxChars struct {
	tl, tr, bl, br string // corners
	h, v           string // horizontal, vertical
	t, ml, m, mr, b string // tees and crosses
}

var (
	unicodeBox = boxChars{
		tl: "┌", tr: "┐", bl: "└", br: "┘",
		h: "─", v: "│",
		t: "┬", ml: "├", m: "┼", mr: "┤", b: "┴",
	}
	asciiBox = boxChars{
		tl: "+", tr: "+", bl: "+", br: "+",
		h: "-", v: "|",
		t: "+", ml: "+", m: "+", mr: "+", b: "+",
	}
)

// RenderTable renders a formatted table
func RenderTable(opts RenderTableOptions) string {
	if opts.Padding == 0 {
		opts.Padding = 1
	}

	box := unicodeBox
	if opts.Border == BorderASCII {
		box = asciiBox
	}

	// Calculate column widths
	widths := make([]int, len(opts.Columns))
	for i, col := range opts.Columns {
		headerW := VisibleWidth(col.Header)
		maxCellW := 0
		for _, row := range opts.Rows {
			cellW := VisibleWidth(row[col.Key])
			if cellW > maxCellW {
				maxCellW = cellW
			}
		}
		base := headerW
		if maxCellW > base {
			base = maxCellW
		}
		base += opts.Padding * 2

		// Apply min/max constraints
		if col.MaxWidth > 0 && base > col.MaxWidth {
			base = col.MaxWidth
		}
		minW := col.MinWidth
		if minW < 3 {
			minW = 3
		}
		if base < minW {
			base = minW
		}
		widths[i] = base
	}

	// Helper functions
	hLine := func(left, mid, right string) string {
		parts := make([]string, len(widths))
		for i, w := range widths {
			parts[i] = strings.Repeat(box.h, w)
		}
		return left + strings.Join(parts, mid) + right
	}

	padCell := func(text string, width int, align Align) string {
		w := VisibleWidth(text)
		if w >= width {
			return text
		}
		pad := width - w
		switch align {
		case AlignRight:
			return spaces(pad) + text
		case AlignCenter:
			left := pad / 2
			return spaces(left) + text + spaces(pad-left)
		default:
			return text + spaces(pad)
		}
	}

	contentWidth := func(i int) int {
		w := widths[i] - opts.Padding*2
		if w < 1 {
			w = 1
		}
		return w
	}

	padStr := spaces(opts.Padding)

	renderRow := func(values []string) string {
		parts := make([]string, len(opts.Columns))
		for i, col := range opts.Columns {
			val := values[i]
			aligned := padCell(val, contentWidth(i), col.Align)
			parts[i] = padStr + aligned + padStr
		}
		return box.v + strings.Join(parts, box.v) + box.v
	}

	// Build table
	var lines []string

	if opts.Border != BorderNone {
		lines = append(lines, hLine(box.tl, box.t, box.tr))
	}

	// Header row
	headers := make([]string, len(opts.Columns))
	for i, col := range opts.Columns {
		headers[i] = col.Header
	}
	lines = append(lines, renderRow(headers))

	if opts.Border != BorderNone {
		lines = append(lines, hLine(box.ml, box.m, box.mr))
	}

	// Data rows
	for _, row := range opts.Rows {
		values := make([]string, len(opts.Columns))
		for i, col := range opts.Columns {
			values[i] = row[col.Key]
		}
		lines = append(lines, renderRow(values))
	}

	if opts.Border != BorderNone {
		lines = append(lines, hLine(box.bl, box.b, box.br))
	}

	return strings.Join(lines, "\n") + "\n"
}

// RenderSimpleTable renders a simple key-value table
func RenderSimpleTable(data map[string]string) string {
	var lines []string

	// Find max key width
	maxKey := 0
	for k := range data {
		if len(k) > maxKey {
			maxKey = len(k)
		}
	}

	for k, v := range data {
		line := fmt.Sprintf("  %s  %s",
			Muted(PadRight(k+":", maxKey+1)),
			Subtle(v))
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
