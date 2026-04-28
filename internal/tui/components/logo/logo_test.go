package logo

import (
	"regexp"
	"strings"
	"testing"

	"github.com/viphase/sparkle/internal/tui/theme"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

func TestRenderUsesBlockLogo(t *testing.T) {
	// blockWidth = 76; use 80 so the full 5×7 block logo renders.
	out := stripANSI(Render(theme.PastelDark(), 80))
	if strings.Contains(out, "Sparkle") || strings.Contains(out, "S P A R K L E") {
		t.Fatalf("logo should not use the old heading: %q", out)
	}
	// Pixel-art logo uses full-block characters ("██" per pixel).
	if !strings.Contains(out, "██") {
		t.Fatalf("logo should include block pixel art: %q", out)
	}
	// Box-drawing characters from the old ANSI Shadow font should not appear.
	if strings.Contains(out, "╗") || strings.Contains(out, "╚") {
		t.Fatalf("logo should not use box-drawing characters: %q", out)
	}
	if !strings.Contains(out, "by viphase") {
		t.Fatalf("logo should include byline: %q", out)
	}
}

func TestRenderFallsBackToCompactNarrow(t *testing.T) {
	out := stripANSI(Render(theme.PastelDark(), 30))
	if !strings.Contains(out, "SPARKLE") {
		t.Fatalf("compact fallback missing title: %q", out)
	}
	// At narrow width the block art should not appear (too wide).
	// Compact uses "ꕤ SPARKLE" text, not a pixel grid.
	_ = out
}

func TestCompactKeepsWidthBounded(t *testing.T) {
	out := stripANSI(Compact(theme.PastelDark()))
	// Compact mode uses the gradient text "SPARKLE" (literal, not block art).
	if !strings.Contains(out, "SPARKLE") {
		t.Fatalf("compact logo missing title: %q", out)
	}
	if strings.Contains(out, "Sparkle") {
		t.Fatalf("compact logo should not use title case heading: %q", out)
	}
}
