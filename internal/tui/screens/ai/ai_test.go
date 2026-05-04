package ai

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	guide "github.com/viphase/sparkle/internal/ai"
	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/theme"
)

type fakeProvider struct {
	seen          domain.CompletionRequest
	text          string
	stageComplete bool
}

func (f *fakeProvider) Complete(_ context.Context, req domain.CompletionRequest) (domain.CompletionResponse, error) {
	f.seen = req
	return domain.CompletionResponse{Text: f.text, StageComplete: f.stageComplete}, nil
}

func (f *fakeProvider) Ping(_ context.Context) error { return nil }

func TestAIEnterSendsMessageThroughProvider(t *testing.T) {
	provider := &fakeProvider{text: "mock answer"}
	m := New(theme.PastelDark(), provider).(*Model)
	next, _ := m.Update(msgs.ProjectsLoadedMsg{Items: []domain.Project{
		{
			ID:             "project_sparkle",
			Title:          "Sparkle",
			Status:         domain.ProjectStatusActive,
			TargetAudience: "solo builders",
			Body:           "# Description\n\nLocal project manager.\n\n# Architecture\n\nClean Go packages.",
		},
	}})
	m = next.(*Model)
	m.input.SetValue("Help with roadmap")

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(*Model)
	if !got.waiting {
		t.Fatal("expected waiting=true after send")
	}
	if cmd == nil {
		t.Fatal("expected provider command")
	}

	msg := cmd()
	next, _ = got.Update(msg)
	got = next.(*Model)
	if got.waiting {
		t.Fatal("expected waiting=false after completion")
	}
	if len(provider.seen.Messages) == 0 || provider.seen.Messages[len(provider.seen.Messages)-1].Content != "Help with roadmap" {
		t.Fatalf("provider did not receive user message: %+v", provider.seen.Messages)
	}
	if provider.seen.Context.Title != "Sparkle" || provider.seen.Context.Architecture != "Clean Go packages." {
		t.Fatalf("provider did not receive project context: %+v", provider.seen.Context)
	}
	if got.messages[len(got.messages)-1].Content != "mock answer" {
		t.Fatalf("assistant response not appended: %+v", got.messages)
	}
}

func TestAIViewShowsMockProviderIndicator(t *testing.T) {
	m := New(theme.PastelDark()).(*Model)
	view := m.View(90, 24)
	if !strings.Contains(view, "mock provider") {
		t.Fatalf("view missing provider indicator: %q", view)
	}
}

func TestAIScrollsWithKeys(t *testing.T) {
	m := New(theme.PastelDark()).(*Model)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	got := next.(*Model)
	if got.scroll != 1 {
		t.Fatalf("scroll after up=%d, want 1", got.scroll)
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = next.(*Model)
	if got.scroll != 0 {
		t.Fatalf("scroll after down=%d, want 0", got.scroll)
	}
}

func TestStageAdviseShownInHintAfterSignal(t *testing.T) {
	provider := &fakeProvider{text: "looks good!", stageComplete: true}
	m := New(theme.PastelDark(), provider).(*Model)
	m.input.SetValue("describe the project")

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*Model)
	msg := cmd()
	next, _ = m.Update(msg)
	m = next.(*Model)

	if !m.stageAdvise {
		t.Fatal("expected stageAdvise=true after StageComplete response")
	}
	hint := m.hintLine()
	if !strings.Contains(hint, "stage done") {
		t.Fatalf("hint should mention stage done, got: %q", hint)
	}
}

func TestCycleModeMarksVisited(t *testing.T) {
	m := New(theme.PastelDark()).(*Model)
	if !m.visitedStages[domain.ModeClarify] {
		t.Fatal("clarify should be pre-visited on init")
	}
	m.cycleMode(1) // advance to structure
	if !m.visitedStages[domain.ModeStructure] {
		t.Fatal("structure should be marked visited after cycle")
	}
	if m.stageAdvise {
		t.Fatal("stageAdvise should be cleared after cycling mode")
	}
}

func TestArtifactStatusFromContext(t *testing.T) {
	ctx := domain.ProjectContext{
		Title:          "Sparkle",
		Description:    "A local TUI.",
		Architecture:   "Clean Go.",
		TargetAudience: "solo devs",
	}
	a := domain.ArtifactStatusFromContext(ctx)
	if !a.Description || !a.Architecture || !a.TargetAudience {
		t.Fatalf("expected desc/arch/audience true, got %+v", a)
	}
	if a.Roadmap || a.Notes || a.Flaws || a.Plan {
		t.Fatalf("expected roadmap/notes/flaws/plan false, got %+v", a)
	}
	if a.FilledCount() != 3 {
		t.Fatalf("expected FilledCount=3, got %d", a.FilledCount())
	}
}

func TestViewIncludesPipelineAndArtifacts(t *testing.T) {
	m := New(theme.PastelDark()).(*Model)
	m.context = domain.ProjectContext{
		Title:       "Sparkle",
		Description: "A local TUI.",
	}
	m.artifacts = domain.ArtifactStatusFromContext(m.context)
	// Mark structure as visited to confirm pipeline shows it.
	m.visitedStages[domain.ModeStructure] = true
	view := m.View(100, 30)
	if !strings.Contains(view, "artifacts") {
		t.Fatalf("view should contain artifact checklist, got:\n%s", view)
	}
	// Pipeline glyphs should appear.
	if !strings.Contains(view, "●") {
		t.Fatalf("view should contain active stage bullet ●, got:\n%s", view)
	}
}

func TestTrackingLoadedEnrichesContext(t *testing.T) {
	m := New(theme.PastelDark()).(*Model)
	// First load a project context.
	m.context = domain.ProjectContext{
		ProjectID: "project_sparkle",
		Title:     "Sparkle",
	}

	// Use a timestamp a few seconds in the past to ensure it's on the same
	// calendar day regardless of timezone, even near midnight boundaries.
	now := time.Now()
	events := map[string][]domain.TrackingEvent{
		"project_sparkle": {
			{
				Timestamp: now.Add(-5 * time.Second),
				Type:      domain.EventWordsAdded,
				Value:     250,
				Source:    "auto",
			},
		},
	}

	next, _ := m.Update(msgs.TrackingLoadedMsg{AllEvents: events})
	got := next.(*Model)
	if got.context.TodayWords != 250 {
		t.Fatalf("expected TodayWords=250 after tracking load, got %d", got.context.TodayWords)
	}
}

func TestSystemPromptIncludesTrackingStats(t *testing.T) {
	ctx := domain.ProjectContext{
		Title:       "Sparkle",
		TodayWords:  120,
		WeekWords:   800,
		Streak:      5,
		ActiveDaysWeek: 4,
	}
	prompt := guide.BuildSystemPrompt(domain.ModeClarify, ctx)
	if !strings.Contains(prompt, "120") {
		t.Fatalf("system prompt should include today word count 120: %q", prompt)
	}
	if !strings.Contains(prompt, "Tracking data") {
		t.Fatalf("system prompt should include tracking section: %q", prompt)
	}
}

func TestRenderMessagesUsesScrollOffset(t *testing.T) {
	m := New(theme.PastelDark()).(*Model)
	m.messages = []domain.Message{
		{Role: domain.MessageRoleAssistant, Content: "old line"},
		{Role: domain.MessageRoleUser, Content: "middle line"},
		{Role: domain.MessageRoleAssistant, Content: "new line"},
	}

	bottom := m.renderMessages(40, 1)
	if !strings.Contains(bottom, "new line") || strings.Contains(bottom, "old line") {
		t.Fatalf("bottom view should show newest line only: %q", bottom)
	}

	m.scroll = 2
	older := m.renderMessages(40, 1)
	if !strings.Contains(older, "old line") {
		t.Fatalf("scrolled view should show older line: %q", older)
	}
}
