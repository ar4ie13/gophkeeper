package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7C3AED") // violet
	colorSecondary = lipgloss.Color("#06B6D4") // cyan
	colorSuccess   = lipgloss.Color("#22C55E") // green
	colorWarning   = lipgloss.Color("#F59E0B") // amber
	colorDanger    = lipgloss.Color("#EF4444") // red
	colorThird     = lipgloss.Color("#188A00") // pink
	colorMuted     = lipgloss.Color("#6B7280") // gray
	colorText      = lipgloss.Color("#F9FAFB") // near-white
	colorBg        = lipgloss.Color("#1F2937") // dark bg

	// Reusable styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Italic(true)

	statusOnline = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	statusOffline = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	menuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	menuCursorStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	secretNameStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	secretValueStyle = lipgloss.NewStyle().
				Foreground(colorText)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	inputActiveStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Bold(true).
				Underline(true)

	inputInactiveStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	successMsgStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2).
			MarginBottom(1)

	headerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 2).
			MarginBottom(1).
			Bold(true).
			Foreground(colorText)
)
