# Sparkle Architecture — v2

Read [`v2-vision.md`](v2-vision.md) first. This file describes the
package layout, layering, async pattern, and routing for v2.

## Core principle (unchanged)

Separate domain, storage, TUI, tracking services, and AI. Domain logic
must not import Bubble Tea, Lip Gloss, or filesystem implementation
details.

## Package layout

```txt
cmd/sparkle/
  main.go                  // workspace bootstrap, first-run wizard launcher

internal/
  domain/
    workspace.go
    spark.go
    project.go
    tracker.go
    ai.go                  // Provider, CompletionRequest/Response, Mode, Quiz, ProposedEdit
    skill.go               // Skill struct (file-loaded, no hardcoded constants)

  storage/
    markdown/
      store.go             // facade
      parser.go
      writer.go
      project.go
      spark.go
      events.go
      sessions.go          // NEW: AI conversation log (.sparkle/sessions/)
      skills.go            // NEW: load .sparkle/skills/*.md
      themes.go            // NEW: load .sparkle/themes/*.toml

  tracker/
    scanner.go             // walks workspace, reads mtimes, computes deltas
    watcher.go             // NEW: debounced rescan during a session
    stats.go               // pure: streaks, velocity, daily/weekly aggregates

  ai/
    provider.go            // package-level helpers
    mock_provider.go
    anthropic_provider.go
    system_prompt.go       // NEW: BuildSystemPrompt extracted to its own file
    quiz.go
    stage.go
    edits.go               // NEW: parseProposedEdits + diff helpers

  config/
    config.go              // .sparkle/config.toml load/save with API-key field
    setup.go               // NEW: first-run wizard model

  tui/
    root.go                // root model, routing, global messages
    commands.go            // shared tea.Cmd helpers
    msgs/                  // typed message envelopes
    surfaces/              // RENAMED from screens/: top-level surfaces
      workspace/           // NEW: spark+project unified view (replaces sparks+projects)
        workspace.go
        rail.go            // left rail of items
        detail.go          // structured fields + section preview
        editor.go          // inline textarea for project.md sections
        ai_panel.go        // embedded AI conversation (replaces ai/ screen)
      pulse/               // RENAMED from dashboard/: ntcharts-driven dashboard
        pulse.go
        cards.go
        weekly_chart.go    // ntcharts BarChart wrapper
        heatmap.go         // ntcharts Heatmap wrapper
        trend.go           // ntcharts Sparkline wrapper
        pipeline_row.go    // per-project pipeline row
    modals/                // NEW: modal overlays
      settings/
        settings.go
        provider_section.go
        skills_section.go
        tracking_section.go
        appearance_section.go
      help/
        help.go            // context-aware ? overlay
      capture/
        capture.go         // n new spark
      review/
        review.go          // <edit> diff approval
    components/            // shared widgets
      chrome/              // NEW: top app strip
      modebar/             // NEW: 1 Workspace / 2 Pulse switcher
      statusbar/
      logo/
      rail/                // NEW: left list pane
      card/                // NEW: hero stat card
      chart/               // NEW: ntcharts wrappers
      pipeline/            // NEW: 6-stage indicator
      diff/                // NEW: markdown diff renderer
      input/
      textarea/            // NEW: multi-line markdown editor
      modal/               // NEW: centered overlay
    theme/
      theme.go
      palettes.go          // built-in: pastel-dark, pastel-light, nova
      gradient.go
      loader.go            // NEW: load .sparkle/themes/*.toml

  workspace/
    workspace.go           // path resolution, bootstrap, .sparkle/ scaffolding
```

Notes:

- `internal/tui/screens/` is renamed `internal/tui/surfaces/` to
  reflect that the v2 model is two surfaces, not five screens.
- `internal/tui/screens/ai/` is **removed**; the embedded AI panel
  lives in `surfaces/workspace/ai_panel.go`.
- `internal/tui/screens/sparks/` and
  `internal/tui/screens/projects/` are **removed**; their
  responsibilities fold into `surfaces/workspace/`.
- `internal/tui/screens/dashboard/` is **renamed** to
  `surfaces/pulse/`.
- `internal/tui/screens/settings/` is **removed**; settings becomes a
  modal at `internal/tui/modals/settings/`.

## Layers

### domain

Business entities and pure logic:
- `Workspace`, `Spark`, `Project`
- `TrackingEvent`, `TrackingStats`, `Mode`
- `Quiz`, `ProposedEdit`, `CompletionRequest`, `CompletionResponse`
- `Skill` — struct with `Key, Label, Description, Body string` (no
  hardcoded constants)
- `Provider interface` with `Complete()` and `Ping()`

No filesystem, no rendering, no networking. `Skill` is a value type —
loading happens in storage.

### storage

Single `markdown.Store` exposing per-entity methods. Internal helpers
(`parser.go`, `writer.go`) are not separate package boundaries.

New for v2:
- `sessions.go` — append-only `.sparkle/sessions/<project_id>.jsonl`
- `skills.go` — read `.sparkle/skills/*.md`, parse frontmatter + body
- `themes.go` — read `.sparkle/themes/*.toml`, parse to `Theme` struct

All writes are atomic (temp + rename). All frontmatter carries
`schema_version: 1`.

### tracker

Two concerns:
- `scanner.go` (I/O) walks the workspace, reads mtimes, computes
  word deltas, appends events.
- `watcher.go` (I/O, NEW) runs on a debounce during the session,
  triggered by file-write Cmds, never inside `Update`.
- `stats.go` (pure) takes events in, returns daily totals, weekly
  activity, streaks, velocity, 12-week trend, pipeline stage.

### ai

- `Provider` interface implementations: `MockProvider`,
  `AnthropicProvider`.
- `system_prompt.go` composes base + skill + mode + context +
  tracking.
- `quiz.go`, `stage.go`, `edits.go` parse the response blocks.

### tui

Bubble Tea root + surfaces + modals + components.

Root owns:
- the route (Workspace or Pulse)
- the workspace + store
- the loaded summaries
- the modal stack (settings / help / capture / review)
- the global status-bar error queue

Surfaces own their own model. Components are pure renderers
(no `tea.Cmd`s inside).

### config

- `config.toml` loader + saver
- `setup.go` (NEW) — first-run wizard model. Runs before the main TUI
  if `.sparkle/config.toml` is missing.

## Bubble Tea rules (unchanged)

Use `Model`, `Update`, `View`, typed messages, `tea.Cmd` for I/O.

Avoid:
- blocking disk I/O in `Update`
- direct goroutine management in screen code
- storage calls inside `View`
- duplicated state across surfaces

## Async pattern (unchanged)

Every `tea.Cmd` returns a typed message. I/O errors flow through one
envelope:

```go
type ErrorMsg struct {
    Source string
    Err    error
}
```

The root model handles `ErrorMsg` once, pushing to the status bar.

## Routing

Two top-level routes:
- `RouteWorkspace`
- `RoutePulse`

Modals stack on top of the active route; pressing `esc` pops the top
modal. Modals are: `Settings`, `Help`, `Capture`, `Review`.

The root maintains a `[]Modal` stack. The active modal absorbs all
keystrokes except `esc` (close), `ctrl+c` (quit), and globals it
explicitly forwards.

## Modal contract

```go
type Modal interface {
    Init() tea.Cmd
    Update(tea.Msg) (Modal, tea.Cmd)
    View(width, height int) string
    Title() string
}
```

Modals render centered with a backdrop. The backdrop is the parent
surface dimmed (lipgloss `Faint(true)`), painted once and cached on
the root.

## Workspace bootstrap

`internal/workspace/workspace.go` resolves the workspace root from:
1. `--workspace <path>` CLI flag
2. `SPARKLE_HOME` env var
3. `~/sparkle` (default)

If the workspace does not contain `.sparkle/config.toml`, the
first-run wizard runs in a separate Bubble Tea program before the
main TUI starts. After the wizard exits, the main TUI launches with
the wizard's output.

## Performance

For fewer than 100 projects:
- Load summaries on startup (lazy-load full Markdown on selection).
- Cache computed stats.
- Update indexes after writes.
- Debounce tracking scans (≥ 2s idle).
- Avoid full workspace rescans on every keypress.
- ntcharts canvases recreated on resize only.
- Logo re-rendered on resize only (cached by width).

## Safety

Storage writes must:
- be atomic (temp file + rename)
- preserve unknown frontmatter fields
- preserve user-written Markdown body when editing frontmatter
- never overwrite a file whose on-disk `updated_at` is newer than the
  in-memory copy without surfacing a conflict

## Removed v1 patterns

- Hardcoded `maxAppWidth` / `maxAppHeight` constants.
- Tab-based five-route navigation.
- AI as a sibling screen.
- Hand-rolled chart functions.
- `Skill` as Go constants.
