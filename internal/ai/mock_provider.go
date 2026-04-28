package ai

import (
	"context"
	"strings"

	"github.com/viphase/sparkle/internal/domain"
)

// MockProvider returns deterministic, local-only responses for the AI screen.
type MockProvider struct{}

func NewMockProvider() MockProvider {
	return MockProvider{}
}

func (MockProvider) Complete(ctx context.Context, req domain.CompletionRequest) (domain.CompletionResponse, error) {
	select {
	case <-ctx.Done():
		return domain.CompletionResponse{}, ctx.Err()
	default:
	}

	last := strings.ToLower(strings.TrimSpace(lastUserMessage(req.Messages)))
	title := strings.TrimSpace(req.Context.Title)
	if title == "" {
		title = "this project"
	}

	switch {
	case last == "":
		return domain.CompletionResponse{Text: "Tell me what you want to develop next, and I will help shape it into a concrete project step."}, nil
	case strings.Contains(last, "architecture"):
		return domain.CompletionResponse{Text: "For " + title + "'s architecture, start by naming the core data model, the storage boundary, and the UI workflow. Keep Bubble Tea at the edge, and keep domain decisions in pure Go packages."}, nil
	case strings.Contains(last, "audience") || strings.Contains(last, "user"):
		return domain.CompletionResponse{Text: "Define the audience as one specific person with one recurring problem. Then write what they can do in Sparkle in under a minute that they could not do comfortably before."}, nil
	case strings.Contains(last, "roadmap") || strings.Contains(last, "milestone"):
		return domain.CompletionResponse{Text: "Use three milestones: first make the manual workflow reliable, then add tracking feedback, then add AI assistance. Each milestone should end with something the user can run locally."}, nil
	case strings.Contains(last, "risk") || strings.Contains(last, "challenge"):
		return domain.CompletionResponse{Text: "The main risk is overbuilding before the local workflow feels good. Validate the smallest capture-to-project path before adding more provider or automation choices."}, nil
	default:
		return domain.CompletionResponse{Text: "I would tighten the next step into one observable outcome: what file changes, what screen reflects it, and what test proves it works."}, nil
	}
}

func lastUserMessage(messages []domain.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == domain.MessageRoleUser {
			return messages[i].Content
		}
	}
	return ""
}
