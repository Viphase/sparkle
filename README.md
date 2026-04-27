# Sparkle

Local-first Go TUI for turning rough project sparks into structured, trackable workspaces.

> Status: very early. Milestone 1 — local TUI foundation. See [`docs/roadmap.md`](docs/roadmap.md).

## Run

Requires Go 1.24+.

```sh
go mod tidy
go run ./cmd/sparkle
```

Press `q` to quit.

## What's here so far

- `cmd/sparkle/` — entry point
- `internal/domain/` — pure domain types (Workspace, Spark, Project)
- `internal/tui/` — Bubble Tea root model
- `internal/tui/theme/` — color tokens (no hardcoded colors in views)
- `internal/config/` — config struct (TOML loader to follow)

Not implemented yet: storage, sparks/projects screens, tracker, AI. Those land in later milestones.

## Tests

```sh
go test ./...
```

Domain types are unit-tested. More to come as features land — see [`docs/testing.md`](docs/testing.md).

## Roadmap

See [`docs/roadmap.md`](docs/roadmap.md).

1. Local TUI foundation — **in progress**
2. Sparks (Markdown-backed)
3. Projects
4. Tracking, charts, mouse support
5. AI-ready architecture with mock provider
6. Real AI provider

## Architecture

See [`docs/architecture.md`](docs/architecture.md). Short version:

- Domain layer never imports Bubble Tea, Lip Gloss, or filesystem packages.
- All I/O happens inside `tea.Cmd`; nothing blocks `Update`.
- Markdown-first storage for content; JSONL event log for tracker; TOML for config.
- Mock AI provider first; real provider lands later behind the same interface.

## Non-goals for v1

Cloud sync, team accounts, plugin marketplace, real-time collaboration, Pomodoro timer, AI fine-tuning. See [`docs/product-spec.md`](docs/product-spec.md).
