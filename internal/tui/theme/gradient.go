package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/rivo/uniseg"
)

// ApplyGrad renders a string with a horizontal foreground gradient between two
// hex colors, one ramp step per grapheme cluster. Falls back to plain text if
// the colors fail to parse.
func ApplyGrad(text string, from, to lipgloss.Color, bold bool) string {
	if text == "" {
		return ""
	}
	f, ferr := colorful.Hex(string(from))
	t, terr := colorful.Hex(string(to))
	if ferr != nil || terr != nil {
		base := lipgloss.NewStyle()
		if bold {
			base = base.Bold(true)
		}
		return base.Render(text)
	}

	clusters := graphemeClusters(text)
	if len(clusters) == 0 {
		return ""
	}

	var sb strings.Builder
	n := len(clusters)
	for i, c := range clusters {
		var ratio float64
		if n > 1 {
			ratio = float64(i) / float64(n-1)
		}
		blended := f.BlendLuv(t, ratio).Clamped()
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(blended.Hex()))
		if bold {
			style = style.Bold(true)
		}
		sb.WriteString(style.Render(c))
	}
	return sb.String()
}

func graphemeClusters(s string) []string {
	gr := uniseg.NewGraphemes(s)
	var out []string
	for gr.Next() {
		out = append(out, string(gr.Runes()))
	}
	return out
}
