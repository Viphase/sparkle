package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/config"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/workspace"
)

func newTestRoot() Root {
	return NewRoot(workspace.Workspace{}, nil)
}

func TestRootInitialRouteIsPulse(t *testing.T) {
	r := newTestRoot()
	if r.route != RoutePulse {
		t.Errorf("initial route = %v, want Pulse (prime tab)", r.route)
	}
}

func TestRootUsesConfiguredTheme(t *testing.T) {
	r := NewRootWithConfig(workspace.Workspace{}, nil, config.Config{Theme: "nova", WordsThreshold: 10})
	if r.theme.Name != "nova" {
		t.Fatalf("theme=%q, want nova", r.theme.Name)
	}
}

func TestRootTabSwitchesRoute(t *testing.T) {
	r := newTestRoot()
	// Initial route is Pulse (prime); tab should advance to Workspace.
	next, _ := r.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := next.(Root).route
	if got != RouteWorkspace {
		t.Errorf("after tab from Pulse: route = %v, want Workspace", got)
	}
}

func TestRootShiftTabGoesBack(t *testing.T) {
	r := newTestRoot()
	// Initial route is Pulse (prime); shift+tab wraps to Settings (last tab).
	next, _ := r.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	got := next.(Root).route
	if got != RouteSettings {
		t.Errorf("after shift+tab from Pulse: route = %v, want Settings", got)
	}
}

func TestRootNumberJumps(t *testing.T) {
	r := newTestRoot()
	// Tab order: 1=Pulse, 2=Workspace, 3=Settings.
	next, _ := r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	got := next.(Root).route
	if got != RouteWorkspace {
		t.Errorf("after '2': route = %v, want Workspace", got)
	}
}

func TestRootErrorMsgFlowsToStatusBar(t *testing.T) {
	r := newTestRoot()
	next, _ := r.Update(msgs.ErrorMsg{Source: "load-workspace", Err: errors.New("boom")})
	view := next.(Root).status.View(80)
	if !strings.Contains(view, "load-workspace") || !strings.Contains(view, "boom") {
		t.Errorf("status bar should show error envelope; got %q", view)
	}
}

func TestRootQuitKey(t *testing.T) {
	r := newTestRoot()
	_, cmd := r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected tea.Quit cmd, got nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg")
	}
}

func TestRootAPIKeyChangedUpdatesStatus(t *testing.T) {
	r := newTestRoot()
	next, _ := r.Update(msgs.APIKeyChangedMsg{Key: "sk-ant-test", Model: "claude-sonnet-4-6"})
	view := next.(Root).status.View(80)
	if !strings.Contains(view, "claude") {
		t.Errorf("status bar should show 'claude' after API key set; got %q", view)
	}
}

func TestRootSparkPromotedStaysOnWorkspace(t *testing.T) {
	r := newTestRoot()
	// Start on pulse.
	r.route = RoutePulse
	next, _ := r.Update(msgs.SparkPromotedMsg{})
	if next.(Root).route != RouteWorkspace {
		t.Errorf("after SparkPromotedMsg: route = %v, want Workspace", next.(Root).route)
	}
}
