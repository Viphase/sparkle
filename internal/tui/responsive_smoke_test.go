package tui

import (
	"strings"
	"testing"

	"github.com/viphase/sparkle/internal/config"
	"github.com/viphase/sparkle/internal/storage/markdown"
	"github.com/viphase/sparkle/internal/workspace"
)

// M12 / v2-cut requirement: the same app must render coherently on
// 60×20, 100×30, 120×36, and 220×60 terminals — no panic, no truncation
// to the "minimum" placeholder, and a non-trivial frame at every size.
//
// This is a smoke test: we render the root View at each size and assert
// the output is non-empty and lacks the "minimum 50x16" placeholder.
func TestRoot_RendersAtAllRequiredSizes(t *testing.T) {
	tmp := t.TempDir()
	ws, err := workspace.Open(tmp)
	if err != nil {
		t.Fatalf("open ws: %v", err)
	}
	store := markdown.NewStore(ws.Root)

	r := NewRootWithConfig(ws, store, config.Defaults())

	sizes := []struct{ w, h int }{
		{60, 20},
		{100, 30},
		{120, 36},
		{220, 60},
	}
	for _, sz := range sizes {
		r.width = sz.w
		r.height = sz.h
		out := r.View()
		if out == "" {
			t.Fatalf("size %dx%d: empty View()", sz.w, sz.h)
		}
		if strings.Contains(out, "minimum") && strings.Contains(out, "50x16") {
			t.Fatalf("size %dx%d: showed minimum placeholder when it should render full UI:\n%s",
				sz.w, sz.h, out)
		}
	}
}

// At sub-minimum sizes the app must not crash; it should show the
// graceful "current WxH · minimum 50x16" placeholder instead.
func TestRoot_GracefullyDegradesBelowMinimum(t *testing.T) {
	tmp := t.TempDir()
	ws, err := workspace.Open(tmp)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	store := markdown.NewStore(ws.Root)
	r := NewRootWithConfig(ws, store, config.Defaults())
	r.width = 38
	r.height = 12
	out := r.View()
	if out == "" {
		t.Fatalf("expected placeholder, got empty")
	}
}
