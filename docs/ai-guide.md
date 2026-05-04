# Sparkle AI Guide — v2

Read [`v2-vision.md`](v2-vision.md) and [`product-spec.md`](product-spec.md)
first. This file specifies what the AI mentor *is*, how it integrates
with the TUI, how skills and prompts are stored, and how the provider
layer works.

## What the AI is

Sparkle's AI is a **project development mentor** that lives next to the
work. It is bound to one project at a time. It does not exist as a
sibling tab or as a generic chat — it is a panel that shadows the
selected project in the Workspace view (see `tui-ux.md`).

Its job is to drive the user from **half-formed spark** to **structured,
trackable project**. It does this by:

1. Asking one good question at a time.
2. Preferring **structured quizzes** to enumerable answers.
3. **Challenging** weak assumptions before generating structure.
4. Proposing **`<edit>` blocks** for every file change, never writing
   silently.
5. Tracking **pipeline stage** (clarify → structure → challenge →
   architect → expand → finalize) and signaling stage transitions.
6. Reading **tracking data** (words today, streak, days since active)
   to keep the user honest.

The AI is not a chatbot. Free-form chat is a **fallback** for when the
user wants to type something the model didn't ask for.

## The artifact pipeline

Every project flows through six stages, driven by conversation:

```
spark → clarify → structure → challenge → architect → expand → finalize → track
```

| Stage     | AI behaviour                                                   |
|-----------|----------------------------------------------------------------|
| clarify   | One probing question per turn. Quiz preferred. No specs yet.  |
| structure | Build artifacts one at a time. Ask before moving on.          |
| challenge | Find the three most dangerous assumptions. Push back.         |
| architect | Advise on data model, boundaries, async, testing.             |
| expand    | Go deeper on the user-selected section. No new constraints.   |
| finalize  | Produce clean Markdown. Wrap every change in an `<edit>` block.|

The AI does not skip stages. It moves forward only when it has enough
signal — and asks the user to confirm, never auto-advancing.

## Tracked artifacts (7)

| Artifact         | Where it lands                                          |
|------------------|---------------------------------------------------------|
| description      | `# Description` in `project.md`                         |
| problem          | `# Problem` in `project.md`                             |
| audience         | `target_audience` frontmatter + `# Target Audience` body |
| features         | `# Core Features` in `project.md`                       |
| architecture     | `# Architecture` in `project.md`                        |
| roadmap          | `# Roadmap` in `project.md`                             |
| open questions   | `# Open Questions` in `project.md`                      |

The artifact bar at the top of the AI panel reads `artifacts 4/7` and
shows a checkmark per filled artifact.

(v1's "flaws" and "plan" tracked artifacts collapse into "open
questions" and "roadmap" respectively.)

## Conversation modalities

The AI panel accepts user input in three modalities:

1. **Quiz answer** — single keystroke `a`–`f` or arrow + enter. The
   selected choice becomes the user's reply.
2. **Edit approval** — `y` approves, `n` rejects, `tab` cycles between
   pending edits.
3. **Free text** — typed reply, `enter` to send.

Whichever modality is active, the others are blocked. The hint line at
the bottom of the panel shows the active modality's keys.

## System prompt structure

The system prompt is composed at request time from four parts, in this
order:

```
1. Base prompt          ← .sparkle/prompts/system.md (or built-in default)
2. Skill fragment       ← .sparkle/skills/<active-skill>.md
3. Mode block           ← per-mode instruction (clarify/structure/…)
4. Project context      ← title, fields, body sections, tracking stats
```

Each part is rendered into a `strings.Builder` in `BuildSystemPrompt()`.
Missing parts (e.g. no skill, no project loaded) collapse cleanly.

### Base prompt (default)

```
You are Sparkle's local project mentor.
Your job is to help one person turn a rough idea into a structured,
tracked project.

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

The base prompt is editable: `.sparkle/prompts/system.md`. If absent,
Sparkle uses the built-in default. Edits take effect on the next AI
request.

### Mode block

Each mode appends a stanza like:

```
Current mode: CLARIFY — ask precise, probing questions to sharpen
the idea before giving answers.
```

Mode blocks are not user-editable in v2 (they would defeat the
pipeline contract).

### Quiz format

The system prompt teaches Claude how to embed a quiz:

```
<quiz>
Your question here?
a) First option
b) Second option
c) Third option
d) Something else — describe
</quiz>
```

Rules: at most one quiz per response; always include a free-text
fallback ("Something else — …"); never embed a quiz inside an `<edit>`
block.

### Edit format

```
<edit path="projects/sparkle/project.md">
# Description

A local-first Go TUI for…
</edit>
```

The path is workspace-relative. The TUI shows a diff preview before
writing. Approval is explicit, per-file.

### Stage-complete signal

When the AI judges the current stage done:

```
…regular response text…
<stage-complete />
```

The TUI strips the tag and shows "stage done · tab → structure" in the
hint line. The user confirms by pressing tab; the AI never auto-advances.

## Skills — extensible specialisation

A **skill** is a reusable prompt fragment that specialises the AI for a
project type. v2 makes skills **filesystem-backed** so the user can
author their own without recompiling.

### Built-in seed skills

The first launch seeds `.sparkle/skills/` with the v1 hardcoded
fragments as Markdown files:

- `cli-tool.md`
- `web-api.md`
- `library.md`
- `solo-saas.md`
- `open-source.md`

The user can edit, duplicate, or remove any of them.

### File format

```md
---
schema_version: 1
key: web-api
label: Web API
description: REST/GraphQL shape, auth, rate limiting, error contracts
---

Project type: WEB API.
Additional focus:
- REST resource naming…
- Auth strategy: JWT, API keys, OAuth2…
- Rate limiting: per-IP, per-user, per-endpoint…
```

`key` is the unique identifier (kebab-case). `label` is the human name
shown in the picker. `description` is the one-liner shown when a row is
selected. Body is the prompt fragment injected into the system prompt.

### Loading

`internal/skill.Load(workspaceRoot)` walks `.sparkle/skills/*.md`,
parses frontmatter + body, and returns `[]domain.Skill`. The Settings
screen renders this list as a picker. The active skill is stored as
`active_skill = "<key>"` in `config.toml`.

### Selection

The skill picker is a row in the Settings modal. Hovering a row shows
the skill's description; pressing `enter` activates it. Activation
broadcasts `SkillChangedMsg` and the AI panel updates its provider line.

### Validation

Invalid skill files (missing key, missing body, bad frontmatter) are
logged to the status bar on load and skipped. The picker shows them
greyed-out with the error reason.

## Provider interface

```go
package domain

type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
    // Ping does a minimal round-trip to validate auth + connectivity.
    // Used by the Settings "Test connection" action.
    Ping(ctx context.Context) error
}

type CompletionRequest struct {
    Messages []Message
    Context  ProjectContext
    Mode     Mode
    Skill    Skill         // resolved Skill value, includes Body
    System   string        // optional override of the base prompt
}

type CompletionResponse struct {
    Text          string
    ProposedEdits []ProposedEdit
    Quizzes       []Quiz
    StageComplete bool
}
```

`Provider` lives in the domain package. Implementations in
`internal/ai/`:

- `MockProvider` — deterministic local replies, no network.
- `AnthropicProvider` — POST `https://api.anthropic.com/v1/messages`,
  headers `x-api-key`, `anthropic-version: 2023-06-01`.

## Provider selection

Selection logic in `NewRoot`:

1. If `cfg.ResolvedAPIKey()` is empty → `MockProvider`.
2. Else → `AnthropicProvider(key, cfg.AIModel)`.

The active provider name is shown in the AI panel's chrome:

```
claude · sonnet-4-6 · skill: web-api
```

A `mock provider · local` indicator replaces it when no key is set.

## Models

Default: `claude-haiku-4-5` for fast, cheap iteration.

Settings model picker offers:
- `claude-haiku-4-5` — fast, cheap, recommended for iterative quizzing
- `claude-sonnet-4-6` — balanced, recommended for architecture mode
- `claude-opus-4-7` — slow, expensive, reserved for finalize/long
  edits

The picker is a dropdown row in Settings.

## Test connection

The Settings modal's "Test connection" button:

1. Calls `provider.Ping(ctx)` with a 10-second timeout.
2. On success, shows green `✓ connected · model claude-haiku-4-5 ·
   150ms`.
3. On failure, shows the actual error inline (not just "failed"). The
   row stays expanded until the user dismisses it.

## Session persistence

Every AI conversation is logged to
`.sparkle/sessions/<project_id>.jsonl`, one JSON object per line:

```json
{"ts":"2026-04-29T10:15:00+03:00","role":"user","content":"…"}
{"ts":"2026-04-29T10:15:08+03:00","role":"assistant","content":"…","mode":"clarify","stage_complete":false}
{"ts":"2026-04-29T10:15:15+03:00","role":"user","content":"a) …","kind":"quiz_answer"}
```

On project select, the AI panel loads the last 20 turns and presents
them as the conversation history. The user can scroll older turns with
`pgup`.

## Tracking-aware prompts

When the project has tracking data, the system prompt includes a
"Tracking data" section:

```
Tracking data (actual workspace activity):
  Words written today: 320
  Words written this week: 1840
  Active-day streak: 12 days
  Active days this week: 5
```

If `DaysSinceActive > 1`, the prompt adds:

```
  Days since last activity: 4 — consider asking why the project
  stalled.
```

The `MockProvider` already implements this; `AnthropicProvider` must
preserve the same fields.

## What the AI must not do

- Silently overwrite project files.
- Invent facts about a project it has not read.
- Skip the challenge stage to make the user feel validated.
- Generate large specs before earning sufficient context.
- Push unnecessary complexity or dependencies.
- Treat the user's first answer as final — probe further.
- Embed a quiz inside an `<edit>` block.
- Auto-advance pipeline stage without explicit user confirmation.

## What the AI must do

- Ask one question at a time when context is thin.
- Use quizzes to speed up enumerable decisions.
- Challenge weak assumptions before generating structure.
- Help define target audience before architecture.
- Preserve the user's voice in generated Markdown.
- Propose file edits via explicit `<edit>` blocks.
- Ask for approval before any write.
- Reference tracking data to keep the user accountable.

## Implementation notes

- The system prompt builder lives in
  `internal/ai/system_prompt.go`. It accepts the base, skill, mode,
  context, and tracking and produces the final string.
- The Anthropic provider parses `<edit>`, `<quiz>`, and `<stage-complete />`
  blocks in that order via `parseProposedEdits`, `parseQuizBlocks`, and
  `parseStageComplete`. The remaining text is the user-visible message.
- The mock provider keys off the last user message; it does not parse
  blocks but emits structured `Quiz`, `ProposedEdit`, and
  `StageComplete: true` directly.
- Quiz responses with fewer than 2 choices are dropped (logged as a
  warning).

## Roadmap pointer

v1 milestones M5–M10 built the AI plumbing. v2 milestones M14 (panel
embedding) and M15 (filesystem skills + editable prompts + session
persistence + test connection) replace the user-visible surface.
