package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/viphase/sparkle/internal/domain"
)

// MockProvider returns deterministic, local-only responses for the AI screen.
type MockProvider struct{}

func NewMockProvider() MockProvider {
	return MockProvider{}
}

func (MockProvider) Ping(_ context.Context) error { return nil }

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

	// Count how many user turns exist — used for stage-complete signals.
	userTurns := 0
	for _, m := range req.Messages {
		if m.Role == domain.MessageRoleUser {
			userTurns++
		}
	}

	switch {
	case last == "":
		return domain.CompletionResponse{
			Text: "Let's shape " + title + " into a well-structured project. First, I need to understand who this is for.",
			Quizzes: []domain.Quiz{{
				Question: "Who is the primary user of " + title + "?",
				Choices: []domain.QuizChoice{
					{Key: "a", Text: "Me — a solo developer managing personal projects"},
					{Key: "b", Text: "A small team (2–5 people) sharing a codebase"},
					{Key: "c", Text: "A non-technical creator who wants to ship faster"},
					{Key: "d", Text: "Someone else — I'll describe them"},
				},
			}},
		}, nil

	case strings.Contains(last, "architecture"):
		return domain.CompletionResponse{
			Text:          "For " + title + "'s architecture, start by naming the core data model, the storage boundary, and the UI workflow. Keep Bubble Tea at the edge, and keep domain decisions in pure Go packages.",
			StageComplete: userTurns >= 4,
		}, nil

	case strings.Contains(last, "audience") || strings.Contains(last, "who") || strings.Contains(last, "user"):
		return domain.CompletionResponse{
			Text: "Good. Let me narrow the audience further.",
			Quizzes: []domain.Quiz{{
				Question: "What is the most important thing your target user lacks today?",
				Choices: []domain.QuizChoice{
					{Key: "a", Text: "A fast way to capture ideas before they fade"},
					{Key: "b", Text: "A structured place to develop half-formed plans"},
					{Key: "c", Text: "Accountability — something that tracks their actual progress"},
					{Key: "d", Text: "Something else — I'll explain"},
				},
			}},
		}, nil

	case strings.Contains(last, "roadmap") || strings.Contains(last, "milestone"):
		return domain.CompletionResponse{
			Text:          "Use three milestones: first make the manual workflow reliable, then add tracking feedback, then add AI assistance. Each milestone should end with something the user can run locally.",
			StageComplete: true,
		}, nil

	case strings.Contains(last, "risk") || strings.Contains(last, "challenge") || strings.Contains(last, "flaw") || strings.Contains(last, "landmine"):
		return domain.CompletionResponse{
			Text: "Before listing risks, let me understand your biggest concern.",
			Quizzes: []domain.Quiz{{
				Question: "What is the most likely reason " + title + " fails?",
				Choices: []domain.QuizChoice{
					{Key: "a", Text: "Overbuilding — adding features nobody asked for"},
					{Key: "b", Text: "Under-delivering — the core workflow is too rough to use"},
					{Key: "c", Text: "Wrong audience — built for me, not for actual users"},
					{Key: "d", Text: "Something else — I'll be specific"},
				},
			}},
		}, nil

	case strings.Contains(last, "descri") || strings.Contains(last, "what is") || strings.Contains(last, "explain"):
		desc := "A local-first TUI tool for capturing project ideas and developing them into structured, trackable workspaces — all in Markdown, no cloud required."
		if title != "this project" {
			desc = title + " is " + desc
		}
		return domain.CompletionResponse{
			Text:          desc,
			StageComplete: userTurns >= 3,
		}, nil

	case strings.Contains(last, "stall") || strings.Contains(last, "stuck") || strings.Contains(last, "progress"):
		// Tracking-aware response (M10): use context stats if available.
		text := "Looking at your activity data — "
		if req.Context.DaysSinceActive > 3 {
			text += fmt.Sprintf("%d days since last commit. Projects often stall when the next step is unclear. What is the single most blocked task right now?", req.Context.DaysSinceActive)
		} else if req.Context.Streak > 7 {
			text += fmt.Sprintf("you have a %d-day streak, which is strong. The risk now is momentum without direction. What milestone should this activity add up to?", req.Context.Streak)
		} else if req.Context.WeekWords > 0 {
			text += fmt.Sprintf("%d words this week. That is measurable progress. What is the next concrete output?", req.Context.WeekWords)
		} else {
			text += "no recent activity recorded. What is the actual blocker — unclear next step, low motivation, or external dependency?"
		}
		return domain.CompletionResponse{Text: text}, nil

	case strings.Contains(last, "streak") || strings.Contains(last, "words") || strings.Contains(last, "activity") || strings.Contains(last, "track"):
		text := "Tracking shows: "
		if req.Context.TodayWords > 0 {
			text += fmt.Sprintf("%d words today, ", req.Context.TodayWords)
		}
		if req.Context.WeekWords > 0 {
			text += fmt.Sprintf("%d this week, ", req.Context.WeekWords)
		}
		if req.Context.Streak > 0 {
			text += fmt.Sprintf("%d-day streak. ", req.Context.Streak)
		}
		if req.Context.TodayWords == 0 && req.Context.WeekWords == 0 {
			text = "No tracking data recorded yet for this project. Open the project files in your editor and Sparkle will start tracking word counts automatically."
		} else {
			text += "Keep the streak alive by shipping one small, testable thing each day."
		}
		return domain.CompletionResponse{Text: text}, nil

	default:
		stageComplete := strings.Contains(last, "done") || strings.Contains(last, "next") || strings.Contains(last, "ready")
		return domain.CompletionResponse{
			Text:          "I would tighten the next step into one observable outcome: what file changes, what screen reflects it, and what test proves it works.",
			StageComplete: stageComplete,
		}, nil
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
