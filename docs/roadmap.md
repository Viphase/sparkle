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
- notes.md bootstrapped alongside every new project
- project count on dashboard
- sample projects in `sparkle sample-data`
- project Markdown storage with frontmatter + default body sections

Remaining:
- architecture, roadmap, and notes sections are in the project.md body — editable via `o` ($EDITOR); no inline TUI editor yet

## Milestone 4 — Tracking, Charts, and Mouse

Implement:
- automatic Markdown file scan
- JSONL event log at `.sparkle/events/<project_id>.jsonl`
- word count tracking with the meaningful-event threshold
- daily consistency chart
- weekly activity chart
- streak calculation
- activity summary (regenerated `tracker.md`)
- dashboard tracking panel
- mouse support across existing screens

## Milestone 5 — AI-Ready Architecture

Implement:
- AI provider interface
- mock provider
- prompt builder
- basic AI chat screen (single mode)

No real provider required. `ProposedEdit`, mode taxonomy, and approval flow are deferred to M6 — see `docs/ai-guide.md`.

## Milestone 6 — Real AI Provider

Implement later:
- API key config
- API provider implementation
- local project context reader
- file-change preview
- explicit approval before writes

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
