# Sparkle AI Guide Design

## v1 Scope

v1 ships the smallest surface that lets a real provider drop in later:

- AI provider interface
- mock provider
- prompt builder
- a single basic AI screen (chat-like, one mode)

That is all. Modes, the proposed-edit data model, diff preview, and approval flow are deferred to milestone 6, when a real provider exists to test them against. Designing them in the abstract risks locking in shapes we haven't validated.

## AI Role (when fully built)

Sparkle's AI acts as:
- project-development mentor
- teacher
- product strategist
- software architect

It helps the user turn vague ideas into structured, practical projects.

## AI Must Do

- ask precise questions when an idea is vague
- challenge weak assumptions respectfully
- help define target audience
- help define project architecture
- preserve the user's voice
- avoid overbuilding
- suggest concrete alternatives when useful
- generate Markdown-ready sections only after enough context exists
- ask for explicit approval before writing files

## AI Must Not Do

- silently overwrite project files
- invent facts about local projects it did not read
- expose unrelated private project details when analyzing context
- generate huge specs before asking clarifying questions
- push unnecessary complexity

## Provider Interface (v1)

```go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

type CompletionRequest struct {
    Messages []Message
    Context  ProjectContext
}

type CompletionResponse struct {
    Text string
}
```

`ProposedEdit`, mode tags, and other request fields are added in M6 alongside the real provider.

## Mock Provider

`mock_provider.go` returns deterministic canned responses keyed off the last user message. Enough for the AI screen to render real-looking output and for tests to assert on flow, not content.

## Prompt Builder

A small function that turns `(messages, project context)` into a single prompt string. Pure and testable.

## AI Screen (v1)

Minimal chat layout:
- message list
- input box
- send / cancel
- "this is a mock provider" indicator

No file-change preview, no approval flow, no mode selector — those land in M6.

## Deferred to M6

When a real provider is wired up, design and implement:
- mode taxonomy (clarify / structure / challenge / architect / expand / finalize) — only the modes that earn their keep after testing
- `ProposedEdit` data model
- file-change diff preview
- explicit approve/reject flow before any write
- master system prompt — written against actual model behavior, not in the abstract
- conversation persistence

Sketched here as a reminder of where the system is going, not as a v1 spec.
