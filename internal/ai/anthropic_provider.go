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
	defaultMaxTokens    = 2048
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

// Ping validates the API key and reachability with a minimal messages request.
func (p *AnthropicProvider) Ping(ctx context.Context) error {
	body := anthropicRequest{
		Model:     p.model,
		MaxTokens: 1,
		Messages:  []anthropicMessage{{Role: "user", Content: "ping"}},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal ping: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("build ping: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	var apiResp anthropicResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return fmt.Errorf("ping decode: %w", err)
	}
	if apiResp.Error != nil {
		return fmt.Errorf("%s: %s", apiResp.Error.Type, apiResp.Error.Message)
	}
	return nil
}

func (p *AnthropicProvider) Complete(ctx context.Context, req domain.CompletionRequest) (domain.CompletionResponse, error) {
	system := BuildSystemPrompt(req.Mode, req.Context, req.Skill)

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
	text, quizzes := parseQuizBlocks(text)
	cleanText, stageComplete := parseStageComplete(text)

	return domain.CompletionResponse{
		Text:          cleanText,
		ProposedEdits: edits,
		Quizzes:       quizzes,
		StageComplete: stageComplete,
	}, nil
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

// BuildSystemPrompt returns the system-level instruction based on mode, context,
// and an optional skill specialisation. The skill fragment is injected between
// the base prompt and the mode-specific block.
func BuildSystemPrompt(mode domain.Mode, ctx domain.ProjectContext, skills ...domain.Skill) string {
	var skill domain.Skill
	if len(skills) > 0 {
		skill = skills[0]
	}

	var b strings.Builder
	b.WriteString("You are Sparkle's local project guide.\n")
	b.WriteString("Help turn rough ideas into practical, structured project work.\n")
	b.WriteString("Be concise. Ask ONE question at a time — never stack multiple questions in one response.\n")
	b.WriteString("When the user says they just created a project and asks you to start the discovery interview, begin immediately with the first question only.\n")
	b.WriteString("Never invent facts. Never write files without explicit permission.\n\n")

	b.WriteString("QUIZ FORMAT — when a question has enumerable answers, wrap it in a quiz block\n")
	b.WriteString("so the UI displays it as a clickable widget the user answers with a single key:\n")
	b.WriteString("<quiz>\n")
	b.WriteString("Your question here?\n")
	b.WriteString("a) First option\n")
	b.WriteString("b) Second option\n")
	b.WriteString("c) Third option\n")
	b.WriteString("d) Something else — describe\n")
	b.WriteString("</quiz>\n")
	b.WriteString("Rules: at most one quiz per response; always include a free-text fallback;\n")
	b.WriteString("never embed a quiz inside an <edit> block.\n\n")

	b.WriteString("PIPELINE STAGES — the conversation flows through: clarify → structure → challenge → architect → expand → finalize.\n")
	b.WriteString("When you have gathered sufficient information for the current stage and the user should advance, add <stage-complete /> at the very end of your response.\n")
	b.WriteString("Do not add <stage-complete /> prematurely — only when you truly have enough to move forward.\n\n")

	// Inject skill-specific prompt fragment between base and mode instructions.
	if frag := skill.SystemFragment(); frag != "" {
		b.WriteString(frag)
		b.WriteString("\n\n")
	}

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

		// Tracking data — only included when non-zero so cold-start conversations
		// are not cluttered with meaningless zeros.
		if ctx.TodayWords > 0 || ctx.WeekWords > 0 || ctx.Streak > 0 {
			b.WriteString("\nTracking data (actual workspace activity):\n")
			if ctx.TodayWords > 0 {
				b.WriteString(fmt.Sprintf("  Words written today: %d\n", ctx.TodayWords))
			}
			if ctx.WeekWords > 0 {
				b.WriteString(fmt.Sprintf("  Words written this week: %d\n", ctx.WeekWords))
			}
			if ctx.Streak > 0 {
				b.WriteString(fmt.Sprintf("  Active-day streak: %d days\n", ctx.Streak))
			}
			if ctx.ActiveDaysWeek > 0 {
				b.WriteString(fmt.Sprintf("  Active days this week: %d\n", ctx.ActiveDaysWeek))
			}
			if ctx.DaysSinceActive > 1 {
				b.WriteString(fmt.Sprintf("  Days since last activity: %d — consider asking why the project stalled.\n", ctx.DaysSinceActive))
			}
		}
	}

	return strings.TrimSpace(b.String())
}

