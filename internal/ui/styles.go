// Package ui provides styled terminal output using lipgloss.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Cyberpunk 2077 / Luxium color palette
var (
	ColorPink    = lipgloss.Color("#deb7ff") // Primary text - lighter magenta
	ColorYellow  = lipgloss.Color("#FFFF00") // Keywords, accents, headers - pure yellow
	ColorCyan    = lipgloss.Color("#00FFFF") // Info, URLs, highlights - pure cyan
	ColorGreen   = lipgloss.Color("#00FF00") // Success, progress - pure green
	ColorOrange  = lipgloss.Color("#FF6600") // Warnings
	ColorRed     = lipgloss.Color("#FF0000") // Errors - pure red
	ColorMagenta = lipgloss.Color("#FF44FF") // Secondary accents - lighter magenta
	ColorDim     = lipgloss.Color("#FFFFFF") // Muted text - white
)

// Base styles - define once, reuse everywhere
var (
	// TextStyle for primary body text
	TextStyle = lipgloss.NewStyle().
			Foreground(ColorPink)

	// AccentStyle for headers, titles, emphasis
	AccentStyle = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)

	// InfoStyle for info messages and prefixes
	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorCyan)

	// SuccessStyle for success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	// WarningStyle for warnings
	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorOrange)

	// ErrorStyle for errors
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed)

	// DimStyle for muted/secondary text and <placeholders>
	DimStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	// BracketStyle for [optional] parameters - high contrast
	BracketStyle = lipgloss.NewStyle().
			Foreground(ColorOrange)

	// URLStyle for URLs and paths
	URLStyle = lipgloss.NewStyle().
			Foreground(ColorCyan).
			Underline(true)

	// LabelStyle for summary labels
	LabelStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	// ValueStyle for summary values
	ValueStyle = lipgloss.NewStyle().
			Foreground(ColorPink).
			Bold(true)

	// SummaryBoxStyle for the summary panel
	SummaryBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorYellow).
			Padding(0, 1).
			MarginTop(1)
)

// Rendered prefixes
var (
	PrefixInfo    = InfoStyle.Render("[*]")
	PrefixSuccess = SuccessStyle.Render("[+]")
	PrefixWarning = WarningStyle.Render("[-]")
	PrefixError   = ErrorStyle.Render("[!]")
)

// Banner returns the styled banner with version
func Banner(version string) string {
	title := AccentStyle.Render("dejank")
	ver := DimStyle.Render(fmt.Sprintf("v%s", version))
	author := TextStyle.Render("by thesavant42")
	return fmt.Sprintf("\n%s %s %s\n", title, ver, author)
}

// Info renders an info message with cyan prefix and pink body
func Info(msg string) string {
	return fmt.Sprintf("%s %s", PrefixInfo, TextStyle.Render(msg))
}

// Success renders a success message with green prefix and pink body
func Success(msg string) string {
	return fmt.Sprintf("%s %s", PrefixSuccess, TextStyle.Render(msg))
}

// Warning renders a warning message with orange prefix and pink body
func Warning(msg string) string {
	return fmt.Sprintf("%s %s", PrefixWarning, TextStyle.Render(msg))
}

// Error renders an error message with red prefix and pink body
func Error(msg string) string {
	return fmt.Sprintf("%s %s", PrefixError, TextStyle.Render(msg))
}

// Target formats a target URL/path with styled output
func Target(target string) string {
	return fmt.Sprintf("%s %s %s\n", PrefixInfo, TextStyle.Render("Target:"), URLStyle.Render(target))
}

// SummaryLine formats a summary line with label and value
func SummaryLine(label string, value interface{}) string {
	return fmt.Sprintf("  %s %s",
		LabelStyle.Render(fmt.Sprintf("%-18s", label)),
		ValueStyle.Render(fmt.Sprintf("%v", value)))
}

// SummaryHeader returns the summary section header
func SummaryHeader() string {
	return fmt.Sprintf("\n%s %s", PrefixInfo, AccentStyle.Render("Summary"))
}

// RenderSummaryBox wraps content in a styled summary box
func RenderSummaryBox(lines ...string) string {
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return SummaryBoxStyle.Render(content)
}

// FormatUsage styles a usage string, applying different colors to:
// - <placeholder> = white (DimStyle)
// - [optional] = orange (BracketStyle)
// - everything else = pink (TextStyle)
func FormatUsage(usage string) string {
	var result strings.Builder
	i := 0
	for i < len(usage) {
		switch usage[i] {
		case '<':
			// Find closing >
			end := strings.Index(usage[i:], ">")
			if end != -1 {
				result.WriteString(DimStyle.Render(usage[i : i+end+1]))
				i += end + 1
			} else {
				result.WriteString(TextStyle.Render(string(usage[i])))
				i++
			}
		case '[':
			// Find closing ]
			end := strings.Index(usage[i:], "]")
			if end != -1 {
				result.WriteString(BracketStyle.Render(usage[i : i+end+1]))
				i += end + 1
			} else {
				result.WriteString(TextStyle.Render(string(usage[i])))
				i++
			}
		case ' ':
			result.WriteString(" ")
			i++
		default:
			// Find next special char or space
			end := i
			for end < len(usage) && usage[end] != '<' && usage[end] != '[' && usage[end] != ' ' {
				end++
			}
			result.WriteString(TextStyle.Render(usage[i:end]))
			i = end
		}
	}
	return result.String()
}
