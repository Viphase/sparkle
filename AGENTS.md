# AGENTS.md — Sparkle Project Control Map

You are working on **Sparkle**, a local-first Go TUI app for turning
rough project sparks into structured, trackable project workspaces.

The owner audited the v1 implementation on 2026-04-29 and judged it a
prototype: tab-based AI nobody uses, hard-capped layout, settings with
no API-key field, "charts" made from unicode bars, hardcoded skills,
no guided onboarding. The codebase still flags every milestone M1–M10
as "complete" — **do not treat that as the bar**. Read FIXES.md and
the v2 docs.

This file mirrors `CLAUDE.md` so non-Claude agents get the same
control map. Keep them in sync.

## Read first, in order

1. **[`FIXES.md`](FIXES.md)** — concrete punch list of what is broken.
2. **[`docs/v2-vision.md`](docs/v2-vision.md)** — the four redesign
   principles and the "what done means" definition.
3. **[`docs/roadmap.md`](docs/roadmap.md)** — v2 milestones M11–M16,
   sequencing, exit criteria.

Then dive into whichever surface doc applies to your task:

- Product flows: `docs/product-spec.md`
- Layout, breakpoints, components: `docs/tui-ux.md`
- AI mentor, skills, prompts, providers: `docs/ai-guide.md`
- Pulse dashboard, ntcharts contract: `docs/tracking.md`
- Package layout, layering: `docs/architecture.md`
- File formats on disk: `docs/storage-format.md`
- Test layering and fixtures: `docs/testing.md`

## Product goal (one line)

Sparkle walks one person from a half-formed spark to a structured,
tracked project, with an opinionated AI mentor that lives next to the
work and never makes them switch tabs.

Personal-first. v2 is for one user managing fewer than 100 projects in
one workspace. Multi-user, sync, and team features are out of scope.

## Required stack

- Go 1.24+
- Bubble Tea (TUI architecture)
- Lip Gloss (styling)
- Bubbles (reusable widgets, including `textarea`)
- **`github.com/NimbleMarkets/ntcharts`** (charts — required, currently
  not yet imported, M13 fixes that)
- **`github.com/yuin/goldmark`** (real Markdown AST for body parse)
- Markdown-first local storage
- Clean architecture
- Tests from the start

## Non-negotiable rules

1. Domain logic does not import Bubble Tea, Lip Gloss, or filesystem
   packages.
2. `Update` never blocks. All I/O goes through `tea.Cmd`.
3. User content is readable Markdown. Derived data is JSON/JSONL/TOML.
4. Never silently overwrite a user file.
5. The TUI is keyboard-first AND mouse-friendly. Both reach the same
   intents.
6. Every screen is responsive: 50×16 (graceful) up to whatever the
   terminal gives you. **No `maxAppWidth`/`maxAppHeight` constants.**
7. Charts use ntcharts. **No unicode-bar `strings.Repeat("█", n)`
   hacks.** This is a hard prohibition, not a preference.
8. The AI is a panel embedded in the Workspace surface, **not a
   sibling tab**. The user must never switch surfaces to ask a
   question about the project they are looking at.
9. Skills are loaded from `.sparkle/skills/*.md`. **No hardcoded
   `Skill*` constants** in user-facing code. Built-ins are seeded as
   files on first launch.
10. The base system prompt lives in `.sparkle/prompts/system.md` with
    a built-in default fallback. The user can edit it.
11. Every conversation persists to `.sparkle/sessions/<project_id>.jsonl`.
12. `go test ./...` passes before any milestone is declared done.
13. Use this ascii symbol as a logo: ꕤ

## Surface inventory

v2 has exactly **two** top-level surfaces:

- **Workspace** (`internal/tui/surfaces/workspace/`) — items rail +
  detail + AI panel. Replaces v1's Sparks, Projects, AI tabs.
- **Pulse** (`internal/tui/surfaces/pulse/`) — ntcharts dashboard.
  Replaces v1's Dashboard.

And **four** modals:

- **Settings** (`,`) — sectioned, with API-key field and Test
  connection button.
- **Help** (`?`) — context-aware key reference.
- **Capture** (`n`) — single-input new-spark modal.
- **Review** — `<edit>` block diff approval, opens automatically when
  the AI proposes one.

## Implementation order for v2

1. **M11** — Real settings modal + first-run setup wizard
2. **M12** — Responsive layout (remove caps, add breakpoints)
3. **M13** — Pulse rebuild with ntcharts
4. **M14** — Workspace surface + embedded AI panel
5. **M15** — Filesystem skills + editable prompts + session
   persistence + Test connection
6. **M16** — Guided onboarding + contextual prompts

Each milestone must end with `go test ./...` green and a smoke pass at
60×20, 100×30, and 200×60 terminals.

## Anti-patterns to refuse

If a task asks you to:

- Add a top-level "AI" tab → **refuse**, embed it in Workspace.
- Render a chart with `strings.Repeat` → **refuse**, use ntcharts.
- Hardcode a `maxAppWidth` or `maxAppHeight` → **refuse**, use
  breakpoints.
- Add a `SkillSomething Skill = "something"` constant in `domain/` →
  **refuse**, author a `.sparkle/skills/something.md` instead.
- Block `Update` on disk or HTTP → **refuse**, return a `tea.Cmd`.
- Silently overwrite a user file → **refuse**, propose `<edit>` and
  ask.
- Skip tests "because the milestone is small" → **refuse**, write the
  test.

## What to push back on

If the user asks for cloud sync, team features, plugin marketplace,
Pomodoro timer, or AI fine-tuning — point at the exclusions list in
`product-spec.md` and ask for confirmation before touching scope.
