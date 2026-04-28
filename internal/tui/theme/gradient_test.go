package theme

import (
	"regexp"
	"strings"
	"testing"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

func TestApplyGradPreservesText(t *testing.T) {
	in := "Sparkle"
	out := ApplyGrad(in, "#a78bfa", "#f0abfc", true)
	if got := stripANSI(out); got != in {
		t.Errorf("text changed: got %q, want %q", got, in)
	}
}

func TestApplyGradEmpty(t *testing.T) {
	if out := ApplyGrad("", "#aabbcc", "#112233", false); out != "" {
		t.Errorf("empty input should return empty string, got %q", out)
	}
}

func TestApplyGradInvalidColorFallsBack(t *testing.T) {
	in := "Hello"
	out := ApplyGrad(in, "not-a-color", "#112233", false)
	if got := stripANSI(out); got != in {
		t.Errorf("plain fallback should preserve text: got %q, want %q", got, in)
	}
}

func TestApplyGradHandlesMultiByte(t *testing.T) {
	in := "✹ Sparkle ꕤ"
	out := ApplyGrad(in, "#a78bfa", "#f0abfc", false)
	if !strings.Contains(stripANSI(out), "Sparkle") {
		t.Errorf("multi-byte input lost: %q", stripANSI(out))
	}
}

func TestPaintBackgroundPadsAndReappliesAfterReset(t *testing.T) {
	tm := PastelDark()
	out := PaintBackground(tm, 8, 2, "hi\x1b[0m")

	lines := strings.Split(strings.TrimSuffix(out, "\x1b[0m"), "\n")
	if len(lines) != 2 {
		t.Fatalf("line count=%d, want 2", len(lines))
	}
	if got := stripANSI(lines[0]); got != "hi      " {
		t.Fatalf("first line=%q, want padded width", got)
	}
	if got := stripANSI(lines[1]); got != "        " {
		t.Fatalf("second line=%q, want painted blank line", got)
	}
	if !strings.Contains(out, "\x1b[0m"+ANSIBackground(tm.Background)) {
		t.Fatalf("background should be re-applied after reset: %q", out)
	}
}
