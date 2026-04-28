# Sparkle

Local-first Go TUI for turning rough project sparks into structured, trackable workspaces.

> Status: early. Milestones 1, 2, 4, and 5 are complete; Milestone 3 remains editor-backed for long Markdown sections. See [`docs/roadmap.md`](docs/roadmap.md).

## Run

Requires Go 1.24+.

```sh
go mod tidy
go run ./cmd/sparkle
```

Use an explicit workspace:

```sh
go run ./cmd/sparkle --workspace ./scratch-workspace
```

Seed a workspace with demo sparks:

```sh
go run ./cmd/sparkle sample-data --workspace ./scratch-workspace
```

### Keys

- `tab` / `shift+tab` — cycle tabs
- `1`–`5` — jump to a tab
- `n` — capture a new spark (Sparks tab)
- `e` — edit selected spark's title
- `a` — archive / unarchive selected spark
- `/` — search sparks by title, description, status, or tags
- `c` — clear active spark search
- `?` — show extra keys in the footer
- `j` / `k` (or arrows) — move cursor
- `g` / `G` — jump to first / last
- `o` — open selected project’s `project.md` in `$EDITOR` (Projects tab)
- `O` — open selected project’s `notes.md` in `$EDITOR` (Projects tab)
- `enter` — send an AI guide message (AI tab)
- `enter` — save spark · `esc` — cancel form
- `q` / `ctrl+c` — quit (`q` is yielded to the input field while typing)

On first run Sparkle creates a workspace at `$HOME/sparkle` (override with `$SPARKLE_HOME`):

```
$HOME/sparkle/
  sparks/
  projects/
  .sparkle/
    events/
    config.toml
```

## What's here so far

- `cmd/sparkle/` — entry point, workspace bootstrap
- `internal/domain/` — pure types (Workspace, Spark, Project) + ID generator
- `internal/workspace/` — workspace path resolution and bootstrap
- `internal/storage/markdown/` — frontmatter parser, atomic writer, spark/project stores, JSONL event store
- `internal/tracker/` — pure tracking stats and workspace scanning helpers
- `internal/ai/` — provider interface, deterministic mock provider, prompt builder
- `internal/tui/` — Bubble Tea root, routing, theme-driven rendering
- `internal/tui/msgs/` — shared message envelopes (`ErrorMsg`, `StatusMsg`, `SparksLoadedMsg`)
- `internal/tui/components/{tabs,statusbar,logo}/` — shared UI widgets
- `internal/tui/theme/` — token struct, three palettes (`pastel-dark`, `pastel-light`, `nova`), gradient helper
- `internal/tui/screens/{dashboard,sparks,projects,ai,settings}/`
- `internal/config/` — default-preserving `.sparkle/config.toml` loader

### What works in the UI

- Centered dashboard with a Crush-style diagonal wordmark, live spark/project counts, and the dashboard tracking panel.
- Sparks tab: list view, real disk reads, `n` opens an inline title form, `enter` writes a spark to disk and refreshes the list, `/` searches, and `p` promotes a spark into a project.
- Projects tab: two-pane list/detail view, inline frontmatter edits, project body section previews, and editor-backed `project.md` / `notes.md` access.
- AI tab: chat-like mock-provider flow with loaded project context, ready for a real provider in M6.
- Themed status bar with global hint + error envelope; `tab` / `shift+tab` and `1`–`5` shortcuts on the bar.
- Three palettes ready to switch between (picker UI lands with the settings rewrite).

### Inspirations

UI patterns (diagonal wordmark, compact metadata row, theme-as-tokens) follow [charmbracelet/crush](https://github.com/charmbracelet/crush). Sparkle keeps its own blue palette and simpler Bubble Tea structure.

## Tests

```sh
go test ./...
```

Domain types are unit-tested. More to come as features land — see [`docs/testing.md`](docs/testing.md).

## Roadmap

See [`docs/roadmap.md`](docs/roadmap.md).

1. Local TUI foundation — **shipped** (M1)
2. Sparks (Markdown-backed) — **shipped** (M2)
3. Projects — **in progress** (M3)
4. Dashboard tracking panel, charts, mouse support — **dashboard-owned tracking shipped** (M4)
5. AI-ready architecture with mock provider — **shipped** (M5)
6. Real AI provider (M6)

## Architecture

See [`docs/architecture.md`](docs/architecture.md). Short version:

- Domain layer never imports Bubble Tea, Lip Gloss, or filesystem packages.
- All I/O happens inside `tea.Cmd`; nothing blocks `Update`.
- Markdown-first storage for content; JSONL event log for dashboard tracking; TOML for config.
- Mock AI provider first; real provider lands later behind the same interface.

## Non-goals for v1

Cloud sync, team accounts, plugin marketplace, real-time collaboration, Pomodoro timer, AI fine-tuning. See [`docs/product-spec.md`](docs/product-spec.md).
