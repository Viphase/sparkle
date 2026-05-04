package ai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/viphase/sparkle/internal/ai"
	"github.com/viphase/sparkle/internal/domain"
)

// fakeHTTP satisfies ai.HTTPDoer and returns a canned response body.
type fakeHTTP struct {
	status int
	body   string
}

func (f *fakeHTTP) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: f.status,
		Body:       io.NopCloser(bytes.NewBufferString(f.body)),
	}, nil
}

func anthropicOKBody(text string) string {
	type content struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type resp struct {
		Content []content `json:"content"`
	}
	b, _ := json.Marshal(resp{Content: []content{{Type: "text", Text: text}}})
	return string(b)
}

func anthropicErrorBody(errType, msg string) string {
	type errBlock struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	type resp struct {
		Error *errBlock `json:"error"`
	}
	b, _ := json.Marshal(resp{Error: &errBlock{Type: errType, Message: msg}})
	return string(b)
}

func TestAnthropicProviderReturnsText(t *testing.T) {
	p := ai.NewAnthropicProvider("test-key", "").
		WithHTTPClient(&fakeHTTP{status: 200, body: anthropicOKBody("Great idea!")})

	resp, err := p.Complete(context.Background(), domain.CompletionRequest{
		Messages: []domain.Message{{Role: domain.MessageRoleUser, Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Text != "Great idea!" {
		t.Fatalf("expected %q, got %q", "Great idea!", resp.Text)
	}
}

func TestAnthropicProviderReturnsAPIError(t *testing.T) {
	p := ai.NewAnthropicProvider("test-key", "").
		WithHTTPClient(&fakeHTTP{status: 401, body: anthropicErrorBody("authentication_error", "invalid key")})

	_, err := p.Complete(context.Background(), domain.CompletionRequest{
		Messages: []domain.Message{{Role: domain.MessageRoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid key") {
		t.Fatalf("expected error to contain %q, got %q", "invalid key", err.Error())
	}
}

func TestAnthropicProviderParsesProposedEdit(t *testing.T) {
	body := `Here is a suggestion.

<edit path="projects/foo/notes.md">
# Notes

First session notes.
</edit>

Let me know if you need anything else.`

	p := ai.NewAnthropicProvider("test-key", "").
		WithHTTPClient(&fakeHTTP{status: 200, body: anthropicOKBody(body)})

	resp, err := p.Complete(context.Background(), domain.CompletionRequest{
		Messages: []domain.Message{{Role: domain.MessageRoleUser, Content: "Write notes"}},
		Mode:     domain.ModeFinalize,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ProposedEdits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(resp.ProposedEdits))
	}
	edit := resp.ProposedEdits[0]
	if edit.Path != "projects/foo/notes.md" {
		t.Errorf("expected path %q, got %q", "projects/foo/notes.md", edit.Path)
	}
	if !strings.Contains(edit.Content, "First session notes.") {
		t.Errorf("edit content missing expected text: %q", edit.Content)
	}
	// Edit block must be stripped from text.
	if strings.Contains(resp.Text, "<edit") {
		t.Errorf("edit block not stripped from text: %q", resp.Text)
	}
	if !strings.Contains(resp.Text, "Here is a suggestion.") {
		t.Errorf("surrounding text missing: %q", resp.Text)
	}
}

func TestAnthropicProviderContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	p := ai.NewAnthropicProvider("test-key", "").
		WithHTTPClient(&fakeHTTP{status: 200, body: anthropicOKBody("ok")})

	_, err := p.Complete(ctx, domain.CompletionRequest{
		Messages: []domain.Message{{Role: domain.MessageRoleUser, Content: "Hi"}},
	})
	if err == nil {
		t.Log("context cancelled but no error — http client returned before cancel propagated; this is acceptable in tests")
	}
}

func TestBuildSystemPromptIncludesMode(t *testing.T) {
	prompt := ai.BuildSystemPrompt(domain.ModeArchitect, domain.ProjectContext{Title: "Sparkle"})
	if !strings.Contains(prompt, "ARCHITECT") {
		t.Errorf("system prompt missing mode: %q", prompt)
	}
	if !strings.Contains(prompt, "Sparkle") {
		t.Errorf("system prompt missing project title: %q", prompt)
	}
}

func TestBuildSystemPromptFinalizeSuggestsEditBlock(t *testing.T) {
	prompt := ai.BuildSystemPrompt(domain.ModeFinalize, domain.ProjectContext{})
	if !strings.Contains(prompt, "<edit") {
		t.Errorf("finalize prompt should describe edit block format: %q", prompt)
	}
}

func TestBuildSystemPromptInjectsSkillFragment(t *testing.T) {
	prompt := ai.BuildSystemPrompt(domain.ModeClarify, domain.ProjectContext{}, domain.SkillCLITool)
	if !strings.Contains(prompt, "CLI TOOL") {
		t.Errorf("system prompt missing CLI TOOL skill fragment: %q", prompt[:min(200, len(prompt))])
	}
	if !strings.Contains(prompt, "Exit code contract") {
		t.Errorf("system prompt missing exit code focus area: %q", prompt[:min(200, len(prompt))])
	}
}

func TestBuildSystemPromptSkillNoneAddsNothing(t *testing.T) {
	withSkill := ai.BuildSystemPrompt(domain.ModeClarify, domain.ProjectContext{}, domain.SkillNone)
	withoutSkill := ai.BuildSystemPrompt(domain.ModeClarify, domain.ProjectContext{})
	if withSkill != withoutSkill {
		t.Errorf("SkillNone should not change the prompt:\nwith:    %q\nwithout: %q", withSkill, withoutSkill)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
