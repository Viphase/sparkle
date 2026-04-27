# Sparkle Storage Format

## Storage Philosophy

Sparkle is Markdown-first for content, structured formats for derived data and config.

User-written content (sparks, projects, notes) lives in Markdown so it is:
- readable
- editable outside Sparkle
- version-control friendly
- portable
- recoverable

Derived data (event log, index) lives in JSON/JSONL inside `.sparkle/`. Config lives in TOML.

## Workspace Layout

```txt
workspace-root/
  .sparkle/
    config.toml
    index.json
    events/
      <project_id>.jsonl

  sparks/
    2026-04-27-writing-project-manager.md

  projects/
    sparkle/
      project.md
      notes.md
      tracker.md     // human-readable summary, regenerated from events
```

One Markdown file holds project content (`project.md`); `notes.md` is freeform; `tracker.md` is auto-managed. Sections like architecture, audience, and the GitHub link live as fields or H1 sections inside `project.md` — not as separate files.

## Spark File

```md
---
schema_version: 1
id: spark_20260427_001
title: Writing Project Manager
status: new
tags: [writing, tui, ai]
created_at: 2026-04-27T10:00:00+03:00
updated_at: 2026-04-27T10:00:00+03:00
promoted_project_id:
---

A Go + Bubble Tea project manager that helps turn rough ideas into structured projects.
```

Statuses:
- `new`
- `questioning`
- `promoted`
- `archived`

## Project File

`project.md`:

```md
---
schema_version: 1
id: project_sparkle
title: Sparkle
status: active
github_url:
target_audience:
tags:
created_at:
updated_at:
---

# Description

# Problem

# Target Audience

# Core Features

# Architecture

# Roadmap

# Open Questions
```

Statuses:
- `draft`
- `active`
- `paused`
- `completed`
- `archived`

## Notes File

`notes.md` is freeform Markdown. Frontmatter is optional and limited to `schema_version` and `project_id`.

## Tracker Summary

`tracker.md` is generated from the JSONL event log. It is fully rewritten on each update — never appended to, never user-edited. Treat it as derived output that happens to live next to user files for convenient browsing.

Suggested sections:
- this week's activity
- streak
- recent milestones
- last computed at

## Event Log

`.sparkle/events/<project_id>.jsonl` — one JSON object per line:

```json
{"ts":"2026-04-27T10:15:00+03:00","type":"words_added","value":420,"source":"auto","note":"project.md changed"}
{"ts":"2026-04-27T10:50:00+03:00","type":"session_minutes","value":35,"source":"auto"}
{"ts":"2026-04-27T18:00:00+03:00","type":"milestone_completed","value":1,"source":"manual","note":"MVP architecture drafted"}
```

JSONL chosen because:
- safe to append without re-parsing
- merges cleanly in git
- trivially streamable for stats computation

If lost, the log can be partially rebuilt from file mtimes — recent activity will be approximate, manual events will be gone.

## Index

`.sparkle/index.json` caches:
- workspace id
- spark summaries
- project summaries
- last scanned times
- word counts
- recent activity

The index is derived. If missing, corrupt, or its `schema_version` doesn't match, rebuild it from Markdown.

## Config

`.sparkle/config.toml` holds user preferences (theme, keybinding overrides, AI provider settings, tracker thresholds). TOML, not Markdown — there is no body content, only fields.

## Markdown Write Rules

When saving:
- preserve unknown frontmatter fields
- preserve user-written Markdown body
- write atomically using temp file + rename
- update `updated_at`
- avoid changing formatting unnecessarily
- if on-disk `updated_at` is newer than the loaded copy, surface a conflict instead of overwriting
