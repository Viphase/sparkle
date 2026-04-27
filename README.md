# Sparkle

Local-first Go TUI for turning rough project sparks into structured, trackable workspaces.

> Status: very early. Milestone 1 — local TUI foundation. See [`docs/roadmap.md`](docs/roadmap.md).

## Run

Requires Go 1.24+.

```sh
go mod tidy
go run ./cmd/sparkle
```

### Keys

- `tab` / `shift+tab` — cycle tabs
- `1`–`6` — jump to a tab
- `n` — capture a new spark (Sparks tab)
- `e` — edit selected spark's title
- `a` — archive / unarchive selected spark
- `j` / `k` (or arrows) — move cursor
- `g` / `G` — jump to first / last
- `enter` — save spark · `esc` — cancel form
- `q` / `ctrl+c` — quit (`q` is yielded to the input field while typing)

On first run Sparkle creates a workspace at `$HOME/sparkle` (override with `$SPARKLE_HOME`):

```
$HOME/sparkle/
  sparks/
  projects/
  .sparkle/
    events/
```

## What's here so far

- `cmd/sparkle/` — entry point, workspace bootstrap
- `internal/domain/` — pure types (Workspace, Spark, Project) + ID generator
- `internal/workspace/` — workspace path resolution and bootstrap
- `internal/storage/markdown/` — frontmatter parser, atomic writer, spark store
- `internal/tui/` — Bubble Tea root, routing, theme-driven rendering
- `internal/tui/msgs/` — shared message envelopes (`ErrorMsg`, `StatusMsg`, `SparksLoadedMsg`)
- `internal/tui/components/{tabs,statusbar,logo}/` — shared UI widgets
- `internal/tui/theme/` — token struct, three palettes (`pastel-dark`, `pastel-light`, `nova`), gradient helper
- `internal/tui/screens/{dashboard,sparks,projects,tracker,ai,settings}/`
- `internal/config/` — config struct (TOML loader still to follow)

### What works in the UI

- Centered dashboard with a gradient `Sparkle` wordmark, sparkle field decoration, and live spark counts.
- Sparks tab: list view, real disk reads, `n` opens an inline title form, `enter` writes a spark to disk and refreshes the list.
- Themed status bar with global hint + error envelope; `tab` / `shift+tab` and `1`–`6` shortcuts on the bar.
- Three palettes ready to switch between (picker UI lands with the settings rewrite).

### Inspirations

UI patterns (gradient wordmark, decorative field around the title, theme-as-tokens) follow [charmbracelet/crush](https://github.com/charmbracelet/crush). Sparkle replaces Crush's diagonals with sparkle glyphs and uses its own palette.

## Tests

```sh
go test ./...
```

Domain types are unit-tested. More to come as features land — see [`docs/testing.md`](docs/testing.md).

## Roadmap

See [`docs/roadmap.md`](docs/roadmap.md).

1. Local TUI foundation — **shipped** (M1)
2. Sparks (Markdown-backed) — **list + create + edit + archive shipped**, search next (M2)
3. Projects (M3)
4. Tracking, charts, mouse support (M4)
5. AI-ready architecture with mock provider (M5)
6. Real AI provider (M6)

## Architecture

See [`docs/architecture.md`](docs/architecture.md). Short version:

- Domain layer never imports Bubble Tea, Lip Gloss, or filesystem packages.
- All I/O happens inside `tea.Cmd`; nothing blocks `Update`.
- Markdown-first storage for content; JSONL event log for tracker; TOML for config.
- Mock AI provider first; real provider lands later behind the same interface.

## Non-goals for v1

Cloud sync, team accounts, plugin marketplace, real-time collaboration, Pomodoro timer, AI fine-tuning. See [`docs/product-spec.md`](docs/product-spec.md).
