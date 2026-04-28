package ai

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/theme"
)

type fakeProvider struct {
	seen domain.CompletionRequest
	text string
}

func (f *fakeProvider) Complete(_ context.Context, req domain.CompletionRequest) (domain.CompletionResponse, error) {
	f.seen = req
	return domain.CompletionResponse{Text: f.text}, nil
}

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
