package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Progress displays a progress bar with percentage
type Progress struct {
	total     int
	current   int
	message   string
	width     int
	mu        sync.Mutex
	startTime time.Time
}

// NewProgress creates a new progress tracker
func NewProgress(total int, message string) *Progress {
	return &Progress{
		total:     total,
		message:   message,
		width:     30,
		startTime: time.Now(),
	}
}

// Increment advances the progress by 1
func (p *Progress) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current++
	p.render()
}

// SetCurrent sets the current progress value
func (p *Progress) SetCurrent(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = n
	p.render()
}

// Done completes the progress bar
func (p *Progress) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = p.total
	p.render()
	fmt.Println() // Move to next line
}

// render displays the progress bar (overwrites current line)
func (p *Progress) render() {
	if p.total == 0 {
		return
	}

	percent := float64(p.current) / float64(p.total)
	filled := int(percent * float64(p.width))
	empty := p.width - filled

	bar := SuccessStyle.Render(strings.Repeat("=", filled)) +
		DimStyle.Render(strings.Repeat("-", empty))

	status := fmt.Sprintf("%d/%d", p.current, p.total)
	percentStr := fmt.Sprintf("%.0f%%", percent*100)

	// Carriage return to overwrite line
	fmt.Printf("\r%s %s [%s] %s %s",
		PrefixInfo,
		DimStyle.Render(p.message),
		bar,
		ValueStyle.Render(status),
		DimStyle.Render(percentStr))
}

// SimpleSpinner shows a simple inline spinner for short operations
type SimpleSpinner struct {
	message string
	frames  []string
	current int
	done    chan bool
	mu      sync.Mutex
}

// NewSimpleSpinner creates a new simple spinner
func NewSimpleSpinner(message string) *SimpleSpinner {
	return &SimpleSpinner{
		message: message,
		frames:  []string{".", "..", "..."},
		done:    make(chan bool),
	}
}

// Start begins the spinner animation
func (s *SimpleSpinner) Start() {
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.mu.Lock()
				s.current = (s.current + 1) % len(s.frames)
				frame := s.frames[s.current]
				s.mu.Unlock()
				fmt.Printf("\r%s %s%s   ",
					PrefixInfo,
					DimStyle.Render(s.message),
					DimStyle.Render(frame))
			}
		}
	}()
}

// Stop stops the spinner and clears the line
func (s *SimpleSpinner) Stop() {
	close(s.done)
	// Clear the spinner line
	fmt.Printf("\r%s\r", strings.Repeat(" ", 60))
}

// StopWithMessage stops and prints a final message
func (s *SimpleSpinner) StopWithMessage(msg string) {
	close(s.done)
	fmt.Printf("\r%s\n", msg)
}

