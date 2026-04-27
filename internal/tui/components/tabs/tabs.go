// Package tabs renders the top tab strip.
package tabs

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/tui/components/logo"
	"github.com/viphase/sparkle/internal/tui/theme"
)

// Render returns the tab strip: a small Sparkle wordmark on the left, then a
// row of tabs separated by dots, with a soft border underline tying the
// whole thing together.
func Render(t theme.Theme, width int, current int, labels []string) string {
	wordmark := lipgloss.NewStyle().Padding(0, 2).Render(logo.Compact(t))

	cells := make([]string, 0, len(labels)*2-1)
	for i, lbl := range labels {
		var cell string
		if i == current {
			cell = lipgloss.NewStyle().
				Foreground(t.Background).
				Background(t.Primary).
				Bold(true).
				Padding(0, 2).
				Render(lbl)
		} else {
			cell = lipgloss.NewStyle().
				Foreground(t.Muted).
				Padding(0, 2).
				Render(lbl)
		}
		if i > 0 {
			cells = append(cells,
				lipgloss.NewStyle().Foreground(t.Subtle).Render("·"),
			)
		}
		cells = append(cells, cell)
	}
	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, cells...)

	row := lipgloss.JoinHorizontal(lipgloss.Top, wordmark,
		lipgloss.NewStyle().Padding(0, 2).Render(tabRow),
	)

	bar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(t.Border)
	if width > 0 {
		bar = bar.Width(width)
	}
	return bar.Render(row)
}
