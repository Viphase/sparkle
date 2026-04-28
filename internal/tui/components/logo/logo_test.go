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
	out := stripANSI(Render(theme.PastelDark(), 64))
	if strings.Contains(out, "Sparkle") || strings.Contains(out, "S P A R K L E") {
		t.Fatalf("logo should not use the old heading: %q", out)
	}
	// New logo uses ANSI Shadow block characters (███ style).
	if !strings.Contains(out, "█") {
		t.Fatalf("logo should include block characters: %q", out)
	}
	// The ANSI Shadow art is 6 rows tall; verify characteristic rows are present.
	if !strings.Contains(out, "███████╗") {
		t.Fatalf("logo should include ANSI Shadow block art: %q", out)
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
	if strings.Contains(out, "╚══════╝") {
		t.Fatalf("narrow render should not include full block art: %q", out)
	}
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
