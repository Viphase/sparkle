package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/workspace"
)

func newTestRoot() Root {
	return NewRoot(workspace.Workspace{}, nil)
}

func TestRootInitialRouteIsDashboard(t *testing.T) {
	r := newTestRoot()
	if r.route != RouteDashboard {
		t.Errorf("initial route = %v, want Dashboard", r.route)
	}
}

func TestRootTabSwitchesRoute(t *testing.T) {
	r := newTestRoot()
	next, _ := r.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := next.(Root).route
	if got != RouteSparks {
		t.Errorf("after tab: route = %v, want Sparks", got)
	}
}

func TestRootShiftTabGoesBack(t *testing.T) {
	r := newTestRoot()
	next, _ := r.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	got := next.(Root).route
	if got != RouteSettings {
		t.Errorf("after shift+tab from Dashboard: route = %v, want Settings (wrap)", got)
	}
}

func TestRootNumberJumps(t *testing.T) {
	r := newTestRoot()
	next, _ := r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	got := next.(Root).route
	if got != RouteTracker {
		t.Errorf("after '4': route = %v, want Tracker", got)
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

// When sparks is in form-mode the global key handler must yield 'q' to the
// input so the user can type a 'q' in their spark title.
func TestRootGlobalKeysSuppressedInSparkForm(t *testing.T) {
	r := newTestRoot()
	// Switch to sparks tab.
	tabbed, _ := r.Update(tea.KeyMsg{Type: tea.KeyTab})
	r = tabbed.(Root)
	// Press 'n' to open the form.
	opened, _ := r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	r = opened.(Root)
	// Press 'q' — should NOT quit.
	next, cmd := r.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		// cmd may be the textinput's Blink cmd, which is fine, just must not be Quit.
		if _, isQuit := cmd().(tea.QuitMsg); isQuit {
			t.Error("'q' inside spark form should not quit")
		}
	}
	if next.(Root).route != RouteSparks {
		t.Errorf("route should still be Sparks; got %v", next.(Root).route)
	}
}
