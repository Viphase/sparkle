# Sparkle Tracking Design

## Goal

Track user consistency and momentum without requiring constant manual input.

The user should not need to click "save progress" for normal tracking.

## What to Track

Automatic:
- Markdown file modification time
- word count deltas
- daily activity
- session-like activity windows
- project activity streak
- project velocity

Manual or semi-manual:
- milestones
- tasks
- deadlines
- mood
- focus notes

## Event Storage

Events are append-only JSONL at `.sparkle/events/<project_id>.jsonl`. One event per line:

```json
{"ts":"2026-04-27T10:15:00+03:00","type":"words_added","value":420,"source":"auto","note":"project.md changed"}
```

Required fields: `ts`, `type`, `value`, `source`. `note` is optional.

Treat the file as derived data — partially rebuildable from mtimes.

## Event Types

- `words_added`
- `words_removed`
- `session_minutes`
- `file_touched`
- `task_completed`
- `milestone_completed`
- `deadline_added`
- `mood_logged`
- `note_added`

## Meaningful-Event Threshold

Not every save creates an event. To keep charts readable:

- `words_added` / `words_removed`: emit only if absolute delta ≥ 10 words.
- `file_touched`: emit at most once per file per 5-minute window.
- `session_minutes`: emit when an activity window closes (no edits for ≥ 10 minutes).
- Task and milestone events: always emit, regardless of size.

These thresholds live in `config.toml` so a user can tighten or loosen them.

## Automatic Scanning

On startup:
1. load index
2. scan `projects/**` and `sparks/**` mtimes
3. count words only for files whose mtime changed
4. compute deltas
5. append events that pass the meaningful-event threshold
6. update index
7. regenerate `tracker.md` summaries

During app use:
- debounce scans (≥ 2s of idle)
- never scan inside `Update`
- write tracker updates through `tea.Cmd`
- surface errors via the unified `ErrorMsg` envelope

## Charts

Show:
- daily consistency
- weekly activity
- word count trend
- session time
- streak
- milestone completion

Use `ntcharts`. Fall back to text charts only if the chart library is unavailable.

## Stats

Computed in `internal/tracker/stats.go` as pure functions over `[]TrackingEvent`:
- today's words added
- active days this week
- current streak
- longest streak
- average daily words
- weekly project velocity
- last active date

## Privacy

Everything remains local. Tracking data is never sent to AI providers unless the user explicitly asks for AI analysis of stats.
