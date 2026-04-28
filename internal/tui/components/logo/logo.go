// Package logo renders the Sparkle wordmark as a 5×7 block pixel-art font.
//
// Each logical pixel is two full-block characters wide ("██") so pixels appear
// square in any terminal, and full-block cells have no descender/ascender gap,
// so rows sit flush against each other regardless of font or line-height.
//
// The 5-column, 7-row grid produces tall, bold letterforms similar to the
// style used in charmbracelet/crush — more visual weight than the older 4×5 font.
package logo

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/tui/theme"
)

// font5x7 encodes each letter as 7 rows × 5 columns.
// Each uint8 uses bits 4..0 (bit 4 = leftmost column).
var font5x7 = map[rune][7]uint8{
	// S: curved with top/bottom bars and a diagonal step
	'S': {0b01111, 0b10000, 0b10000, 0b01110, 0b00001, 0b00001, 0b11110},
	// P: top half filled box, stem goes straight down
	'P': {0b11110, 0b10001, 0b10001, 0b11110, 0b10000, 0b10000, 0b10000},
	// A: peaked arch, crossbar on row 3
	'A': {0b01110, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	// R: P-shape with a diagonal leg
	'R': {0b11110, 0b10001, 0b10001, 0b11110, 0b10100, 0b10010, 0b10001},
	// K: two diagonals branching from a vertical stem
	'K': {0b10001, 0b10010, 0b10100, 0b11000, 0b10100, 0b10010, 0b10001},
	// L: vertical stem with a full-width foot
	'L': {0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b11111},
	// E: three horizontal bars (top, mid, bottom) with a stem
	'E': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b11111},
}

const (
	pixelOn  = "██"
	pixelOff = "  "
	// letterGap is a single space between letters (1 terminal column), keeping
	// the logo compact enough to render on 80-column terminals.
	letterGap = " "
	// blockWidth: 7 letters × (5 cols × 2 chars) + 6 gaps × 1 char.
	blockWidth = 7*10 + 6*1 // 76 terminal columns
)

// renderWord returns 7 rows of block pixel art for the given word.
// Unknown runes are silently skipped.
func renderWord(word string) []string {
	rows := [7]strings.Builder{}
	first := true
	for _, ch := range word {
		bits, ok := font5x7[ch]
		if !ok {
			continue
		}
		if !first {
			for i := range rows {
				rows[i].WriteString(letterGap)
			}
		}
		first = false
		for r := 0; r < 7; r++ {
			for col := 4; col >= 0; col-- {
				if bits[r]&(1<<uint(col)) != 0 {
					rows[r].WriteString(pixelOn)
				} else {
					rows[r].WriteString(pixelOff)
				}
			}
		}
	}
	out := make([]string, 7)
	for i, sb := range rows {
		out[i] = sb.String()
	}
	return out
}

// Render produces the 7-row block pixel wordmark with a per-row gradient.
// Falls back to Compact for widths narrower than the art.
func Render(t theme.Theme, width int) string {
	if width < blockWidth {
		return Compact(t)
	}
	rawRows := renderWord("SPARKLE")
	styled := make([]string, len(rawRows))
	for i, row := range rawRows {
		styled[i] = theme.ApplyGradOn(row, t.GradientFrom, t.GradientTo, t.Background, true)
	}
	block := strings.Join(styled, "\n")
	sub := theme.Fg(t, t.Muted).Render("by viphase")
	return lipgloss.JoinVertical(lipgloss.Left, block, sub)
}

// Compact returns a single-line gradient "ꕤ SPARKLE" wordmark for narrow views.
func Compact(t theme.Theme) string {
	glyph := theme.Fg(t, t.Primary).Render("ꕤ")
	title := theme.ApplyGradOn("SPARKLE", t.GradientFrom, t.GradientTo, t.Background, true)
	return lipgloss.JoinHorizontal(lipgloss.Left, glyph, " ", title)
}
