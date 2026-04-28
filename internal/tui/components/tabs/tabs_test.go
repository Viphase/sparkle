package tabs

import (
	"strings"
	"testing"

	"github.com/viphase/sparkle/internal/tui/theme"
)

func TestRenderUsesExplicitActiveTabBackground(t *testing.T) {
	tm := theme.PastelDark()
	out := Render(tm, 80, 1, []string{"AI", "Sparks", "Dashboard"})

	if !strings.Contains(out, theme.ANSIBackground(tm.Primary)) {
		t.Fatalf("active tab should include explicit primary background: %q", out)
	}
}
