// Package tabs renders the top tab strip.
package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/tui/theme"
)

// Render returns a centered tab strip. The route order is meaningful: AI and
// Sparks sit left of Dashboard, Projects and Settings sit right of it.
func Render(t theme.Theme, width int, current int, labels []string) string {
	if width < 1 {
		width = 1
	}

	compact := width < 90
	row := renderRow(t, current, labels, 2, compact)
	if lipgloss.Width(row) > width {
		row = renderRow(t, current, labels, 1, compact)
	}
	if lipgloss.Width(row) > width {
		row = renderRow(t, current, labels, 1, true)
	}
	row = lipgloss.NewStyle().MaxWidth(width).Render(row)

	row = lipgloss.PlaceHorizontal(width, lipgloss.Center, row,
		lipgloss.WithWhitespaceBackground(t.Background))

	bar := theme.Base(t).
		Width(width).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(t.Border)
	return bar.Render(row)
}

func renderRow(t theme.Theme, current int, labels []string, padding int, compact bool) string {
	cells := make([]string, 0, max(1, len(labels)*2-1))
	for i, label := range labels {
		lbl := label
		if compact {
			lbl = compactLabel(label)
		}
		lbl = fmt.Sprintf("%d %s", i+1, lbl)

		var cell string
		if i == current {
			cell = activePill(t, lbl, padding)
		} else {
			cell = theme.Fg(t, t.Muted).
				Padding(0, padding).
				Render(lbl)
		}
		if i > 0 {
			cells = append(cells,
				theme.Fg(t, t.Subtle).Render("·"),
			)
		}
		cells = append(cells, cell)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

func activePill(t theme.Theme, label string, padding int) string {
	pad := strings.Repeat(" ", padding)
	return theme.ANSIBackground(t.Primary) +
		theme.ANSIForeground(t.Background) +
		"\x1b[1m" +
		pad + label + pad +
		"\x1b[0m" +
		theme.ANSIBackground(t.Background)
}

func compactLabel(label string) string {
	switch label {
	case "Dashboard":
		return "Dash"
	case "Projects":
		return "Proj"
	case "Settings":
		return "Set"
	}
	return label
}
