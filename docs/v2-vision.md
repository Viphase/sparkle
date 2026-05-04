# Sparkle v2 — Design North Star

Read this first. Every other v2 doc is a refinement of the principles
here.

## Why a v2

v1 (M1–M10) is a working prototype. It captures sparks, lists projects,
edits frontmatter, and renders deterministic mock-AI replies. But the
holistic experience is a toy:

- Real Anthropic calls work, but only if the user discovers
  `ANTHROPIC_API_KEY` outside the app.
- Stats are unicode bars, not charts.
- The UI is hard-capped at 118×36, leaving big terminals as wasted
  whitespace.
- The AI is a sibling tab — every conversation requires leaving the work
  surface.
- "Skills" are 5 baked-in Go strings. The user cannot author one.
- The app never proactively asks anything; opening it cold gives no
  hint of what to do.

v2 keeps all the storage formats and the domain core but rebuilds the
parts of the surface that read as a stub.

## The four v2 principles

### 1. Seamless, not tabbed

The five-tab navigation is a v1 holdover. v2 collapses Sparks and
Projects into a single **Workspace** view with a left rail of items and
a right working surface. AI is **not a tab** — it is a panel that
shadows whatever the user is editing.

A spark turns into a project turns into a developed plan without ever
re-orienting. The user always knows what context the AI is reading
because that context is on screen, behind the conversation.

### 2. The AI is a guide, not a chatbot

Sparkle's AI mentor's job is to **drive the user forward**. Cold-start
on a new project should fire a question, not wait for one. After a
write the AI should ask the next question. After a stage-complete
signal the AI should advance to the next stage on the user's confirm.

The AI is opinionated:
- Asks one question at a time.
- Prefers structured quizzes for enumerable answers.
- Challenges weak assumptions before structuring anything.
- Refuses to produce specs without context.

Free-form chat is the **fallback** for when the user wants to type
something the model didn't ask for. Quiz answers and approved edits are
the **primary** input modalities.

### 3. Real data, real charts

Sparkle is a tracking tool. Tracking deserves real visualization.

- Use `ntcharts` for every dashboard graph: weekly bars, 30-day
  heatmap, 12-week trend, streak ribbon.
- Show absolute numbers next to charts, not instead of them.
- The dashboard answers four questions on first glance:
  1. Did I work today?
  2. Am I on a streak?
  3. Which project moved this week?
  4. Where is each active project in its pipeline?

If a dashboard panel can't answer one of those, redesign or cut it.

### 4. Configurable, not hardcoded

Skills, prompts, themes, and the AI model live on disk in editable
files, not in Go constants. A power user can:
- Author a new skill in `.sparkle/skills/research-paper.md` and pick it
  from the Settings screen.
- Edit the base system prompt in `.sparkle/prompts/system.md` and have
  Sparkle reload it.
- Drop a new theme palette in `.sparkle/themes/midnight.toml`.
- Switch models from the Settings screen with a "Test connection"
  action that round-trips the API.

## What v2 is NOT

To keep scope honest:
- **Not a multi-user product.** v1's "personal-first" rule stands.
- **Not a sync product.** Markdown on disk; the user's git is the sync.
- **Not a generic chat client.** No tool-use, no MCP, no agentic loops.
- **Not a Pomodoro timer or a kanban board.** v1's exclusion list still
  applies.

## Required stack additions for v2

| Dep                                   | Why                              |
|---------------------------------------|----------------------------------|
| `github.com/NimbleMarkets/ntcharts`   | Real charts (P0 — was always required) |
| `github.com/yuin/goldmark`            | Real Markdown AST for body parse |
| `github.com/charmbracelet/glamour`    | Markdown rendering inside the TUI |
| `github.com/charmbracelet/bubbles/textarea` | Inline long-form editor |

If a dep would land in `indirect` already from one of these, take it.

## Deprecation list (remove on the v2 cut)

These v1 surfaces are explicitly replaced and should not survive v2:

- `internal/tui/screens/ai/` as a top-level route.
- `internal/tui/screens/sparks/` as a top-level route — folded into
  Workspace.
- `internal/tui/screens/projects/` as a top-level route — folded into
  Workspace.
- The `maxAppWidth`/`maxAppHeight` constants in `internal/tui/root.go`.
- All hand-rolled chart functions in
  `internal/tui/screens/dashboard/dashboard.go`.
- Hardcoded `Skill*` constants in `internal/domain/skill.go` (kept as
  default seed skills only, sourced from the same loader).

## Definition of v2 done

- The app launches and on first run runs an interactive setup that
  asks for a workspace path, an API key (skippable), a theme, and one
  starter spark.
- The same app works correctly on a 60×20 terminal, an 80×24, a
  120×36, and a 220×60 terminal — no letterboxing, no truncation.
- The dashboard renders ntcharts-driven bar, heatmap, and line charts.
- The Workspace view shows Sparks and Projects as a single navigable
  list with an embedded detail/AI surface on the right.
- The AI panel can hold a multi-turn conversation about the selected
  project, propose `<edit>` blocks, and have them approved without
  leaving the Workspace view.
- Settings has fields for API key, model, theme, default skill,
  tracking thresholds, with descriptions and validation.
- Custom skills authored under `.sparkle/skills/` show up in the skill
  picker.
- A "Test connection" button in Settings round-trips an Anthropic call
  and reports success or the actual error.
- `go test ./...` passes.
