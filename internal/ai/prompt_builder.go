package ai

import (
	"strings"

	"github.com/viphase/sparkle/internal/domain"
)

// BuildPrompt serializes context and chat turns into a deterministic prompt
// string. Used by the mock provider; real providers call BuildSystemPrompt
// separately and map messages to their native API shape.
func BuildPrompt(req domain.CompletionRequest) string {
	var b strings.Builder
	b.WriteString("You are Sparkle's local project guide.\n")
	b.WriteString("Help turn rough ideas into practical, structured project work.\n")
	if req.Mode != "" {
		b.WriteString("Mode: ")
		b.WriteString(req.Mode.Label())
		b.WriteString("\n")
	}
	writeContext(&b, req.Context)
	if len(req.Messages) > 0 {
		b.WriteString("\nConversation:\n")
	}
	for _, msg := range req.Messages {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		b.WriteString(strings.ToUpper(string(msg.Role)))
		b.WriteString(": ")
		b.WriteString(content)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func writeContext(b *strings.Builder, ctx domain.ProjectContext) {
	if ctx == (domain.ProjectContext{}) {
		return
	}
	b.WriteString("\nProject context:\n")
	writeField(b, "Title", ctx.Title)
	writeField(b, "Status", string(ctx.Status))
	writeField(b, "Description", ctx.Description)
	writeField(b, "Architecture", ctx.Architecture)
	writeField(b, "Target audience", ctx.TargetAudience)
	writeField(b, "Roadmap", ctx.Roadmap)
	writeField(b, "Notes", ctx.Notes)
}

func writeField(b *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	b.WriteString("- ")
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteString("\n")
}
