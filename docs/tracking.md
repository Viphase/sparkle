# Sparkle Tracking & Pulse — v2

Read [`v2-vision.md`](v2-vision.md) and [`tui-ux.md`](tui-ux.md) first.
This file specifies what is tracked, how it is stored, and — critically
— how the **Pulse** dashboard renders it.

## Goals

1. Track user momentum without manual input.
2. Show momentum in **real charts** built with `ntcharts`. Unicode
   bars are not acceptable; the v1 implementation
   (`strings.Repeat("█", n)`) is deprecated.
3. Answer four questions on first glance:
   - Did I work today?
   - Am I on a streak?
   - Which project moved this week?
   - Where is each active project in its pipeline?

## What is tracked

### Automatic (no user input)

- Markdown file mtime changes
- Word count deltas (only when `|delta| ≥ words_threshold`)
- File touches (rate-limited per file per `touch_window_secs`)
- Session windows (closed when no edits for ≥ `session_idle_minutes`)
- Project activity streaks
- Per-project velocity (words/week)

### Semi-manual

- Milestone completion (`task_completed`, `milestone_completed`)
- Deadlines (`deadline_added`)
- Mood / focus notes (`mood_logged`, `note_added`)

These are emitted by the AI panel when the user acknowledges a
stage-complete signal or approves a roadmap edit.

## Event storage

Append-only JSONL, one file per project, at
`.sparkle/events/<project_id>.jsonl`:

```json
{"ts":"2026-04-29T10:15:00+03:00","type":"words_added","value":420,"source":"auto","note":"project.md"}
{"ts":"2026-04-29T10:50:00+03:00","type":"session_minutes","value":35,"source":"auto"}
{"ts":"2026-04-29T18:00:00+03:00","type":"milestone_completed","value":1,"source":"manual","note":"M2 done"}
```

Required: `ts`, `type`, `value`, `source`. Optional: `note`.

The file is treated as derived data — partially rebuildable from file
mtimes if lost.

### Event types

```go
const (
    EventWordsAdded         EventType = "words_added"
    EventWordsRemoved       EventType = "words_removed"
    EventFileTouched        EventType = "file_touched"
    EventSessionMinutes     EventType = "session_minutes"
    EventTaskCompleted      EventType = "task_completed"
    EventMilestoneComplete  EventType = "milestone_completed"
    EventDeadlineAdded      EventType = "deadline_added"
    EventMoodLogged         EventType = "mood_logged"
    EventNoteAdded          EventType = "note_added"
    EventStageAdvanced      EventType = "stage_advanced"   // new in v2
    EventEditApproved       EventType = "edit_approved"    // new in v2
)
```

`stage_advanced` and `edit_approved` are new in v2: emitted by the AI
panel so Pulse can correlate AI activity with raw word counts.

## Thresholds (config-tunable)

```toml
words_threshold      = 10    # minimum |delta| for words events
touch_window_secs    = 300   # 5 minutes between file_touched events
session_idle_minutes = 10    # idle gap that closes a session window
streak_grace_hours   = 4     # tolerance for streak day boundaries
```

Defaults live in `config.Defaults()`. Settings UI exposes all four.

## Scanning lifecycle

### On launch

1. Load `.sparkle/index.json` (cached state).
2. Walk `projects/**` and `sparks/**` reading mtimes.
3. For each file whose mtime changed since last scan, recount words.
4. Emit events that pass the meaningful-event threshold.
5. Update the index.

### During the session (new in v2)

A debounced rescanner runs every 2 seconds of idle:

- Triggered by file-write Cmds and the editor-shell-out flow.
- Lives in `internal/tracker/watcher.go`.
- Emits `tea.Cmd`-driven events; never blocks `Update`.

### On exit

The session-minutes window is closed for any active sessions and an
event appended.

## Pure stats (no I/O)

`internal/tracker/stats.go` — pure functions over `[]TrackingEvent`:

```go
func Compute(events []TrackingEvent, now time.Time) TrackingStats
func WeeklyWordsByDay(events []TrackingEvent, now time.Time) [7]int
func Last30DaysActivity(events []TrackingEvent, now time.Time) [30]bool
func Last12WeeksWords(events []TrackingEvent, now time.Time) [12]int   // new
func ProjectVelocity(events []TrackingEvent, now time.Time) float64    // new (words/week)
func PipelineStage(events []TrackingEvent) Mode                        // new (last stage_advanced)
```

`TrackingStats` carries:

```go
type TrackingStats struct {
    TodayWords     int
    WeekWords      int
    CurrentStreak  int
    LongestStreak  int
    ActiveDaysWeek int
    LastActive     time.Time
    SessionMinutesToday int   // new
    FilesTouchedToday   int   // new
}
```

## Pulse dashboard panels

Pulse is the second top-level surface. Layout in
[`tui-ux.md`](tui-ux.md). Each panel below maps to one Pulse section.

### Panel 1 — Today (4 hero cards)

| Card           | Source                                       |
|----------------|----------------------------------------------|
| `words today`  | `TrackingStats.TodayWords`                   |
| `files touched`| `TrackingStats.FilesTouchedToday`            |
| `streak days`  | `TrackingStats.CurrentStreak`                |
| `active`       | count of projects with `Status=active`       |

Each card is a bordered `card` component (see `tui-ux.md`). No chart.

### Panel 2 — Words this week (ntcharts BarChart)

Library: `github.com/NimbleMarkets/ntcharts/barchart`.

```go
import "github.com/NimbleMarkets/ntcharts/barchart"

bar := barchart.New(width, height,
    barchart.WithBarGap(1),
    barchart.WithStyle(theme.Accent),
)
bar.PushAll(WeeklyWordsByDay(events, now), labels=[]string{"M","T","W","T","F","S","S"})
bar.Draw()
```

Caption: `words added · this week`. Y-axis auto-scales to the max bar.
Day labels under each bar. Values to the right of each bar.

Below 60 columns: collapses to a 7-line list `Mon  ▆ 320 / Tue ▇ 410 …`.

### Panel 3 — Activity heatmap (ntcharts Heatmap)

Library: `github.com/NimbleMarkets/ntcharts/heatmap`.

A real calendar grid: 7 rows (days of week) × 5 columns (weeks). Cells
filled when `activeDays[date]` is true. Color intensity by word count
(quartile bucket).

Caption: `activity · last 35 days`. Today is the bottom-right cell.

Below 80 columns: collapses to a 5×7 grid with smaller cells.

### Panel 4 — 12-week trend (ntcharts Sparkline)

Library: `github.com/NimbleMarkets/ntcharts/sparkline`.

Single-row sparkline of `Last12WeeksWords(events, now)`. Shows
trend at a glance.

Caption: `12-week words trend`. Below the sparkline, a one-liner with
the absolute average: `avg 1240 words/wk · peak 2310 (week of 2026-03-30)`.

### Panel 5 — Active projects pipeline & velocity

Custom row renderer (no ntcharts). For each active project:

```
Sparkle      ●─●─●─○─·─·    260 wpw · streak 12d
taproot      ●─○─·─·─·─·     40 wpw · streak  0d
novel-tracker●─●─●─●─●─●     ✓ shipped
```

`●` = visited stage, `○` = active stage, `·` = future stage.
`wpw` = words per week (rounded). `streak` = `CurrentStreak` for that
project's events.

Sort: most recently active first. Truncate at 5 projects with a "+N
more" footer when more exist.

## Per-project tracking on Workspace

The Workspace detail pane shows a one-line tracking summary above the
fields:

```
streak 12d · 320 today · 1840 this week · last active 2 hours ago
```

No charts in the Workspace view. Charts live on Pulse.

## Privacy

All tracking is local. Events are never sent to AI providers
**unless** the user explicitly invokes a "review my activity" command
in the AI panel — and even then, only the aggregated stats from
`ProjectContext`, never raw events.

## Performance

- The tracking scanner is rate-limited; full rescans never run inside
  `Update`.
- Stats are computed on `TrackingLoadedMsg` and cached in the
  dashboard model.
- ntcharts canvases are recreated only on resize.

## Tests

Pure stats functions are unit-tested in
`internal/tracker/stats_test.go` with synthetic event slices.

Scanner is integration-tested with temp dirs.

The Pulse view is tested for:
- happy path renders all panels
- empty event log renders without panic
- below-narrow widths collapses panels gracefully
- ntcharts panels are present (assert by rendered border + caption)
