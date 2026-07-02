package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorGreen  = lipgloss.Color("#1DB954")
	colorBlack  = lipgloss.Color("#191414")
	colorGray   = lipgloss.Color("#B3B3B3")
	colorDark   = lipgloss.Color("#282828")
	colorRed    = lipgloss.Color("#E91429")
	colorWhite  = lipgloss.Color("#FFFFFF")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGreen).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDark).
			Padding(1, 2)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			PaddingLeft(2)

	menuSelectedStyle = lipgloss.NewStyle().
				Foreground(colorBlack).
				Background(colorGreen).
				Bold(true).
				PaddingLeft(2)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorGray)

	valueStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			MarginTop(1)
)
