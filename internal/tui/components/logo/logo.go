// Package logo renders the Sparkle wordmark in ANSI Shadow block style,
// inspired by charmbracelet/crush's bold title treatment.
package logo

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/tui/theme"
)

// sparkleRows is SPARKLE in ANSI Shadow block font (56 terminal cells wide).
var sparkleRows = []string{
	"███████╗██████╗  █████╗ ██████╗ ██╗  ██╗██╗     ███████╗",
	"██╔════╝██╔══██╗██╔══██╗██╔══██╗██║ ██╔╝██║     ██╔════╝",
	"███████╗██████╔╝███████║██████╔╝█████╔╝ ██║     █████╗  ",
	"╚════██║██╔═══╝ ██╔══██║██╔══██╗██╔═██╗ ██║     ██╔══╝  ",
	"███████║██║     ██║  ██║██║  ██╗██║  ██╗███████╗███████╗",
	"╚══════╝╚═╝     ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚══════╝",
}

const blockWidth = 56

// Render produces the ANSI Shadow block wordmark with a per-row gradient.
// Falls back to Compact for widths narrower than the art.
func Render(t theme.Theme, width int) string {
	if width < blockWidth {
		return Compact(t)
	}
	rows := make([]string, len(sparkleRows))
	for i, row := range sparkleRows {
		rows[i] = theme.ApplyGradOn(row, t.GradientFrom, t.GradientTo, t.Background, true)
	}
	block := strings.Join(rows, "\n")
	sub := theme.Fg(t, t.Muted).Render("by viphase")
	return lipgloss.JoinVertical(lipgloss.Left, block, sub)
}

// Compact returns a single-line gradient "ꕤ SPARKLE" wordmark for narrow views.
func Compact(t theme.Theme) string {
	glyph := theme.Fg(t, t.Primary).Render("ꕤ")
	title := theme.ApplyGradOn("SPARKLE", t.GradientFrom, t.GradientTo, t.Background, true)
	return lipgloss.JoinHorizontal(lipgloss.Left, glyph, " ", title)
}
