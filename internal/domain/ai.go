package domain

import "strings"

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

// QuizChoice is one lettered option in a multiple-choice quiz.
type QuizChoice struct {
	Key  string // "a", "b", "c", "d" …
	Text string // the choice description
}

// Quiz is a multiple-choice question the AI embeds in a response to speed up
// decisions and reduce user cognitive load. The user answers with a single key.
type Quiz struct {
	Question string
	Choices  []QuizChoice
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

	// Tracking fields (M10) — populated when stats are available for the project.
	// Zero values mean "no tracking data yet"; providers must treat them as absent.
	TodayWords    int
	WeekWords     int
	Streak        int // current consecutive active-day streak
	ActiveDaysWeek int
	DaysSinceActive int // 0 = active today; -1 = no activity ever recorded
}

// CompletionRequest is the provider-neutral request shape for AI guide calls.
type CompletionRequest struct {
	Messages []Message
	Context  ProjectContext
	Mode     Mode
	Skill    Skill // optional specialisation — injected into system prompt
}

// ArtifactStatus records which project artifacts have been filled in.
// Each field is true when the corresponding section has content.
type ArtifactStatus struct {
	Description    bool
	Architecture   bool
	TargetAudience bool
	Roadmap        bool
	Notes          bool
	Flaws          bool
	Plan           bool
}

// ArtifactStatusFromContext derives artifact status from a ProjectContext.
// Fields tracked only inside the AI screen (Flaws, Plan) stay false and are
// updated separately by the screen.
func ArtifactStatusFromContext(ctx ProjectContext) ArtifactStatus {
	return ArtifactStatus{
		Description:    strings.TrimSpace(ctx.Description) != "",
		Architecture:   strings.TrimSpace(ctx.Architecture) != "",
		TargetAudience: strings.TrimSpace(ctx.TargetAudience) != "",
		Roadmap:        strings.TrimSpace(ctx.Roadmap) != "",
		Notes:          strings.TrimSpace(ctx.Notes) != "",
	}
}

// FilledCount returns how many of the 7 artifacts have content.
func (a ArtifactStatus) FilledCount() int {
	n := 0
	if a.Description {
		n++
	}
	if a.Architecture {
		n++
	}
	if a.TargetAudience {
		n++
	}
	if a.Roadmap {
		n++
	}
	if a.Notes {
		n++
	}
	if a.Flaws {
		n++
	}
	if a.Plan {
		n++
	}
	return n
}

// CompletionResponse is the provider-neutral response shape.
type CompletionResponse struct {
	Text          string
	ProposedEdits []ProposedEdit
	// Quizzes holds any multiple-choice questions the AI embedded in the
	// response. They are stripped from Text and held here for the UI to display
	// as interactive widgets.
	Quizzes []Quiz
	// StageComplete signals that the AI considers the current pipeline stage
	// done and the conversation should advance to the next mode.
	StageComplete bool
}
