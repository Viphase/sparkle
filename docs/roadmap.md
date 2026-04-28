# Sparkle Roadmap

## Milestone 1 — Local TUI Foundation

Status: complete.

Implemented:
- Go module
- folder structure
- app shell
- workspace selection through `$SPARKLE_HOME` and `--workspace <path>`
- dashboard
- tabs
- themes
- basic keyboard navigation
- config loading at `.sparkle/config.toml`
- sample data generation command: `sparkle sample-data`

No real AI. Mouse support deferred to M4 — keep M1 keyboard-only.

## Milestone 2 — Sparks

Status: complete.

Implemented:
- create spark
- edit spark
- list sparks
- archive spark
- Markdown storage
- search
- promote spark to project (`p` key → creates project, marks spark promoted, routes to Projects tab)

## Milestone 3 — Projects

Status: in progress.

Implemented:
- project list (left pane, keyboard navigation)
- project detail (right pane)
- editable project fields: title, status, GitHub URL, target audience, tags
- status cycling with ← → keys
- open project.md in `$EDITOR` with `o` key
- open notes.md in `$EDITOR` with `O` key
- notes.md bootstrapped alongside every new project
- project detail previews description, architecture, and roadmap sections from project.md
- project count on dashboard
- sample projects in `sparkle sample-data`
- project Markdown storage with frontmatter + default body sections

Remaining:
- architecture, roadmap, and notes remain editor-backed; no inline TUI editor for long Markdown sections yet

## Milestone 4 — Dashboard Tracking, Charts, and Mouse

Status: complete.

Implemented:
- `internal/domain/tracker.go` — TrackingEvent + EventType constants
- `internal/storage/markdown/events.go` — JSONL append/load per project
- `internal/tracker/stats.go` — pure stats: streak, words today/week, active days, 30-day heatmap
- `internal/tracker/scanner.go` — file-change word-count scanner with configurable threshold
- Dashboard tracking panel — weekly bar chart, 30-day activity heatmap, stat cells, real streak + today words from event data
- no standalone Tracker tab, route, or screen; tracking UI is Dashboard content
- Mouse enabled via `tea.EnableMouseCellMotion` on startup

Deferred:
- dashboard-readable `tracker.md` summary regeneration on disk
- Debounced in-app rescanning (currently startup-only)

## Milestone 5 — AI-Ready Architecture

Status: complete.

Implemented:
- AI provider interface
- mock provider
- prompt builder
- basic AI chat screen (single mode)
- loaded project context passed into AI guide requests

No real provider required. `ProposedEdit`, mode taxonomy, and approval flow are deferred to M6 — see `docs/ai-guide.md`.

## Milestone 6 — Real AI Provider

Status: complete.

Implemented:
- `internal/domain/ai.go` — Mode type (clarify/structure/challenge/architect/expand/finalize), ProposedEdit model, updated CompletionRequest/Response
- `internal/ai/anthropic_provider.go` — real Anthropic HTTP provider (POST /v1/messages), injectable HTTPDoer for tests
- `internal/ai/anthropic_provider.go:BuildSystemPrompt` — mode-aware system prompt; finalize mode teaches Claude the `<edit path="…">` block format
- `internal/ai/anthropic_provider.go:parseProposedEdits` — strips edit blocks from response text and returns []ProposedEdit
- Config: `anthropic_api_key` + `ai_model` config keys; `ANTHROPIC_API_KEY` env var wins over file value
- AI screen: mode bar (tab/shift+tab cycle), edit-review overlay (y approve / n reject / tab next), provider indicator ("claude · real provider" vs "mock provider · local only")
- `writeEditCmd` writes approved edits to workspace-relative paths atomically
- Root: wires real provider when key is set, mock otherwise; passes workDir so approved edits can be written
- Tests: 6 new tests in `internal/ai/` covering text response, API errors, edit parsing, context cancellation, system prompt content

## Exclusions for v1

Do not implement:
- team collaboration
- cloud sync
- account system
- real-time multi-user editing
- plugin system
- Pomodoro timer
- complex permissions
- database migration system
- AI fine-tuning
- templates/personas marketplace
