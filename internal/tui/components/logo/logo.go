// Package logo renders the Sparkle wordmark with a sparkle field, inspired by
// charmbracelet/crush's diagonal field + gradient title pattern.
package logo

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/tui/theme"
)

// Glyphs is the four-step pattern that makes up the sparkle field.
var glyphs = []string{"✦", "·", "✧", "·"}

// Render produces a centered title block: a sparkle field, the gradient
// "Sparkle" wordmark, a tagline, then another sparkle field. width is used to
// size the field; it does not pad the result.
func Render(t theme.Theme, width int) string {
	if width < 12 {
		width = 12
	}

	field := buildField(t, width)
	title := theme.ApplyGrad("✦  S P A R K L E  ✦", t.GradientFrom, t.GradientTo, true)
	tagline := lipgloss.NewStyle().
		Foreground(t.Muted).
		Italic(true).
		Render("turn rough sparks into structured projects")

	return lipgloss.JoinVertical(lipgloss.Center,
		field,
		"",
		title,
		tagline,
		"",
		field,
	)
}

// Compact returns a single-line version suitable for tight headers.
func Compact(t theme.Theme) string {
	return theme.ApplyGrad("✦ Sparkle", t.GradientFrom, t.GradientTo, true)
}

func buildField(t theme.Theme, width int) string {
	from, to := t.GradientFrom, t.GradientTo

	var sb strings.Builder
	cellWidth := 2 // glyph + space
	cells := width / cellWidth
	if cells < 4 {
		cells = 4
	}
	for i := 0; i < cells; i++ {
		sb.WriteString(glyphs[i%len(glyphs)])
		if i < cells-1 {
			sb.WriteByte(' ')
		}
	}
	// Use a soft gradient on the field too so the whole logo glows.
	return theme.ApplyGrad(sb.String(), from, to, false)
}
