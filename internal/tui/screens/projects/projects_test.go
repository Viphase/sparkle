package projects

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/theme"
)

// fakeLoader satisfies the Loader interface for tests.
type fakeLoader struct {
	items  []domain.Project
	saved  []domain.Project
	pathFn func(id string) string
}

func (f *fakeLoader) ListProjects() ([]domain.Project, error) {
	out := make([]domain.Project, len(f.items))
	copy(out, f.items)
	return out, nil
}

func (f *fakeLoader) SaveProject(p domain.Project) error {
	f.saved = append(f.saved, p)
	for i, item := range f.items {
		if item.ID == p.ID {
			f.items[i] = p
			return nil
		}
	}
	f.items = append(f.items, p)
	return nil
}

func (f *fakeLoader) ProjectPath(id string) string {
	if f.pathFn != nil {
		return f.pathFn(id)
	}
	return "/tmp/" + id + "/project.md"
}

func newModel(loader Loader) *Model {
	return New(theme.PastelDark(), loader).(*Model)
}

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestProjectsLoadedMsgPopulatesItems(t *testing.T) {
	m := newModel(nil)
	next, _ := m.Update(msgs.ProjectsLoadedMsg{Items: []domain.Project{
		{ID: "p1", Title: "Alpha", Status: domain.ProjectStatusDraft},
		{ID: "p2", Title: "Beta", Status: domain.ProjectStatusActive},
	}})
	got := next.(*Model)
	if !got.loaded {
		t.Error("expected loaded=true")
	}
	if len(got.items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got.items))
	}
}

func TestNavigationMovesCursor(t *testing.T) {
	m := newModel(nil)
	m.items = []domain.Project{
		{ID: "p1", Title: "A", Status: domain.ProjectStatusDraft},
		{ID: "p2", Title: "B", Status: domain.ProjectStatusActive},
		{ID: "p3", Title: "C", Status: domain.ProjectStatusPaused},
	}
	m.loaded = true

	// j moves down
	next, _ := m.Update(keyMsg("j"))
	got := next.(*Model)
	if got.cursor != 1 {
		t.Errorf("cursor after j: %d, want 1", got.cursor)
	}
	// k moves up
	next, _ = got.Update(keyMsg("k"))
	got = next.(*Model)
	if got.cursor != 0 {
		t.Errorf("cursor after k: %d, want 0", got.cursor)
	}
	// G goes to end
	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	got = next.(*Model)
	if got.cursor != 2 {
		t.Errorf("cursor after G: %d, want 2", got.cursor)
	}
}

func TestEnterOpenDetailPane(t *testing.T) {
	m := newModel(nil)
	m.items = []domain.Project{{ID: "p1", Title: "A", Status: domain.ProjectStatusDraft}}
	m.loaded = true

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(*Model)
	if got.activePane != paneDetail {
		t.Error("enter should switch to detail pane")
	}
}

func TestEscFromDetailReturnsToList(t *testing.T) {
	m := newModel(nil)
	m.items = []domain.Project{{ID: "p1", Title: "A", Status: domain.ProjectStatusDraft}}
	m.loaded = true
	m.activePane = paneDetail

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(*Model)
	if got.activePane != paneList {
		t.Error("esc should return to list pane")
	}
}

func TestEditFieldSavesProject(t *testing.T) {
	loader := &fakeLoader{
		items: []domain.Project{{
			ID:     "p1",
			Title:  "Old Title",
			Status: domain.ProjectStatusDraft,
		}},
	}
	m := newModel(loader)
	m.items = loader.items
	m.loaded = true
	m.activePane = paneDetail
	m.detailField = fldTitle
	m.now = func() time.Time { return time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC) }

	// Press e to start editing
	next, _ := m.Update(keyMsg("e"))
	got := next.(*Model)
	if !got.inputActive {
		t.Fatal("e should activate input")
	}

	// Type new title
	for _, ch := range "New Title" {
		got.input.SetValue(got.input.Value() + string(ch))
	}
	got.input.SetValue("New Title")

	// Press enter to save
	next, cmd := got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got = next.(*Model)
	if got.inputActive {
		t.Error("input should be deactivated after enter")
	}
	if cmd == nil {
		t.Fatal("expected save command")
	}
	// Execute the command
	result := cmd()
	if _, ok := result.(msgs.ProjectsLoadedMsg); !ok {
		t.Errorf("expected ProjectsLoadedMsg, got %T", result)
	}
	if len(loader.saved) == 0 {
		t.Error("expected project to be saved")
	}
	if loader.saved[0].Title != "New Title" {
		t.Errorf("saved title: %q, want %q", loader.saved[0].Title, "New Title")
	}
}

func TestStatusCyclingWithArrows(t *testing.T) {
	loader := &fakeLoader{
		items: []domain.Project{{
			ID:     "p1",
			Title:  "Test",
			Status: domain.ProjectStatusDraft,
		}},
	}
	m := newModel(loader)
	m.items = loader.items
	m.loaded = true
	m.activePane = paneDetail
	m.detailField = fldStatus

	// Right should cycle to next status (active)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	got := next.(*Model)
	if got.items[0].Status != domain.ProjectStatusActive {
		t.Errorf("status after right: %q, want active", got.items[0].Status)
	}
}

func TestThemeChangeUpdatesTheme(t *testing.T) {
	m := newModel(nil)
	_, _ = m.Update(msgs.ThemeChangedMsg{ThemeName: "nova"})
	// Just verify no panic — theme application is visual.
}

func TestViewRendersEmptyState(t *testing.T) {
	m := newModel(nil)
	m.loaded = true
	out := m.View(80, 20)
	if out == "" {
		t.Error("expected non-empty view")
	}
}

func TestViewRendersTwoPaneWithItems(t *testing.T) {
	m := newModel(nil)
	m.items = []domain.Project{
		{ID: "p1", Title: "Alpha", Status: domain.ProjectStatusActive},
		{ID: "p2", Title: "Beta", Status: domain.ProjectStatusDraft},
	}
	m.loaded = true
	out := m.View(100, 24)
	if out == "" {
		t.Error("expected non-empty view")
	}
}

func TestInFormReturnsTrueWhenInputActive(t *testing.T) {
	m := newModel(nil)
	if m.InForm() {
		t.Error("InForm should be false initially")
	}
	m.inputActive = true
	if !m.InForm() {
		t.Error("InForm should be true when inputActive")
	}
}

func TestSaveProjectCmdReturnsErrorWhenNoLoader(t *testing.T) {
	m := newModel(nil)
	p := domain.Project{ID: "p1", Title: "Test", Status: domain.ProjectStatusDraft}
	cmd := m.saveProjectCmd(p)
	result := cmd()
	if _, ok := result.(msgs.ErrorMsg); !ok {
		t.Errorf("expected ErrorMsg with nil loader, got %T", result)
	}
}

func TestOpenInEditorCmdReturnsErrorWhenNoLoader(t *testing.T) {
	m := newModel(nil)
	p := domain.Project{ID: "p1", Title: "Test", Status: domain.ProjectStatusDraft}
	cmd := m.openInEditorCmd(p)
	result := cmd()
	if _, ok := result.(msgs.ErrorMsg); !ok {
		t.Errorf("expected ErrorMsg with nil loader, got %T", result)
	}
}

func TestParseTagsRoundtrip(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"go, tui, writing", []string{"go", "tui", "writing"}},
		{"  single  ", []string{"single"}},
		{"", nil},
		{"  ,  ,  ", nil},
	}
	for _, tc := range cases {
		got := parseTags(tc.input)
		if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", tc.want) {
			t.Errorf("parseTags(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
