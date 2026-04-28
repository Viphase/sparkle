// Package theme defines color tokens and theme palettes for the TUI.
// Views never hardcode colors — they reference theme tokens or pull a
// pre-built lipgloss.Style from the theme.
package theme

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

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

// Base returns the default app surface. Prefer building view styles from this
// so styled text does not fall back to the terminal's transparent background.
func Base(t Theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Background(t.Background).
		Foreground(t.Foreground)
}

// Fg returns a text style with the app background and the requested foreground.
func Fg(t Theme, color lipgloss.Color) lipgloss.Style {
	return Base(t).Foreground(color)
}

// Surface returns a raised surface style while keeping app text defaults.
func SurfaceStyle(t Theme) lipgloss.Style {
	return Base(t)
}

// Place centers or aligns content in an app-background-filled rectangle.
func Place(t Theme, width, height int, hPos, vPos lipgloss.Position, str string) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return Base(t).
		Width(width).
		Height(height).
		Render(lipgloss.Place(width, height, hPos, vPos, str,
			lipgloss.WithWhitespaceBackground(t.Background)))
}

// PaintBackground forces the app background across a fixed terminal canvas.
// Lip Gloss nested styles emit resets; this re-applies the app background after
// those resets and pads each line so blank cells are painted too.
func PaintBackground(t Theme, width, height int, content string) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	bg := ANSIBackground(t.Background)
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}

	for i, line := range lines {
		line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bg)
		line = strings.ReplaceAll(line, "\x1b[m", "\x1b[m"+bg)
		if ansi.StringWidth(line) > width {
			line = ansi.Truncate(line, width, "")
		}
		if pad := width - ansi.StringWidth(line); pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		lines[i] = bg + line
	}

	return strings.Join(lines, "\n") + "\x1b[0m"
}

func ANSIBackground(c lipgloss.Color) string {
	return ansiColor(c, true)
}

func ANSIForeground(c lipgloss.Color) string {
	return ansiColor(c, false)
}

func ansiColor(c lipgloss.Color, background bool) string {
	s := strings.TrimPrefix(string(c), "#")
	if len(s) != 6 {
		return ""
	}
	r, rErr := strconv.ParseUint(s[0:2], 16, 8)
	g, gErr := strconv.ParseUint(s[2:4], 16, 8)
	b, bErr := strconv.ParseUint(s[4:6], 16, 8)
	if rErr != nil || gErr != nil || bErr != nil {
		return ""
	}
	mode := 38
	if background {
		mode = 48
	}
	return fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", mode, r, g, b)
}
