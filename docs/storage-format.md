# Sparkle Storage Format — v2

Markdown-first for content, structured formats for derived data and
config. v1 formats stay; v2 adds three new directories under `.sparkle/`:
`sessions/`, `skills/`, `themes/`, `prompts/`.

## Workspace layout

```txt
workspace-root/
  .sparkle/
    config.toml             // user preferences
    index.json              // derived cache
    events/                 // tracking event log per project
      <project_id>.jsonl
    sessions/               // NEW: AI conversation log per project
      <project_id>.jsonl
    skills/                 // NEW: skill prompt fragments
      cli-tool.md
      web-api.md
      library.md
      solo-saas.md
      open-source.md
      <user-skill>.md
    prompts/                // NEW: editable system prompts
      system.md
    themes/                 // NEW: user-authored themes
      <name>.toml

  sparks/
    2026-04-29-writing-project-manager.md

  projects/
    sparkle/
      project.md
      notes.md
      tracker.md            // optional, derived
```

`tracker.md` is regenerated on each tracking update for users who want
to browse summaries outside the TUI.

## Spark file (unchanged from v1)

```md
---
schema_version: 1
id: spark_20260429_001
title: Writing Project Manager
status: new
tags: [writing, tui, ai]
created_at: 2026-04-29T10:00:00+03:00
updated_at: 2026-04-29T10:00:00+03:00
promoted_project_id:
---

A Go + Bubble Tea project manager…
```

Statuses: `new`, `clarifying`, `promoted`, `archived`.

(v2 renames v1's `questioning` → `clarifying` to align with pipeline
stage naming.)

## Project file (unchanged)

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

Statuses: `draft`, `active`, `paused`, `completed`, `archived`.

## Notes file (unchanged)

`notes.md` is freeform Markdown. Frontmatter optional, limited to
`schema_version` and `project_id`.

## Event log (unchanged shape, two new types)

`.sparkle/events/<project_id>.jsonl`:

```json
{"ts":"2026-04-29T10:15:00+03:00","type":"words_added","value":420,"source":"auto","note":"project.md"}
{"ts":"2026-04-29T10:30:00+03:00","type":"stage_advanced","value":1,"source":"manual","note":"clarify→structure"}
{"ts":"2026-04-29T10:45:00+03:00","type":"edit_approved","value":1,"source":"manual","note":"project.md description"}
```

Two new types: `stage_advanced`, `edit_approved`.

## AI session log (NEW)

`.sparkle/sessions/<project_id>.jsonl` — append-only:

```json
{"ts":"2026-04-29T10:15:00+03:00","role":"assistant","content":"Who is the primary user?","mode":"clarify","quizzes":1}
{"ts":"2026-04-29T10:15:08+03:00","role":"user","content":"a) A solo developer","kind":"quiz_answer"}
{"ts":"2026-04-29T10:15:15+03:00","role":"assistant","content":"…","mode":"clarify","stage_complete":false}
```

Required: `ts`, `role`, `content`. Optional: `mode`, `kind`,
`stage_complete`, `quizzes` (count of structured quizzes in the message).

Roles: `user`, `assistant`. (System messages are not logged — the
system prompt is reconstructed from skill + mode + context at request
time.)

Used by:
- AI panel restoration on project select (loads last 20 turns).
- Pulse "AI activity" stat (count of assistant turns this week).

If lost, the AI starts cold; the project is unaffected.

## Skill files (NEW)

`.sparkle/skills/<key>.md`:

```md
---
schema_version: 1
key: web-api
label: Web API
description: REST/GraphQL shape, auth, rate limiting, error contracts
---

Project type: WEB API.
Additional focus:
- REST resource naming (plural nouns, no verbs) or GraphQL schema design.
- Auth strategy: JWT, API keys, OAuth2 — pick one early and defend it.
- …
```

Required frontmatter: `schema_version`, `key`, `label`. Optional:
`description`. Body is the prompt fragment injected into the system
prompt when this skill is active.

`key` rules:
- kebab-case, lowercase ASCII letters, digits, hyphens
- unique within `skills/`
- two reserved keys: `none` (= no skill) and `default`

The five v1 skill names (`cli-tool`, `web-api`, `library`,
`solo-saas`, `open-source`) ship as seed files on first launch and
can be edited freely.

## System prompt file (NEW)

`.sparkle/prompts/system.md` — plain Markdown, no frontmatter:

```md
You are Sparkle's local project mentor.
Your job is to help one person turn a rough idea into a structured,
tracked project.

You MUST:
- Ask one question at a time.
…
```

If the file is missing, Sparkle uses the built-in default. Edits take
effect on the next request (no reload required — read on each
`BuildSystemPrompt` call).

## Theme files (NEW)

`.sparkle/themes/<name>.toml`:

```toml
schema_version = 1
name = "midnight"
display_name = "Midnight"

[colors]
background    = "#0e0e1a"
foreground    = "#e8e8f0"
surface       = "#181828"
subtle        = "#3a3a4a"
muted         = "#6a6a7a"
primary       = "#7aa2f7"
accent        = "#bb9af7"
success       = "#9ece6a"
warning       = "#e0af68"
danger        = "#f7768e"
info          = "#7dcfff"
border        = "#3a3a4a"
border_focus  = "#bb9af7"
gradient_from = "#7aa2f7"
gradient_to   = "#bb9af7"
```

Validation: every key must be a valid hex color or named ANSI color
(`#rrggbb`, `#rgb`, or `red`/`green`/…). Invalid themes log to status
bar and are skipped.

Built-in themes (`pastel-dark`, `pastel-light`, `nova`) ship as Go
literals seeded on first run; users can copy them via the Settings
modal's "Duplicate theme" button.

## Index (unchanged)

`.sparkle/index.json` caches workspace ID, spark/project summaries,
last scanned times, word counts, recent activity.

Derived. If missing, corrupt, or schema-mismatched, rebuild from
Markdown.

## Config (extended)

`.sparkle/config.toml`:

```toml
# Sparkle workspace preferences
schema_version = 1

theme = "pastel-dark"
mouse_enabled = true

[ai]
provider = "claude"               # "mock" or "claude"
model = "claude-haiku-4-5"        # claude-haiku-4-5 | claude-sonnet-4-6 | claude-opus-4-7
api_key = ""                      # blank → mock provider; ANTHROPIC_API_KEY env wins
active_skill = "web-api"          # key of skill in .sparkle/skills/

[tracking]
words_threshold      = 10
touch_window_secs    = 300
session_idle_minutes = 10
streak_grace_hours   = 4
```

Sectioned TOML in v2 (vs. flat v1). The loader supports both for
backward compatibility but writes the sectioned form.

`api_key` is intentionally blank in default writes. The Settings modal
accepts and stores it; `ANTHROPIC_API_KEY` env always wins.

## Markdown write rules (unchanged)

When saving:
- preserve unknown frontmatter fields
- preserve user-written Markdown body
- write atomically using temp file + rename
- update `updated_at`
- avoid changing formatting unnecessarily
- if on-disk `updated_at` is newer than the loaded copy, surface a
  conflict instead of overwriting

## Migration from v1

On first v2 launch:
1. If `.sparkle/config.toml` is the flat v1 form, parse it, write back
   the sectioned v2 form.
2. If `.sparkle/skills/` does not exist, seed it with the 5 v1 skills.
3. If `.sparkle/prompts/` does not exist, seed `system.md` with the v1
   built-in.
4. If `.sparkle/sessions/` does not exist, create it empty.
5. If `.sparkle/themes/` does not exist, create it empty (built-ins
   stay in Go for now).

Never delete or rename existing user files during migration.
