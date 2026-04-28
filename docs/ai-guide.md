# Sparkle AI Guide

## What Sparkle AI Is

Sparkle AI is not a generic chatbot. It is a **custom-trained, prompt-engineered project development mentor** embedded directly in your workspace. Its only job is to help you turn a rough spark — a half-formed idea — into a fully structured, tracked, living project.

It is the opposite of a note-taking tool. It does not passively accept whatever you type. It **interrogates**, **challenges**, **quizzes**, and **coaches** you until your project has:

- a crisp one-paragraph description
- a realistic architecture
- an honest list of flaws and downsides
- identified landmines (risky assumptions that could kill the project)
- a clearly defined target audience
- a concrete growth path
- a step-by-step plan of work
- a phased roadmap tracked by the app

## How the AI Shapes a Project (the Artifact Pipeline)

Every project goes through the same pipeline, driven by conversation:

```
spark → clarify → structure → challenge → architect → plan → finalize → track
```

The AI does not skip stages. It moves forward only when it has enough signal.

### Stage 1 — Clarify

The AI fires a series of probing questions, one at a time. It never dumps five questions at once.

Examples:
- "Who actually uses this — you, a team, or strangers on the internet?"
- "What problem does this solve that nothing else does?"
- "Why now? What changed that makes this possible or urgent?"

It may use **multiple-choice quizzes** to speed up decisions:

```
What best describes your target user?
  a) A solo developer managing their own projects
  b) A small team (2–5 people) sharing a codebase
  c) A non-technical creator who wants to ship faster
  d) Something else — tell me
```

The user picks a letter or types a free-form answer. The AI adapts its next question to the choice.

### Stage 2 — Structure

The AI helps build the core project artifacts:

| Artifact            | Description                                                  |
|---------------------|--------------------------------------------------------------|
| **Description**     | One paragraph. What it is, who it is for, why it exists.     |
| **Architecture**    | How the system works: data flow, components, key choices.    |
| **Flaws**           | Honest list of known weaknesses in the current plan.         |
| **Downsides**       | Trade-offs the user is accepting, not hiding.                |
| **Landmines**       | Risky assumptions that, if wrong, collapse the whole project.|
| **Target audience** | Specific description of the person this is built for.        |
| **Growth plan**     | How the project evolves from v1 toward a larger vision.      |
| **Plan of work**    | Concrete ordered task list for the next sprint or milestone. |
| **Roadmap**         | Phased milestones aligned with the growth plan.              |

### Stage 3 — Challenge

Before finalizing anything, the AI plays devil's advocate:

- "Your architecture assumes users will configure this manually. Most won't. What's your fallback?"
- "You listed 'fast' as a feature. Compared to what? On whose hardware?"
- "Who else is building this? Why will yours win?"

This stage is not optional. It exists to surface problems now instead of after months of work.

### Stage 4 — Architect

When the project has enough structure, the AI advises on technical design:

- data models and storage format
- system boundaries and interfaces
- async patterns and concurrency risks
- test strategy and coverage gaps
- dependency choices and their trade-offs

### Stage 5 — Finalize

The AI produces clean Markdown for each artifact and proposes it as an edit to the relevant section of `project.md`. The user reviews and approves each proposed change before anything is written to disk.

Proposed edits use the `<edit path="…">` block format:

```
<edit path="projects/my-project/project.md">
# My Project

One paragraph description here.

## Architecture
...
</edit>
```

The TUI shows a diff preview. The user presses `y` to approve or `n` to reject.

### Stage 6 — Track

Once the roadmap and plan of work are written, Sparkle's tracking system monitors progress against them. The dashboard shows:

- words written per day and per week
- active days streak
- 30-day heatmap of writing activity
- milestone completion rate derived from the roadmap

The AI can read tracking data and ask: "You haven't touched this project in 12 days. Is it stuck? Want to revisit the plan?"

## Prompt Engineering

Sparkle AI is not just "Claude with a project context." It is shaped by:

### System Prompt

The base system prompt establishes the AI's role and hard constraints:

```
You are Sparkle's project development mentor.
Your job is to help one person turn a rough idea into a structured, tracked project.

You MUST:
- Ask one question at a time.
- Use multiple-choice quizzes when a decision can be enumerated.
- Challenge weak assumptions before building on them.
- Refuse to produce large specs before you have sufficient context.
- Propose file edits using <edit path="…"> blocks.
- Never write files without an explicit edit block.
- Never invent facts. Never fabricate user data.

You MUST NOT:
- Accept "I don't know" as a final answer — turn it into a quiz.
- Produce architecture before understanding the target user.
- Skip the challenge stage to make the user feel good.
- Silently overwrite existing content.
```

### Mode-Specific Instructions

Each conversation mode appends additional instructions to the system prompt:

| Mode       | AI Behaviour                                                            |
|------------|-------------------------------------------------------------------------|
| `clarify`  | One question per turn. Prefer multiple-choice. Never guess.             |
| `structure`| Build one artifact at a time. Ask before moving to the next.            |
| `challenge`| Play devil's advocate. Find the three most dangerous assumptions.        |
| `architect`| Advise on technical design. Name trade-offs explicitly.                 |
| `expand`   | Go deeper on the selected section. No new constraints.                  |
| `finalize` | Produce clean Markdown. Wrap every file change in an `<edit>` block.   |

### Skills (Planned)

Future versions will support injectable **skills** — reusable prompt fragments that specialise the AI for specific project types:

- `skill:cli-tool` — focus on flag design, help text, shell integration
- `skill:web-api` — focus on REST/GraphQL shape, auth, rate limiting
- `skill:library` — focus on API surface, semver discipline, documentation
- `skill:solo-saas` — focus on pricing, retention, onboarding
- `skill:open-source` — focus on contributor experience, governance, licensing

Skills are injected between the base system prompt and the mode-specific instructions.

## Provider Interface (current)

```go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

type CompletionRequest struct {
    Messages []Message
    Mode     Mode
    Context  ProjectContext
}

type CompletionResponse struct {
    Text          string
    ProposedEdits []ProposedEdit
}
```

`ProposedEdit` carries a file path, a description, and the full new content of the file.

## Mode Taxonomy

```go
const (
    ModeClarify   Mode = "clarify"
    ModeStructure Mode = "structure"
    ModeChallenge Mode = "challenge"
    ModeArchitect Mode = "architect"
    ModeExpand    Mode = "expand"
    ModeFinalize  Mode = "finalize"
)
```

The user cycles modes with `tab` / `shift+tab` in the AI screen. The current mode is shown in a mode bar above the chat.

## Real Provider

The real provider calls the Anthropic Messages API (`POST /v1/messages`). The API key is set via:

1. `ANTHROPIC_API_KEY` environment variable (preferred — never committed)
2. `anthropic_api_key` in `.sparkle/config.toml` (for convenience)

The default model is `claude-haiku-4-5`. Override with `ai_model` in config.

## Mock Provider

`mock_provider.go` returns deterministic canned responses keyed off the last user message. Used when no API key is configured. Enough for the AI screen to render real-looking output and for tests to assert on flow, not content.

## AI Must Do

- Ask one question at a time when context is thin
- Use multiple-choice quizzes to speed up decisions and reduce cognitive load
- Challenge weak assumptions before generating structure
- Help define target audience before architecture
- Preserve the user's voice in generated Markdown
- Propose file edits explicitly via `<edit>` blocks
- Ask for approval before any write
- Reference tracking data to keep the user accountable

## AI Must Not Do

- Silently overwrite project files
- Invent facts about a project it has not read
- Skip the challenge stage to make the user feel validated
- Generate large specs before earning sufficient context
- Push unnecessary complexity or dependencies
- Treat the user's first answer as final — probe further

## Roadmap

- **M5** (done): provider interface, mock provider, prompt builder, basic chat screen
- **M6** (done): real Anthropic provider, mode taxonomy, ProposedEdit model, approve/reject flow
- **M7** (planned): quiz-mode UX — multiple-choice input widget, answer history, stage tracker
- **M8** (planned): artifact pipeline UI — step-by-step wizard view, artifact completion status
- **M9** (planned): skills system — injectable prompt fragments, skill selection in settings
- **M10** (planned): AI-aware tracking — AI reads event data, surfaces progress insights in chat
