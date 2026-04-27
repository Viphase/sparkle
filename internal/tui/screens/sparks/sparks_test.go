package sparks

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/theme"
)

type fakeSaver struct {
	saved []domain.Spark
}

func (f *fakeSaver) SaveSpark(s domain.Spark) error {
	f.saved = append(f.saved, s)
	return nil
}

func (f *fakeSaver) ListSparks() ([]domain.Spark, error) {
	out := make([]domain.Spark, len(f.saved))
	copy(out, f.saved)
	return out, nil
}

func newModel(saver Saver) *Model {
	return New(theme.PastelDark(), saver).(*Model)
}

func TestLoadedMsgPopulatesItems(t *testing.T) {
	m := newModel(nil)
	next, _ := m.Update(msgs.SparksLoadedMsg{Items: []domain.Spark{
		{ID: "1", Title: "First", Status: domain.SparkStatusNew},
		{ID: "2", Title: "Second", Status: domain.SparkStatusNew},
	}})
	got := next.(*Model)
	if !got.loaded {
		t.Error("expected loaded=true")
	}
	if len(got.items) != 2 {
		t.Errorf("got %d items, want 2", len(got.items))
	}
}

func TestCursorMovesWithJK(t *testing.T) {
	m := newModel(nil)
	loaded, _ := m.Update(msgs.SparksLoadedMsg{Items: []domain.Spark{
		{ID: "1", Status: domain.SparkStatusNew},
		{ID: "2", Status: domain.SparkStatusNew},
		{ID: "3", Status: domain.SparkStatusNew},
	}})
	m = loaded.(*Model)

	for _, key := range []rune{'j', 'j'} {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		m = next.(*Model)
	}
	if m.cursor != 2 {
		t.Errorf("after jj: cursor=%d, want 2", m.cursor)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = next.(*Model)
	if m.cursor != 2 {
		t.Errorf("cursor should clamp; got %d", m.cursor)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = next.(*Model)
	if m.cursor != 1 {
		t.Errorf("after k: cursor=%d, want 1", m.cursor)
	}
}

func TestNKeyOpensForm(t *testing.T) {
	m := newModel(nil)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := next.(*Model)
	if got.mode != modeForm {
		t.Errorf("expected modeForm after 'n', got %v", got.mode)
	}
}

func TestEscClosesForm(t *testing.T) {
	m := newModel(nil)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = next.(*Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(*Model)
	if m.mode != modeList {
		t.Errorf("esc should return to list mode; got %v", m.mode)
	}
}

func TestEnterOnEmptyFormDoesNotSave(t *testing.T) {
	saver := &fakeSaver{}
	m := newModel(saver)
	m.now = func() time.Time { return time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC) }

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = next.(*Model)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*Model)
	if cmd != nil {
		t.Errorf("empty title should not produce a save cmd")
	}
	if m.mode != modeForm {
		t.Errorf("empty title should keep form open; got mode=%v", m.mode)
	}
	if len(saver.saved) != 0 {
		t.Errorf("nothing should have been saved; got %v", saver.saved)
	}
}

func TestEnterPersistsSparkAndRefreshes(t *testing.T) {
	saver := &fakeSaver{}
	m := newModel(saver)
	m.now = func() time.Time { return time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC) }

	// Open form, type, submit.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = next.(*Model)
	for _, r := range "Hello world" {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(*Model)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*Model)
	if cmd == nil {
		t.Fatal("expected save cmd from filled form")
	}

	msg := cmd()
	loaded, ok := msg.(msgs.SparksLoadedMsg)
	if !ok {
		t.Fatalf("expected SparksLoadedMsg, got %T", msg)
	}
	if len(loaded.Items) != 1 {
		t.Fatalf("expected 1 item after save, got %d", len(loaded.Items))
	}
	if loaded.Items[0].Title != "Hello world" {
		t.Errorf("title: %q", loaded.Items[0].Title)
	}
	if loaded.Items[0].Status != domain.SparkStatusNew {
		t.Errorf("status should be new, got %q", loaded.Items[0].Status)
	}
	if !strings.HasPrefix(loaded.Items[0].ID, "spark_20260427_120000_") {
		t.Errorf("unexpected id: %s", loaded.Items[0].ID)
	}
}

func TestEmptyStateShownBeforeLoad(t *testing.T) {
	m := newModel(nil)
	out := m.View(60, 20)
	if !strings.Contains(stripANSI(out), "Loading") {
		t.Errorf("expected loading state; got: %s", stripANSI(out))
	}
}

func TestEmptyStateAfterLoadWithNoItems(t *testing.T) {
	m := newModel(nil)
	next, _ := m.Update(msgs.SparksLoadedMsg{Items: nil})
	got := next.(*Model)
	out := got.View(60, 20)
	if !strings.Contains(stripANSI(out), "No sparks yet") {
		t.Errorf("expected empty-state copy; got: %s", stripANSI(out))
	}
}

// stripANSI removes ANSI escape sequences so tests can match content.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
