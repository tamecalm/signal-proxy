package ui

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Spinner frames for animation
var spinnerFrames = []string{"◐", "◓", "◑", "◒"}

// ProgressReporter interface for progress updates
type ProgressReporter interface {
	SetLabel(label string)
	SetPercent(percent int)
	Tick(delta int)
	Done()
}

// progressReporter implements ProgressReporter
type progressReporter struct {
	mu       sync.Mutex
	label    string
	percent  int
	total    int
	current  int
	started  bool
	done     bool
	stopChan chan struct{}
	frame    int
}

// CreateProgress creates a new progress reporter
func CreateProgress(label string, total int) ProgressReporter {
	if !isTTY() {
		// Return no-op for non-TTY
		return &noopProgress{}
	}

	p := &progressReporter{
		label:    label,
		total:    total,
		stopChan: make(chan struct{}),
	}

	// Start spinner goroutine
	go p.animate()

	return p
}

func (p *progressReporter) animate() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.mu.Lock()
			if p.done {
				p.mu.Unlock()
				return
			}
			p.render()
			p.frame = (p.frame + 1) % len(spinnerFrames)
			p.mu.Unlock()
		}
	}
}

func (p *progressReporter) render() {
	// Clear line
	fmt.Fprint(os.Stderr, "\r\033[K")

	spinner := spinnerFrames[p.frame]
	if IsRich() {
		spinner = Primary(spinner)
	}

	if p.total > 0 {
		bar := renderProgressBar(p.percent, 20)
		fmt.Fprintf(os.Stderr, "  %s %s %s %d%%",
			spinner,
			Subtle(p.label),
			bar,
			p.percent)
	} else {
		fmt.Fprintf(os.Stderr, "  %s %s",
			spinner,
			Subtle(p.label))
	}
}

func (p *progressReporter) SetLabel(label string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.label = label
}

func (p *progressReporter) SetPercent(percent int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	p.percent = percent
}

func (p *progressReporter) Tick(delta int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.total <= 0 {
		return
	}
	p.current += delta
	if p.current > p.total {
		p.current = p.total
	}
	p.percent = (p.current * 100) / p.total
}

func (p *progressReporter) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done {
		return
	}
	p.done = true
	close(p.stopChan)
	// Clear line
	fmt.Fprint(os.Stderr, "\r\033[K")
}

// renderProgressBar creates a simple progress bar
func renderProgressBar(percent, width int) string {
	filled := (percent * width) / 100
	if filled > width {
		filled = width
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	if IsRich() {
		return Accent(bar)
	}
	return bar
}

// noopProgress is a no-op progress reporter for non-TTY
type noopProgress struct{}

func (n *noopProgress) SetLabel(label string) {}
func (n *noopProgress) SetPercent(percent int) {}
func (n *noopProgress) Tick(delta int)         {}
func (n *noopProgress) Done()                  {}

// WithProgress runs a function with a progress indicator
func WithProgress(label string, total int, fn func(ProgressReporter) error) error {
	progress := CreateProgress(label, total)
	defer progress.Done()
	return fn(progress)
}

// Spinner shows a simple spinner for a message
func Spinner(label string) ProgressReporter {
	return CreateProgress(label, 0)
}
