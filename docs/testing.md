# Sparkle Testing Strategy — v2

## Hard requirement

`go test ./...` must pass before any milestone is considered done.
Build with tests from the start; do not accept "it works on my
machine."

## Layered testing

### Domain layer (`internal/domain/`)

Pure types and pure functions. Tests must not import:
- Bubble Tea
- Lip Gloss
- Filesystem implementation packages
- HTTP client packages

Cover:
- Spark / Project state transitions and validation
- Mode taxonomy (`AllModes`, labels)
- Quiz / ProposedEdit / CompletionRequest helpers
- ArtifactStatus computation
- Skill struct equality, validation predicates
- TrackingStats arithmetic helpers

### Storage layer (`internal/storage/markdown/`)

Use `t.TempDir()` for isolation. Cover:
- Spark write/read round trip
- Project write/read round trip
- Markdown body preserved across frontmatter edits
- Unknown frontmatter fields preserved on save
- Atomic write survives a simulated mid-write crash (write to temp
  fails — no partial original)
- Index rebuild from raw Markdown when `.sparkle/index.json` is
  missing, corrupt, or schema-mismatched
- Conflict detection when on-disk `updated_at` is newer than the
  loaded copy
- Invalid Markdown recovers gracefully

NEW for v2:
- **Sessions:** append/read JSONL round trip; bad lines tolerated
- **Skills:** load fixtures with valid + invalid frontmatter; invalid
  ones reported, valid ones returned in deterministic order
- **System prompt:** `system.md` round trip; missing file returns
  built-in default

### Tracker layer (`internal/tracker/`)

`stats.go` — pure, easy:
- Word counts
- Daily totals
- Weekly totals
- Streaks (current + longest)
- Active days in week
- Velocity (words / week)
- 12-week trend (`Last12WeeksWords`)
- Pipeline stage from events

Synthesise events with literal slices. No filesystem.

`scanner.go` — integration via `t.TempDir()`:
- Unchanged file detection — skip word-count work when mtime unchanged
- Threshold suppresses small word deltas
- JSONL append survives interleaved scans
- Partial event-log rebuild from mtimes when JSONL deleted
- Touch window suppresses file_touched bursts

NEW for v2:
- **Watcher:** debounce contract — multiple writes within window emit
  one event; idle resets the timer

### AI layer (`internal/ai/`)

Use the injectable `HTTPDoer` to avoid real network calls.

Cover:
- Mock provider returns deterministic structured responses for known
  keywords
- Mock returns Quiz objects directly (no parsing)
- Mock returns `StageComplete=true` on stage-end keywords
- Anthropic provider builds the right request body (model, system,
  messages, max_tokens)
- Anthropic provider parses `<edit>`, `<quiz>`, `<stage-complete />`
  blocks correctly
- Anthropic provider surfaces API errors as `error` returns
- `ctx.Done()` is honored on cancel
- Skill fragment is injected into the system prompt between base and
  mode block
- `SkillNone` (or empty skill) leaves the prompt unchanged
- Tracking-data section appears when stats are non-zero, suppressed
  otherwise

NEW for v2:
- **Provider.Ping:** mock returns nil; Anthropic POSTs a minimal
  request and propagates HTTP errors
- **System prompt builder:** reads `.sparkle/prompts/system.md` when
  present, falls back to the built-in default

### TUI layer (`internal/tui/`)

Practical, not exhaustive. Cover:
- Route changes (where applicable — v2 has only Workspace and Pulse)
- Modal stack push/pop on `,` / `?` / `n` / `esc`
- Message handling for typed envelopes (`SparksLoadedMsg`,
  `ProjectsLoadedMsg`, `TrackingLoadedMsg`, `SkillChangedMsg`,
  `ProjectContextMsg`, `SparkPromotedMsg`)
- No panic at minimum size (50×16)
- No panic at large size (220×60)

NEW for v2:
- **Workspace surface:**
  - rail rendering with mixed sparks + projects
  - detail pane field editing round trip
  - inline section editor save/cancel
  - AI panel populated on item select
  - promote spark inline updates rail + AI context
- **Pulse surface:**
  - 4 hero cards render with correct values from `TrackingStats`
  - empty event log renders without panic
  - all 5 panels present at wide width
  - graceful collapse below narrow breakpoint
  - ntcharts panels detected by their characteristic borders + caption
- **Settings modal:**
  - field navigation
  - API-key value masking
  - Test connection success path (mock provider)
  - Test connection failure path (injected error)
- **First-run wizard:**
  - all 5 steps complete
  - skipping API-key uses mock provider
  - resulting `config.toml` parses back to expected `Config`
- **Responsive contract:**
  - Workspace at 50, 80, 120, 200 columns — assert breakpoint-correct
    column count
  - Pulse at the same widths — assert panel-collapse rules

Do not over-test visual rendering character-by-character; assert on:
- Border presence
- Caption / label presence
- Number / count correctness
- Hint text correctness
- Cursor position when meaningful

## Test fixtures

`testdata/` directories per package. Use them for:
- Sample skill files (valid + invalid)
- Sample sessions
- Sample event logs
- Sample workspace trees

Load via `os.ReadFile(filepath.Join("testdata", name))`. Update with
`golden file` patterns where rendered output is asserted.

## Coverage discipline

Coverage is not a quality gate, but the following surfaces have a
floor:

| Package                            | Floor coverage |
|------------------------------------|----------------|
| `internal/domain/`                 | 90%            |
| `internal/tracker/` (pure stats.go)| 90%            |
| `internal/storage/markdown/`       | 80%            |
| `internal/ai/`                     | 70%            |
| `internal/config/`                 | 80%            |
| `internal/tui/...`                 | 50%            |

Run `go test -coverprofile=cover.out ./...` and check.

## Test naming

`TestX_When_Y_ThenZ` for unit tests where the behaviour matters. `Test_X`
is fine for trivial round-trips. Subtest names should describe inputs,
not implementation.

## What NOT to test

- Lipgloss rendering character-by-character (style choices change)
- ntcharts internal canvas pixels (library responsibility)
- Live network calls to Anthropic (use `HTTPDoer`)
- `$EDITOR` shell-out behaviour (treat as opaque)
- Color values inside themes (assert keys exist, not their hex)

## Pre-commit gate

Before declaring a milestone done:

1. `go vet ./...` — no warnings
2. `gofmt -l .` — no diffs
3. `go test ./...` — all green
4. Manual smoke at three terminal sizes: 60×20, 100×30, 200×60
5. Manual smoke through the milestone's primary flow

## Test dependencies

Allowed in tests:
- `testing/quick` for property tests
- `golang.org/x/text/...` indirectly via colorprofile
- `t.TempDir()` for filesystem tests
- `httptest.NewServer` for HTTP-level tests of `AnthropicProvider`

Do not pull in `testify`, `gomega`, or any other matcher framework
unless a milestone explicitly authorizes it. Standard-library testing
is the bar.
