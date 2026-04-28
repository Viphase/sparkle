package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/viphase/sparkle/internal/domain"
)

const (
	anthropicAPIURL     = "https://api.anthropic.com/v1/messages"
	anthropicAPIVersion = "2023-06-01"
	defaultMaxTokens    = 1024
)

// HTTPDoer is the interface the provider uses to make requests.
// Replaced in tests to avoid real network calls.
type HTTPDoer interface {
	Do(r *http.Request) (*http.Response, error)
}

// AnthropicProvider calls the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey string
	model  string
	client HTTPDoer
}

// NewAnthropicProvider creates a ready-to-use provider. model defaults to
// "claude-haiku-4-5" when empty.
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = "claude-haiku-4-5"
	}
	return &AnthropicProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// WithHTTPClient replaces the HTTP client; used in tests to avoid real network
// calls.
func (p *AnthropicProvider) WithHTTPClient(c HTTPDoer) *AnthropicProvider {
	p.client = c
	return p
}

// anthropicRequest is the Anthropic API request shape.
type anthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	System    string              `json:"system,omitempty"`
	Messages  []anthropicMessage  `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the Anthropic API response shape (simplified).
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *AnthropicProvider) Complete(ctx context.Context, req domain.CompletionRequest) (domain.CompletionResponse, error) {
	system := BuildSystemPrompt(req.Mode, req.Context)

	var msgs []anthropicMessage
	for _, m := range req.Messages {
		if m.Role == domain.MessageRoleSystem {
			continue // folded into the top-level system field
		}
		role := "user"
		if m.Role == domain.MessageRoleAssistant {
			role = "assistant"
		}
		msgs = append(msgs, anthropicMessage{Role: role, Content: m.Content})
	}
	if len(msgs) == 0 {
		msgs = append(msgs, anthropicMessage{Role: "user", Content: "Hello"})
	}

	body := anthropicRequest{
		Model:     p.model,
		MaxTokens: defaultMaxTokens,
		System:    system,
		Messages:  msgs,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(raw))
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("read response: %w", err)
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("decode response: %w", err)
	}
	if apiResp.Error != nil {
		return domain.CompletionResponse{}, fmt.Errorf("anthropic error (%s): %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	var textParts []string
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			textParts = append(textParts, block.Text)
		}
	}
	full := strings.Join(textParts, "\n")
	text, edits := parseProposedEdits(full)

	return domain.CompletionResponse{Text: text, ProposedEdits: edits}, nil
}

// editBlockRE matches fenced edit blocks:
//
//	<edit path="some/file.md">
//	...content...
//	</edit>
var editBlockRE = regexp.MustCompile(`(?s)<edit path="([^"]+)">\n?(.*?)\n?</edit>`)

// parseProposedEdits strips <edit> blocks from text and returns them as ProposedEdits.
func parseProposedEdits(text string) (string, []domain.ProposedEdit) {
	var edits []domain.ProposedEdit
	clean := editBlockRE.ReplaceAllStringFunc(text, func(match string) string {
		sub := editBlockRE.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		path := strings.TrimSpace(sub[1])
		content := strings.TrimSpace(sub[2])
		description := descriptionFromContent(content)
		edits = append(edits, domain.ProposedEdit{
			Path:        path,
			Description: description,
			Content:     content,
		})
		return ""
	})
	return strings.TrimSpace(clean), edits
}

func descriptionFromContent(content string) string {
	lines := strings.SplitN(content, "\n", 3)
	for _, l := range lines {
		l = strings.TrimSpace(strings.TrimPrefix(l, "#"))
		l = strings.TrimSpace(l)
		if l != "" {
			return truncateDesc(l, 80)
		}
	}
	return "edit proposed"
}

func truncateDesc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// BuildSystemPrompt returns the system-level instruction based on mode and context.
func BuildSystemPrompt(mode domain.Mode, ctx domain.ProjectContext) string {
	var b strings.Builder
	b.WriteString("You are Sparkle's local project guide.\n")
	b.WriteString("Help turn rough ideas into practical, structured project work.\n")
	b.WriteString("Be concise. Ask one clarifying question at a time when context is thin.\n")
	b.WriteString("Never invent facts. Never write files without explicit permission.\n\n")

	switch mode {
	case domain.ModeClarify:
		b.WriteString("Current mode: CLARIFY — ask precise, probing questions to sharpen the idea before giving answers.\n")
	case domain.ModeStructure:
		b.WriteString("Current mode: STRUCTURE — help the user organise project sections: description, architecture, audience, roadmap.\n")
	case domain.ModeChallenge:
		b.WriteString("Current mode: CHALLENGE — respectfully push back on weak assumptions. Point out gaps. Do not validate everything.\n")
	case domain.ModeArchitect:
		b.WriteString("Current mode: ARCHITECT — advise on technical design: data models, system boundaries, async patterns, testing.\n")
	case domain.ModeExpand:
		b.WriteString("Current mode: EXPAND — elaborate on the chosen section or concept. Go deeper without inventing constraints.\n")
	case domain.ModeFinalize:
		b.WriteString("Current mode: FINALIZE — produce clean Markdown-ready output for the relevant project section.\n")
		b.WriteString("If you want to propose a file change, wrap it in an <edit> block:\n")
		b.WriteString("<edit path=\"relative/path/to/file.md\">\n...full new content...\n</edit>\n")
		b.WriteString("Only propose one file at a time. Never write without an explicit edit block.\n")
	}

	if ctx.Title != "" {
		b.WriteString("\nProject context:\n")
		writeField(&b, "Title", ctx.Title)
		writeField(&b, "Status", string(ctx.Status))
		writeField(&b, "Description", ctx.Description)
		writeField(&b, "Architecture", ctx.Architecture)
		writeField(&b, "Target audience", ctx.TargetAudience)
		writeField(&b, "Roadmap", ctx.Roadmap)
	}

	return strings.TrimSpace(b.String())
}

