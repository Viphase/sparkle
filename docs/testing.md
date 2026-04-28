# Sparkle Testing Strategy

## Requirement

Build with tests from the start.

`go test ./...` must pass before work is considered complete.

## Unit Tests

Add tests for:
- Markdown frontmatter parsing
- Spark creation
- Spark editing
- Spark archiving
- Spark promotion to project
- Project field updates
- Workspace indexing
- Dashboard tracking stat calculations
- Theme loading
- Route/navigation state transitions where practical

## Domain Tests

Domain tests should not import:
- Bubble Tea
- Lip Gloss
- filesystem implementation packages

## Storage Tests

Use temporary directories.

Test:
- write spark
- read spark
- preserve Markdown body
- update frontmatter
- preserve unknown frontmatter fields across a write
- atomic write (temp + rename) survives a simulated mid-write crash
- rebuild index from raw Markdown when `.sparkle/index.json` is missing, corrupt, or has a mismatched `schema_version` (integration test)
- handle invalid Markdown gracefully
- conflict detection when on-disk `updated_at` is newer than the loaded copy

## Dashboard Tracking Tests

Pure calculations supporting the Dashboard tracking panel (`internal/tracker/stats.go`):
- word counts
- daily totals
- weekly totals
- streaks
- velocity

Scanner (`internal/tracker/scanner.go`, integration via temp dirs):
- unchanged file detection (skip word-count work when mtime unchanged)
- meaningful-event threshold suppresses small deltas
- JSONL append survives interleaved scans
- partial event-log rebuild from mtimes when the JSONL is deleted

Dashboard screen:
- consumes tracking-loaded messages
- renders tracking stats/charts without relying on a separate Tracker route

## TUI Tests

Keep TUI tests practical.

Test:
- route changes
- message handling
- basic model state transitions
- no panics on small terminal sizes

Do not over-test visual rendering unless specific regressions appear.
