package domain

// MessageRole identifies who authored an AI guide message.
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

// Message is one turn in an AI guide conversation.
type Message struct {
	Role    MessageRole
	Content string
}

// Mode selects the AI guide's operational stance for a request.
type Mode string

const (
	ModeClarify   Mode = "clarify"   // ask probing questions to sharpen the idea
	ModeStructure Mode = "structure" // help organise project sections
	ModeChallenge Mode = "challenge" // push back on weak assumptions
	ModeArchitect Mode = "architect" // advise on technical design
	ModeExpand    Mode = "expand"    // elaborate on a section or concept
	ModeFinalize  Mode = "finalize"  // produce Markdown-ready output after full context
)

// AllModes returns modes in display order.
func AllModes() []Mode {
	return []Mode{ModeClarify, ModeStructure, ModeChallenge, ModeArchitect, ModeExpand, ModeFinalize}
}

// Label returns a short human-readable label.
func (m Mode) Label() string {
	switch m {
	case ModeClarify:
		return "clarify"
	case ModeStructure:
		return "structure"
	case ModeChallenge:
		return "challenge"
	case ModeArchitect:
		return "architect"
	case ModeExpand:
		return "expand"
	case ModeFinalize:
		return "finalize"
	}
	return string(m)
}

// ProposedEdit is a file change the AI guide wants to make, held for explicit
// user approval before any write happens.
type ProposedEdit struct {
	// Path is workspace-relative (e.g. "projects/project_foo_20260428_ab12/notes.md").
	Path string
	// Description is a one-line summary of what the edit does.
	Description string
	// Content is the full new file content proposed by the AI.
	Content string
}

// ProjectContext is the project snapshot passed to an AI provider.
type ProjectContext struct {
	ProjectID      string
	Title          string
	Status         ProjectStatus
	Description    string
	Architecture   string
	TargetAudience string
	Roadmap        string
	Notes          string
}

// CompletionRequest is the provider-neutral request shape for AI guide calls.
type CompletionRequest struct {
	Messages []Message
	Context  ProjectContext
	Mode     Mode
}

// CompletionResponse is the provider-neutral response shape.
type CompletionResponse struct {
	Text          string
	ProposedEdits []ProposedEdit
}
