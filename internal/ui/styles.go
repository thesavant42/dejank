// Package ui provides styled terminal output using lipgloss.
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Color palette - hacker aesthetic
var (
	ColorCyan    = lipgloss.Color("#00D9FF")
	ColorGreen   = lipgloss.Color("#00FF9F")
	ColorYellow  = lipgloss.Color("#FFD700")
	ColorRed     = lipgloss.Color("#FF6B6B")
	ColorMagenta = lipgloss.Color("#FF79C6")
	ColorGray    = lipgloss.Color("#666666")
	ColorWhite   = lipgloss.Color("#FAFAFA")
)

// Styles for different output types
var (
	// BannerStyle for the tool name and version
	BannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan)

	// SuccessStyle for positive messages [+]
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	// WarningStyle for warnings [-]
	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorYellow)

	// ErrorStyle for errors [!]
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed)

	// InfoStyle for informational messages [*]
	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorCyan)

	// DimStyle for secondary/less important text
	DimStyle = lipgloss.NewStyle().
			Foreground(ColorGray)

	// URLStyle for URLs and paths
	URLStyle = lipgloss.NewStyle().
			Foreground(ColorMagenta)

	// BoldStyle for emphasis
	BoldStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite)

	// LabelStyle for summary labels
	LabelStyle = lipgloss.NewStyle().
			Foreground(ColorGray)

	// ValueStyle for summary values
	ValueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite)
)

// Prefixes for log-style output
var (
	PrefixInfo    = InfoStyle.Render("[*]")
	PrefixSuccess = SuccessStyle.Render("[+]")
	PrefixWarning = WarningStyle.Render("[-]")
	PrefixError   = ErrorStyle.Render("[!]")
)

// Banner returns the styled banner with version
func Banner(version string) string {
	title := BannerStyle.Render("dejank")
	ver := DimStyle.Render(fmt.Sprintf("v%s", version))
	author := DimStyle.Render("by thesavant42")
	return fmt.Sprintf("\n%s %s %s\n", title, ver, author)
}

// Info prints an info message
func Info(msg string) string {
	return fmt.Sprintf("%s %s", PrefixInfo, msg)
}

// Success prints a success message
func Success(msg string) string {
	return fmt.Sprintf("%s %s", PrefixSuccess, msg)
}

// Warning prints a warning message
func Warning(msg string) string {
	return fmt.Sprintf("%s %s", PrefixWarning, msg)
}

// Error prints an error message
func Error(msg string) string {
	return fmt.Sprintf("%s %s", PrefixError, msg)
}

// Target formats a target URL/path
func Target(target string) string {
	return fmt.Sprintf("%s Target: %s\n", PrefixInfo, URLStyle.Render(target))
}

// SummaryLine formats a summary line with label and value
func SummaryLine(label string, value interface{}) string {
	return fmt.Sprintf("    %s %s",
		LabelStyle.Render(fmt.Sprintf("%-18s", label)),
		ValueStyle.Render(fmt.Sprintf("%v", value)))
}

// SummaryHeader returns the summary section header
func SummaryHeader() string {
	return fmt.Sprintf("\n%s Summary:", PrefixInfo)
}

