package workspace

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/theme"
)

func TestWorkspaceRailShowsSparks(t *testing.T) {
	m := New(theme.PastelDark(), "")
	next, _ := m.Update(msgs.SparksLoadedMsg{Items: []domain.Spark{
		{ID: "spark_one", Title: "Idea Alpha", Status: domain.SparkStatusNew},
		{ID: "spark_two", Title: "Idea Beta", Status: domain.SparkStatusNew},
	}})

	if len(next.items) != 2 {
		t.Fatalf("expected 2 rail items, got %d", len(next.items))
	}
	if next.items[0].title != "Idea Alpha" {
		t.Errorf("first item title = %q, want Idea Alpha", next.items[0].title)
	}
	view := next.View(120, 30)
	if !strings.Contains(view, "Idea Alpha") {
		t.Error("view should contain spark title 'Idea Alpha'")
	}
}

func TestWorkspaceRailShowsProjects(t *testing.T) {
	m := New(theme.PastelDark(), "")
	next, _ := m.Update(msgs.ProjectsLoadedMsg{Items: []domain.Project{
		{ID: "project_one", Title: "Sparkle", Status: domain.ProjectStatusActive},
	}})

	if len(next.items) != 1 {
		t.Fatalf("expected 1 rail item, got %d", len(next.items))
	}
	if next.items[0].kind != "project" {
		t.Errorf("expected kind=project, got %q", next.items[0].kind)
	}
}

func TestWorkspaceAIPanelHiddenByDefault(t *testing.T) {
	m := New(theme.PastelDark(), "")
	if m.aiVisible {
		t.Error("AI panel should be hidden by default")
	}
}

func TestWorkspaceAIPanelTogglesWithI(t *testing.T) {
	m := New(theme.PastelDark(), "")
	m.items = []railItem{{kind: "spark", id: "s1", title: "Test Spark"}}

	// Update mutates m in-place (pointer receiver) — check m directly.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	if !m.aiVisible {
		t.Error("AI panel should be visible after pressing 'i'")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	if m.aiVisible {
		t.Error("AI panel should be hidden after pressing 'i' again")
	}
}

func TestWorkspaceBreakpoints(t *testing.T) {
	m := New(theme.PastelDark(), "")
	m.items = []railItem{{kind: "spark", id: "s1", title: "T"}}

	for _, w := range []int{50, 80, 120, 200} {
		view := m.View(w, 24)
		if view == "" {
			t.Errorf("View(%d, 24) returned empty string", w)
		}
	}
}

func TestWorkspaceEmptyStateHasCTA(t *testing.T) {
	m := New(theme.PastelDark(), "")
	view := m.View(120, 30)
	if !strings.Contains(view, "no items") && !strings.Contains(view, "select") {
		t.Error("empty workspace should show empty state or select message")
	}
}

func TestWorkspaceDetailLoadedMsgUpdatesBody(t *testing.T) {
	m := New(theme.PastelDark(), "")
	m.items = []railItem{{kind: "project", id: "proj_01", title: "Demo"}}

	next, _ := m.Update(detailLoadedMsg{
		title:  "●  Demo",
		body:   "Description\n─────────────\nA great project.",
		itemID: "proj_01",
		kind:   "project",
	})
	if next.detailTitle != "●  Demo" {
		t.Errorf("detailTitle = %q, want '●  Demo'", next.detailTitle)
	}
	if !strings.Contains(next.detailBody, "great project") {
		t.Errorf("detailBody = %q, should contain 'great project'", next.detailBody)
	}
	view := next.View(120, 30)
	if !strings.Contains(view, "Demo") {
		t.Error("view should contain the project title")
	}
}

func TestWorkspaceInlineEditToggle(t *testing.T) {
	m := New(theme.PastelDark(), "")
	m.detailID = "proj_01"
	m.detailKind = "project"
	m.detailTitle = "●  Demo"
	m.detailBody = "Some body text."

	// Press 'e' — enters edit mode (mutates m in-place via pointer receiver).
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if !m.editing {
		t.Error("expected editing=true after pressing 'e'")
	}
	if !m.InForm() {
		t.Error("InForm() should return true while editing")
	}

	// Press Esc — cancel and revert body.
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.editing {
		t.Error("expected editing=false after esc")
	}
	if m.detailBody != "Some body text." {
		t.Errorf("body after cancel = %q, want 'Some body text.'", m.detailBody)
	}
}

func TestWorkspaceDetailScrollKeys(t *testing.T) {
	m := New(theme.PastelDark(), "")
	m.detailID = "s1"
	m.detailKind = "spark"
	m.detailTitle = "✦  Test"
	m.detailBody = strings.Repeat("line\n", 50)

	// Press J — detailScroll should increase.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("J")}) // mutates m in-place
	scrollAfterJ := m.detailScroll
	if scrollAfterJ <= 0 {
		t.Errorf("expected detailScroll > 0 after J, got %d", scrollAfterJ)
	}

	// Press K — detailScroll should decrease from scrollAfterJ.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")}) // mutates m in-place
	if m.detailScroll >= scrollAfterJ {
		t.Errorf("expected detailScroll < %d after K, got %d", scrollAfterJ, m.detailScroll)
	}
}

func TestWorkspaceInFormDuringQuizAndAIInput(t *testing.T) {
	m := New(theme.PastelDark(), "")

	// Not in form by default.
	if m.InForm() {
		t.Error("InForm() should be false with no active input")
	}

	// Editing sets InForm.
	m.editing = true
	if !m.InForm() {
		t.Error("InForm() should be true when editing")
	}
	m.editing = false

	// Quiz sets InForm.
	m.pendingQuiz = &domain.Quiz{Question: "Q?", Choices: []domain.QuizChoice{{Key: "a", Text: "Yes"}}}
	if !m.InForm() {
		t.Error("InForm() should be true when quiz is pending")
	}
}
