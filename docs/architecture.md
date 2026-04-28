# Sparkle Architecture

## Core Principle

Separate domain, storage, TUI, tracking, and AI.

The domain layer must not import Bubble Tea, Lip Gloss, or filesystem implementation details.

## Proposed Tree

```txt
cmd/sparkle/
  main.go

internal/
  domain/
    workspace.go
    spark.go
    project.go
    tracking.go
    ai.go

  storage/
    markdown/
      store.go
      parser.go
      writer.go

  tui/
    root.go            // root model, routing, global messages, command dispatch
    commands.go        // shared tea.Cmd helpers
    screens/
      dashboard/
      sparks/
      projects/
      projectdetail/
      ai/
      settings/
    components/
      tabs/
      cards/
      chart/
      modal/
      form/
      statusbar/
    theme/
      theme.go
      palettes.go

  tracker/
    scanner.go         // I/O: walk workspace, read mtimes, count words
    stats.go           // pure: streaks, velocity, daily/weekly aggregates

  ai/
    provider.go
    mock_provider.go
    prompt_builder.go

  config/
    config.go            // .sparkle/config.toml defaults, load, first-run ensure
```

## Layers

### domain

Contains business entities and pure logic:
- Workspace
- Spark
- Project
- TrackingEvent
- Milestone
- Task
- AI request/response abstractions

No terminal rendering. No file paths unless represented as domain data.

### storage

A single `markdown.Store` exposes per-entity methods (LoadWorkspace, ListSparks, SaveSpark, LoadProject, SaveProject, AppendEvent, etc.). Internal `parser.go` and `writer.go` are helpers, not separate package boundaries. Split per-entity stores out later only if any one of them grows past ~300 LOC.

Responsibilities:
- parse frontmatter (YAML)
- write Markdown atomically (temp + rename)
- preserve unknown frontmatter fields and user-written body
- maintain `.sparkle/index.json` as derived cache
- rebuild index from raw Markdown when missing or invalid

All frontmatter carries `schema_version: 1`. Bump and migrate when shape changes.

### tracker

Two files, two concerns:
- `scanner.go` walks the workspace, reads mtimes, computes word deltas, appends events. I/O-heavy; tested via temp dirs.
- `stats.go` is pure: takes events in, returns daily totals, weekly activity, streaks, velocity. Trivially unit-tested.

### tui

Bubble Tea root model and screens:
- `root.go` owns the route, the selected workspace, loaded summaries, and the status-bar error queue.
- Each screen has its own model in `screens/<name>/`.
- Shared widgets live in `components/`. Theme tokens live in `theme/`.

No raw business calculations inside views. No storage calls inside `View`. No blocking I/O in `Update`.

### ai

Provider abstraction, mock provider, prompt builder. Real API integration is deferred — see roadmap M6.

## Bubble Tea Rules

Use:
- `Model`, `Update`, `View`
- typed messages
- `tea.Cmd` for all I/O (loading, saving, scanning, AI calls)

Avoid:
- blocking disk I/O in `Update`
- direct goroutine management in screen code
- storage calls inside `View`
- duplicated state across screens

## Async Pattern

Every `tea.Cmd` returns a typed message. I/O errors flow through one envelope so the status bar handles them uniformly:

```go
type ErrorMsg struct {
    Source string // "load-workspace", "save-spark", ...
    Err    error
}

type LoadWorkspaceMsg struct {
    Workspace domain.Workspace
}

func LoadWorkspaceCmd(path string, store WorkspaceStore) tea.Cmd {
    return func() tea.Msg {
        ws, err := store.Load(path)
        if err != nil {
            return ErrorMsg{Source: "load-workspace", Err: err}
        }
        return LoadWorkspaceMsg{Workspace: ws}
    }
}
```

The root model's `Update` handles `ErrorMsg` once, pushing to the status bar. Per-screen success messages stay narrow.

## Routing

Routes:
- dashboard
- sparks
- projects
- ai
- settings

Tracking appears as a dashboard panel in the primary navigation. The pure
tracking package remains separate from Bubble Tea.

Workspace selection is resolved before Bubble Tea starts through `$SPARKLE_HOME`
or `--workspace <path>`. Settings displays the active workspace and loaded
config; richer in-app switching can land later.

## Performance

For fewer than 100 projects:
- load project summaries on startup
- lazy-load full Markdown content on selection
- cache computed stats
- update indexes after writes
- debounce tracking writes
- avoid full workspace rescans on every keypress

## Safety

Storage writes must:
- be atomic (temp file + rename)
- preserve unknown frontmatter fields
- preserve user-written Markdown body when editing frontmatter
- never overwrite a file whose on-disk `updated_at` is newer than the in-memory copy without surfacing a conflict
