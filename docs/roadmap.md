# Sparkle Roadmap

## Milestone 1 — Local TUI Foundation

Implement:
- Go module
- folder structure
- app shell
- workspace selection
- dashboard
- tabs
- themes
- basic keyboard navigation
- config loading (TOML)
- sample data generation command

No real AI. Mouse support deferred to M4 — keep M1 keyboard-only.

## Milestone 2 — Sparks

Implement:
- create spark
- edit spark
- list sparks
- archive spark
- promote spark to project
- Markdown storage
- search/filter

## Milestone 3 — Projects

Implement:
- project list
- project detail
- editable project fields
- architecture section
- target audience section
- GitHub URL section
- roadmap section
- notes section

## Milestone 4 — Tracking, Charts, and Mouse

Implement:
- automatic Markdown file scan
- JSONL event log at `.sparkle/events/<project_id>.jsonl`
- word count tracking with the meaningful-event threshold
- daily consistency chart
- weekly activity chart
- streak calculation
- activity summary (regenerated `tracker.md`)
- tracker screen
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
