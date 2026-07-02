package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderProgressBar(fraction float64, width int) string {
	width = max(10, width)
	fraction = max(0, min(1, fraction))

	filled := int(math.Round(float64(width) * fraction))
	empty := width - filled

	bar := lipgloss.NewStyle().Foreground(colorGreen).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(colorDark).Render(strings.Repeat("░", empty))

	return bar
}

func progressLine(completed, total int, workers int) string {
	percent := 0
	if total > 0 {
		percent = int(math.Round(float64(completed) / float64(total) * 100))
	}
	return subtitleStyle.Render(fmtPercent(completed, total, percent, workers))
}

func fmtPercent(completed, total, percent, workers int) string {
	return fmt.Sprintf("%d%%  %d / %d tracks  •  %d workers", percent, completed, total, workers)
}
