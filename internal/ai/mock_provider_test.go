package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/viphase/sparkle/internal/domain"
)

func TestMockProviderRespondsDeterministically(t *testing.T) {
	provider := NewMockProvider()
	req := domain.CompletionRequest{
		Messages: []domain.Message{{Role: domain.MessageRoleUser, Content: "Help with architecture"}},
		Context:  domain.ProjectContext{Title: "Sparkle"},
	}

	got, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Text, "Sparkle") || !strings.Contains(strings.ToLower(got.Text), "architecture") {
		t.Fatalf("unexpected response: %q", got.Text)
	}

	got2, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got2.Text != got.Text {
		t.Fatalf("mock provider must be deterministic: %q != %q", got2.Text, got.Text)
	}
}

func TestBuildPromptIncludesContextAndMessages(t *testing.T) {
	prompt := BuildPrompt(domain.CompletionRequest{
		Messages: []domain.Message{
			{Role: domain.MessageRoleUser, Content: "What next?"},
			{Role: domain.MessageRoleAssistant, Content: "Pick one milestone."},
		},
		Context: domain.ProjectContext{
			Title:          "Sparkle",
			Status:         domain.ProjectStatusActive,
			TargetAudience: "solo builders",
		},
	})

	for _, want := range []string{"Project context", "Title: Sparkle", "Target audience: solo builders", "USER: What next?", "ASSISTANT: Pick one milestone."} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}
