package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerResult holds the result of an async operation
type SpinnerResult struct {
	Data  interface{}
	Error error
}

// spinnerModel is the bubbletea model for the spinner
type spinnerModel struct {
	spinner  spinner.Model
	message  string
	done     bool
	result   SpinnerResult
	workFunc func() SpinnerResult
}

// workDoneMsg signals that the async work is complete
type workDoneMsg struct {
	result SpinnerResult
}

func newSpinnerModel(message string, workFunc func() SpinnerResult) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorCyan)
	return spinnerModel{
		spinner:  s,
		message:  message,
		workFunc: workFunc,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.doWork(),
	)
}

func (m spinnerModel) doWork() tea.Cmd {
	return func() tea.Msg {
		result := m.workFunc()
		return workDoneMsg{result: result}
	}
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.done = true
			m.result = SpinnerResult{Error: fmt.Errorf("interrupted")}
			return m, tea.Quit
		}

	case workDoneMsg:
		m.done = true
		m.result = msg.result
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), DimStyle.Render(m.message))
}

// RunWithSpinner executes workFunc while showing a spinner with the given message.
// Returns the result of the work function.
func RunWithSpinner(message string, workFunc func() SpinnerResult) SpinnerResult {
	m := newSpinnerModel(message, workFunc)
	p := tea.NewProgram(m)
	
	finalModel, err := p.Run()
	if err != nil {
		return SpinnerResult{Error: err}
	}
	
	return finalModel.(spinnerModel).result
}

// RunWithSpinnerSimple is a convenience wrapper for functions that return (data, error)
func RunWithSpinnerSimple[T any](message string, workFunc func() (T, error)) (T, error) {
	result := RunWithSpinner(message, func() SpinnerResult {
		data, err := workFunc()
		return SpinnerResult{Data: data, Error: err}
	})
	
	if result.Error != nil {
		var zero T
		return zero, result.Error
	}
	
	return result.Data.(T), nil
}

