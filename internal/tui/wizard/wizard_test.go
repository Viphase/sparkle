package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/domain"
)

// pressKey delivers a single keystroke to the model and returns the updated
// model. Helpful for shaping a deterministic step-through in tests.
func pressKey(t *testing.T, m *Model, key string) *Model {
	t.Helper()
	var msg tea.KeyMsg
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		msg = tea.KeyMsg{Type: tea.KeyRight}
	case "ctrl+s":
		msg = tea.KeyMsg{Type: tea.KeyCtrlS}
	case "ctrl+t":
		msg = tea.KeyMsg{Type: tea.KeyCtrlT}
	case "ctrl+c":
		msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	updated, _ := m.Update(msg)
	return updated.(*Model)
}

func TestWizardHappyPath(t *testing.T) {
	skills := []domain.SkillDef{
		{Slug: "", Label: "none", Description: "no specialisation"},
		{Slug: "cli-tool", Label: "CLI tool", Description: "build a CLI"},
	}
	m := New("/tmp/sparkle-test", skills)
	if got := m.View(); got == "" {
		t.Fatal("View returned empty string")
	}

	// Step 1: workspace — accept default.
	m = pressKey(t, m, "enter")
	if m.step != stepTheme {
		t.Fatalf("expected stepTheme, got %d", m.step)
	}

	// Step 2: theme — cycle right once, then advance.
	m = pressKey(t, m, "right")
	m = pressKey(t, m, "enter")
	if m.step != stepAPIKey {
		t.Fatalf("expected stepAPIKey, got %d", m.step)
	}

	// Step 3: API key — skip.
	m = pressKey(t, m, "ctrl+s")
	if m.step != stepSkill {
		t.Fatalf("expected stepSkill, got %d", m.step)
	}

	// Step 4: skill — advance to "cli-tool".
	m = pressKey(t, m, "right")
	m = pressKey(t, m, "enter")
	if m.step != stepFirstSpark {
		t.Fatalf("expected stepFirstSpark, got %d", m.step)
	}

	// Step 5: type a spark title and confirm.
	m.sparkInput.SetValue("first idea")
	m = pressKey(t, m, "enter")
	if m.step < stepDone {
		t.Fatalf("expected stepDone, got %d", m.step)
	}

	res := m.Result()
	if res.Cancelled {
		t.Fatal("wizard reported cancelled on happy path")
	}
	if res.WorkspacePath != "/tmp/sparkle-test" {
		t.Fatalf("WorkspacePath=%q", res.WorkspacePath)
	}
	if res.Config.ActiveSkill != "cli-tool" {
		t.Fatalf("ActiveSkill=%q want cli-tool", res.Config.ActiveSkill)
	}
	if res.FirstSparkTitle != "first idea" {
		t.Fatalf("FirstSparkTitle=%q", res.FirstSparkTitle)
	}
	if res.Config.AnthropicAPIKey != "" {
		t.Fatalf("expected empty API key after skip, got %q", res.Config.AnthropicAPIKey)
	}
}

func TestWizardCancel(t *testing.T) {
	m := New("/tmp/x", nil)
	m = pressKey(t, m, "ctrl+c")
	if !m.cancelled {
		t.Fatal("ctrl+c should mark cancelled")
	}
	if !m.Result().Cancelled {
		t.Fatal("Result should report cancelled")
	}
}

func TestWizardEscGoesBack(t *testing.T) {
	m := New("/tmp/x", nil)
	m = pressKey(t, m, "enter") // -> theme
	m = pressKey(t, m, "esc")
	if m.step != stepWorkspace {
		t.Fatalf("esc should return to stepWorkspace, got %d", m.step)
	}
}

func TestWizardViewMentionsLogo(t *testing.T) {
	m := New("/tmp/x", nil)
	if !strings.Contains(m.View(), "ꕤ") {
		t.Fatal("wizard view missing ꕤ logo")
	}
}
