# Sparkle Product Spec — v2

**Read [`v2-vision.md`](v2-vision.md) before this file.** This spec
operationalises the v2 principles into concrete user flows and
surfaces.

## One-line definition

Sparkle is a local-first Go TUI that walks one person through the
journey from a half-formed spark to a structured, tracked project,
with an opinionated AI mentor that lives next to the work, not in a
separate tab.

## Target user (unchanged from v1)

One person. Solo developer or solo writer. Manages 5 to 100 personal
project ideas across one workspace. Values keyboard speed, terminal
aesthetics, and not having their data in someone else's cloud.

Not in scope: teams, shared workspaces, real-time collaboration,
cloud sync.

## The two surfaces

v2 collapses the v1 five-tab navigation into **two surfaces**:

| Surface       | Purpose                                                         |
|---------------|-----------------------------------------------------------------|
| **Workspace** | Daily driver. List of items on the left, detail + AI on the right. |
| **Pulse**     | The dashboard. Activity, charts, momentum, roadmap progress.   |

Settings is a modal overlay reachable from anywhere with `,` (comma).

(There is no AI tab. There is no Sparks tab. There is no Projects tab.)

## Core concepts

### Spark

A spark is a short, undeveloped idea — title plus optional one-line
description. Designed to be captured in under five seconds.

Lifecycle:
1. **captured** — created, no AI conversation yet
2. **clarifying** — AI is asking questions to sharpen it
3. **promoted** — became a project (the spark is preserved as the
   project's origin record)
4. **archived** — set aside, kept for reference

### Project

A project is a developed workspace promoted from a spark. It owns:
- structured fields (title, status, github, audience, tags)
- a `project.md` with seven canonical sections (description,
  problem, audience, features, architecture, roadmap, open
  questions)
- a `notes.md` for free-form thinking
- a session log of every AI conversation
- a tracking event log

### AI Mentor

The AI is bound to the selected project. It is not a global chat. It:
- Asks the next-most-useful question one at a time.
- Embeds quizzes (multiple choice) when an answer is enumerable.
- Proposes file edits as `<edit>` blocks with diff preview + approval.
- Tracks pipeline stage (clarify → structure → challenge → architect
  → expand → finalize) and signals stage transitions.
- Can be specialised by a **skill** (CLI tool, web API, library, solo
  SaaS, open source, or any user-authored skill on disk).

## Primary user flows

### Flow 1 — First run

1. The user runs `sparkle`.
2. Sparkle detects no `~/sparkle/` workspace and starts the Setup
   wizard.
3. Steps:
   1. *Where should I keep your work?* (default `~/sparkle`)
   2. *Pick a theme.* (preview live)
   3. *Anthropic API key?* (paste, test, or skip — mock provider used
      until set)
   4. *Default skill?* (none / cli-tool / web-api / library / solo-saas
      / open-source)
   5. *Capture your first spark.* (or `skip`)
4. Land in Workspace with the freshly created spark selected.

### Flow 2 — Capture-to-project loop

1. User presses `n` anywhere to capture a spark. A modal opens with
   one input. Enter saves and returns to where they were.
2. Select the spark in Workspace. The right pane shows the spark
   detail and the AI panel below it (or beside, if width allows).
3. User presses `p` (promote). The AI panel switches to Mentor mode
   and immediately fires its first clarifying question.
4. The user answers one question at a time (text or quiz). Each answer
   updates the project artifacts; the artifact bar at the top of the
   panel ticks from 0/7 to 7/7.
5. When the AI signals stage-complete, the panel offers
   "→ next stage" inline; the user accepts or stays.
6. At Finalize, every approved `<edit>` writes to disk atomically.
   The user sees a diff view first; one keystroke approves.
7. Pulse picks up the writes via the tracking scanner and the next
   day's heatmap fills in.

### Flow 3 — Continue a project

1. User runs `sparkle` again later.
2. The Workspace cursor lands on the project they touched most
   recently.
3. The AI panel shows the last few turns of the previous conversation
   (loaded from the session log) and the next-step prompt:
   "Last time we were defining the audience. Want to keep going, or
   pick a different angle?"

### Flow 4 — Daily Pulse

1. User presses `2` to switch to Pulse.
2. Pulse shows:
   - **Today** — words written, files touched, active projects
   - **Streak ribbon** — last 14 days as ntcharts bars
   - **30-day heatmap** — proper calendar grid, not a single row
   - **Per-project velocity** — top 5 projects by recent activity
   - **Pipeline overview** — for each active project, where it is in
     the 6-stage pipeline
3. Numbers and charts coexist: every chart panel shows the absolute
   number alongside the visualization.

## Mode keys (global)

- `1` — Workspace
- `2` — Pulse
- `n` — new spark (modal)
- `,` — Settings (modal)
- `?` — context-aware help overlay
- `q` / `ctrl+c` — quit (yields to inputs)

## What v2 must include

- Workspace view with spark+project unified left rail
- Embedded AI panel bound to selected item
- Pulse dashboard with ntcharts-driven visualizations
- First-run setup wizard
- Settings modal with API-key field, model picker, test action,
  skill picker (file-backed), theme picker
- Custom skills loaded from `.sparkle/skills/*.md`
- Session history persisted to `.sparkle/sessions/<project_id>.jsonl`
- Inline editing of project.md sections via textarea (no editor shell-out)
- Real responsive layout: 60×20 → unbounded
- Live tracking rescans (debounced) during the session

## Out of scope for v2

- Cloud sync, team accounts, shared workspaces
- Real-time multi-user editing
- Plugin marketplace, third-party MCP integrations
- Pomodoro timer, kanban board, calendar
- Tool-use / function-calling on the AI side
- AI fine-tuning or training data export
- Database backend (markdown stays the source of truth)
- Mobile / web ports

## Quality bar

A v2 surface is "done" only when:
- It works at 60×20, 100×30, and 200×60 with no truncation or
  letterboxing.
- All keyboard actions have visible affordances or appear in `?` help.
- All mouse-clickable affordances are also keyboard-accessible.
- All long-running work goes through `tea.Cmd`, never blocks `Update`.
- Errors surface in the status bar with a one-line summary plus a
  `?` to reveal full text.
- It has at least one test that protects the happy path.
