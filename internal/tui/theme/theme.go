package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name       string
	Primary    lipgloss.Color
	Accent     lipgloss.Color
	Muted      lipgloss.Color
	Background lipgloss.Color
	Foreground lipgloss.Color
	Border     lipgloss.Color
}

func PastelDark() Theme {
	return Theme{
		Name:       "pastel-dark",
		Primary:    lipgloss.Color("#c4b5fd"),
		Accent:     lipgloss.Color("#fbcfe8"),
		Muted:      lipgloss.Color("#9ca3af"),
		Background: lipgloss.Color("#1e1e2e"),
		Foreground: lipgloss.Color("#e5e7eb"),
		Border:     lipgloss.Color("#374151"),
	}
}
