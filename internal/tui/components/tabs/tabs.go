// Package tabs renders the top tab strip.
package tabs

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/tui/theme"
)

// Zone is the inclusive column range [Start, End) of a single tab cell
// within the rendered row, already accounting for centering inside appWidth.
// All coordinates are relative to the left edge of the app widget (not the
// full terminal).
type Zone struct {
	Start int
	End   int
}

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

// Zones returns the click-zone for each tab label, in the same order as
// labels. The coordinates are app-relative (0 = leftmost column of the app).
// Use this together with a mouse click's X position to determine which tab
// was clicked.
func Zones(appWidth, current int, labels []string) []Zone {
	if appWidth < 1 {
		return nil
	}
	compact, padding := chooseRenderParams(appWidth, labels)
	return computeZones(appWidth, labels, compact, padding)
}

// chooseRenderParams mirrors the fallback logic in Render.
func chooseRenderParams(appWidth int, labels []string) (compact bool, padding int) {
	compact = appWidth < 90
	padding = 2
	if rowVisualWidth(labels, 2, compact) <= appWidth {
		return compact, 2
	}
	padding = 1
	if rowVisualWidth(labels, 1, compact) <= appWidth {
		return compact, 1
	}
	return true, 1 // final fallback: compact=true, padding=1
}

// rowVisualWidth computes the visual width of the full tab row without ANSI.
func rowVisualWidth(labels []string, padding int, compact bool) int {
	w := 0
	for i, label := range labels {
		lbl := effectiveLabel(label, compact)
		lbl = fmt.Sprintf("%d %s", i+1, lbl)
		if i > 0 {
			w++ // separator "·" = 1 visual column
		}
		w += utf8.RuneCountInString(lbl) + 2*padding
	}
	return w
}

// computeZones calculates the app-relative start/end columns of each tab.
func computeZones(appWidth int, labels []string, compact bool, padding int) []Zone {
	totalW := rowVisualWidth(labels, padding, compact)
	// lipgloss.Center centres by placing (appWidth-totalW)/2 spaces on the left.
	startX := (appWidth - totalW) / 2

	zones := make([]Zone, len(labels))
	x := startX
	for i, label := range labels {
		if i > 0 {
			x++ // skip the separator "·"
		}
		lbl := effectiveLabel(label, compact)
		lbl = fmt.Sprintf("%d %s", i+1, lbl)
		w := utf8.RuneCountInString(lbl) + 2*padding
		zones[i] = Zone{Start: x, End: x + w}
		x += w
	}
	return zones
}

func effectiveLabel(label string, compact bool) string {
	if compact {
		return compactLabel(label)
	}
	return label
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
