// Package layout provides terminal-width breakpoint helpers for responsive rendering.
package layout

// Breakpoint thresholds (columns).
const (
	MinWidth    = 50  // graceful degraded floor
	NarrowMax   = 79  // narrow: single-column, no AI panel
	MediumMax   = 119 // medium: two columns, AI drawer
	WideMin     = 120 // wide: three columns — rail + detail + AI
	UltraWideMin = 180 // ultra-wide: generous padding everywhere
)

// IsNarrow reports whether width fits only a single-column layout (50–79).
func IsNarrow(w int) bool { return w >= MinWidth && w <= NarrowMax }

// IsMedium reports whether width supports two columns (80–119).
func IsMedium(w int) bool { return w >= NarrowMax+1 && w <= MediumMax }

// IsWide reports whether width supports the full three-column workspace (≥ 120).
func IsWide(w int) bool { return w >= WideMin }

// IsUltraWide reports whether width supports generous padding (≥ 180).
func IsUltraWide(w int) bool { return w >= UltraWideMin }

// RailWidth returns the items-rail width for a given terminal width.
func RailWidth(termW int) int {
	switch {
	case termW >= UltraWideMin:
		return 36
	case termW >= WideMin:
		return 30
	case termW >= NarrowMax+1:
		return 28
	default:
		return termW
	}
}

// AIPanelWidth returns the AI panel width for a given terminal width.
// Returns 0 when the panel should be hidden (narrow layouts).
func AIPanelWidth(termW int) int {
	switch {
	case termW >= UltraWideMin:
		return termW / 3
	case termW >= WideMin:
		return termW * 2 / 5
	default:
		return 0
	}
}
