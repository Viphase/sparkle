package sparks

import (
	"fmt"
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
	for i, existing := range f.saved {
		if existing.ID == s.ID {
			f.saved[i] = s
			return nil
		}
	}
	f.saved = append(f.saved, s)
	return nil
}

func (f *fakeSaver) LoadSpark(id string) (domain.Spark, error) {
	for _, s := range f.saved {
		if s.ID == id {
			return s, nil
		}
	}
	return domain.Spark{}, fmt.Errorf("not found: %s", id)
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

func TestNKeyOpensFormForCreate(t *testing.T) {
	m := newModel(nil)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := next.(*Model)
	if got.mode != modeForm {
		t.Errorf("expected modeForm after 'n', got %v", got.mode)
	}
	if got.editingID != "" {
		t.Errorf("editingID should be empty for new; got %q", got.editingID)
	}
	if got.input.Value() != "" {
		t.Errorf("input should be empty for new; got %q", got.input.Value())
	}
}

func TestEKeyOpensFormPrefilled(t *testing.T) {
	saver := &fakeSaver{}
	saver.saved = []domain.Spark{
		{ID: "x", Title: "Existing", Status: domain.SparkStatusNew},
	}
	m := newModel(saver)
	next, _ := m.Update(msgs.SparksLoadedMsg{Items: saver.saved})
	m = next.(*Model)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	got := next.(*Model)
	if got.mode != modeForm {
		t.Errorf("expected modeForm after 'e', got %v", got.mode)
	}
	if got.editingID != "x" {
		t.Errorf("editingID = %q, want x", got.editingID)
	}
	if got.input.Value() != "Existing" {
		t.Errorf("input = %q, want Existing", got.input.Value())
	}
}

func TestEKeyNoopWhenEmpty(t *testing.T) {
	m := newModel(nil)
	next, _ := m.Update(msgs.SparksLoadedMsg{Items: nil})
	m = next.(*Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	got := next.(*Model)
	if got.mode != modeList {
		t.Errorf("'e' on empty list should stay in list mode; got %v", got.mode)
	}
}

func TestEscClosesFormAndClearsEditingID(t *testing.T) {
	saver := &fakeSaver{}
	saver.saved = []domain.Spark{{ID: "x", Title: "T", Status: domain.SparkStatusNew}}
	m := newModel(saver)
	next, _ := m.Update(msgs.SparksLoadedMsg{Items: saver.saved})
	m = next.(*Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = next.(*Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(*Model)
	if got.mode != modeList {
		t.Errorf("esc should return to list; got %v", got.mode)
	}
	if got.editingID != "" {
		t.Errorf("editingID should be cleared; got %q", got.editingID)
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

func TestEnterCreatesSpark(t *testing.T) {
	saver := &fakeSaver{}
	m := newModel(saver)
	m.now = func() time.Time { return time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC) }

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = next.(*Model)
	for _, r := range "Hello world" {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(*Model)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*Model)
	if cmd == nil {
		t.Fatal("expected save cmd")
	}
	loaded, ok := cmd().(msgs.SparksLoadedMsg)
	if !ok {
		t.Fatalf("expected SparksLoadedMsg")
	}
	if len(loaded.Items) != 1 {
		t.Fatalf("got %d items", len(loaded.Items))
	}
	if loaded.Items[0].Title != "Hello world" {
		t.Errorf("title: %q", loaded.Items[0].Title)
	}
	if loaded.Items[0].Status != domain.SparkStatusNew {
		t.Errorf("status: %q", loaded.Items[0].Status)
	}
	if !strings.HasPrefix(loaded.Items[0].ID, "spark_20260427_120000_") {
		t.Errorf("unexpected id: %s", loaded.Items[0].ID)
	}
}

func TestEditPreservesIDAndDescription(t *testing.T) {
	saver := &fakeSaver{}
	saver.saved = []domain.Spark{
		{
			ID:          "x",
			Title:       "Old",
			Description: "long body that must survive",
			Status:      domain.SparkStatusNew,
			CreatedAt:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	m := newModel(saver)
	m.now = func() time.Time { return time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC) }

	next, _ := m.Update(msgs.SparksLoadedMsg{Items: saver.saved})
	m = next.(*Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = next.(*Model)
	m.input.SetValue("Renamed")

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*Model)
	if cmd == nil {
		t.Fatal("expected save cmd")
	}
	loaded := cmd().(msgs.SparksLoadedMsg)
	if len(loaded.Items) != 1 {
		t.Fatalf("got %d items", len(loaded.Items))
	}
	got := loaded.Items[0]
	if got.ID != "x" {
		t.Errorf("ID changed: %q", got.ID)
	}
	if got.Title != "Renamed" {
		t.Errorf("title: %q", got.Title)
	}
	if got.Description != "long body that must survive" {
		t.Errorf("description lost: %q", got.Description)
	}
	if !got.UpdatedAt.Equal(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("UpdatedAt not bumped: %v", got.UpdatedAt)
	}
	if !got.CreatedAt.Equal(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("CreatedAt mutated: %v", got.CreatedAt)
	}
}

func TestArchiveTogglesStatus(t *testing.T) {
	saver := &fakeSaver{}
	saver.saved = []domain.Spark{{ID: "x", Title: "T", Status: domain.SparkStatusNew}}
	m := newModel(saver)
	m.now = func() time.Time { return time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC) }

	next, _ := m.Update(msgs.SparksLoadedMsg{Items: saver.saved})
	m = next.(*Model)

	// First 'a' archives.
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = next.(*Model)
	if cmd == nil {
		t.Fatal("expected archive cmd")
	}
	loaded := cmd().(msgs.SparksLoadedMsg)
	if loaded.Items[0].Status != domain.SparkStatusArchived {
		t.Errorf("expected archived, got %q", loaded.Items[0].Status)
	}

	// Re-feed the new state, then 'a' again to unarchive.
	next, _ = m.Update(loaded)
	m = next.(*Model)
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = next.(*Model)
	if cmd == nil {
		t.Fatal("expected unarchive cmd")
	}
	loaded = cmd().(msgs.SparksLoadedMsg)
	if loaded.Items[0].Status != domain.SparkStatusNew {
		t.Errorf("expected unarchived (new), got %q", loaded.Items[0].Status)
	}
}

func TestArchiveNoopWhenEmpty(t *testing.T) {
	saver := &fakeSaver{}
	m := newModel(saver)
	next, _ := m.Update(msgs.SparksLoadedMsg{Items: nil})
	m = next.(*Model)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("'a' on empty list should be a no-op")
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
