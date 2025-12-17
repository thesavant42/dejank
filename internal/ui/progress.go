package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// progressModel is the bubbletea model for the progress bar
type progressModel struct {
	progress progress.Model
	percent  float64
	done     bool
	message  string
	total    int
	current  int
	updates  chan int
	quit     chan bool
}

type tickMsg time.Time
type updateMsg int
type quitMsg struct{}

func (m progressModel) Init() tea.Cmd {
	return tea.Batch(
		m.waitForUpdate(),
		m.progress.Init(),
	)
}

func (m progressModel) waitForUpdate() tea.Cmd {
	return func() tea.Msg {
		select {
		case n := <-m.updates:
			return updateMsg(n)
		case <-m.quit:
			return quitMsg{}
		}
	}
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, nil // Ignore key presses

	case updateMsg:
		m.current = int(msg)
		if m.total > 0 {
			m.percent = float64(m.current) / float64(m.total)
		}
		if m.percent >= 1.0 {
			m.done = true
			return m, tea.Quit
		}
		cmd := m.progress.SetPercent(m.percent)
		return m, tea.Batch(cmd, m.waitForUpdate())

	case quitMsg:
		m.done = true
		return m, tea.Quit

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}
	return m, nil
}

func (m progressModel) View() string {
	status := fmt.Sprintf("%d/%d", m.current, m.total)
	percentStr := fmt.Sprintf("%.0f%%", m.percent*100)

	return fmt.Sprintf("%s %s %s %s %s\n",
		PrefixInfo,
		TextStyle.Render(m.message),
		m.progress.View(),
		ValueStyle.Render(status),
		AccentStyle.Render(percentStr))
}

// Progress wraps a bubbletea program for progress display
type Progress struct {
	program *tea.Program
	updates chan int
	quit    chan bool
	total   int
	current int
}

// NewProgress creates a new progress bar using bubbles
func NewProgress(total int, message string) *Progress {
	updates := make(chan int, 100)
	quit := make(chan bool)

	// Create progress bar with gradient
	bar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(30),
		progress.WithoutPercentage(),
	)

	model := progressModel{
		progress: bar,
		message:  message,
		total:    total,
		updates:  updates,
		quit:     quit,
	}

	p := tea.NewProgram(model, tea.WithOutput(os.Stderr))

	prog := &Progress{
		program: p,
		updates: updates,
		quit:    quit,
		total:   total,
	}

	// Start the program in background
	go func() {
		p.Run()
	}()

	return prog
}

// Increment advances progress by 1
func (p *Progress) Increment() {
	p.current++
	select {
	case p.updates <- p.current:
	default:
	}
}

// SetCurrent sets the current progress value
func (p *Progress) SetCurrent(n int) {
	p.current = n
	select {
	case p.updates <- n:
	default:
	}
}

// Done completes the progress bar
func (p *Progress) Done() {
	p.current = p.total
	select {
	case p.updates <- p.total:
	default:
	}
	time.Sleep(100 * time.Millisecond) // Let animation finish
	close(p.quit)
	p.program.Wait()
}

// SimpleSpinner shows a simple inline spinner for short operations
type SimpleSpinner struct {
	message string
	frames  []string
	current int
	done    chan bool
	style   lipgloss.Style
}

// NewSimpleSpinner creates a new simple spinner
func NewSimpleSpinner(message string) *SimpleSpinner {
	return &SimpleSpinner{
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:    make(chan bool),
		style:   lipgloss.NewStyle().Foreground(ColorCyan),
	}
}

// Start begins the spinner animation
func (s *SimpleSpinner) Start() {
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.current = (s.current + 1) % len(s.frames)
				frame := s.frames[s.current]
				fmt.Printf("\r%s %s %s   ",
					PrefixInfo,
					TextStyle.Render(s.message),
					s.style.Render(frame))
			}
		}
	}()
}

// Stop stops the spinner and clears the line
func (s *SimpleSpinner) Stop() {
	close(s.done)
	fmt.Printf("\r%s\r", strings.Repeat(" ", 60))
}

// StopWithMessage stops and prints a final message
func (s *SimpleSpinner) StopWithMessage(msg string) {
	close(s.done)
	fmt.Printf("\r%s\n", msg)
}
