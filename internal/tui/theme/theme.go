// Package theme defines color tokens and theme palettes for the TUI.
// Views never hardcode colors — they reference theme tokens or pull a
// pre-built lipgloss.Style from the theme.
package theme

import "github.com/charmbracelet/lipgloss"

// Theme is the full set of color tokens used across screens. New tokens belong
// here; views should not reach for raw colors.
type Theme struct {
	Name string

	// Surfaces
	Background  lipgloss.Color
	Surface     lipgloss.Color
	Overlay     lipgloss.Color
	Border      lipgloss.Color
	BorderFocus lipgloss.Color

	// Foreground tints
	Foreground lipgloss.Color
	Muted      lipgloss.Color
	Subtle     lipgloss.Color

	// Accents
	Primary     lipgloss.Color
	PrimaryGlow lipgloss.Color
	Accent      lipgloss.Color
	AccentGlow  lipgloss.Color

	// Gradient endpoints — used by the title wordmark and other glow effects.
	GradientFrom lipgloss.Color
	GradientTo   lipgloss.Color

	// Semantic
	Success lipgloss.Color
	Warning lipgloss.Color
	Danger  lipgloss.Color
	Info    lipgloss.Color
}
