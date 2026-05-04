# Sparkle Roadmap

## Status banner

v1 milestones M1–M10 are **shipped but not done**. The owner's verdict
on 2026-04-29 (see [`v2-vision.md`](v2-vision.md) and [`../FIXES.md`](../FIXES.md)):
the implementation is a working prototype with toy-feeling UX, broken
AI onboarding, hand-rolled charts, hard size caps, and a tab-based AI
that nobody uses. The v2 milestones M11–M16 redesign the broken parts.

## v1 milestones (prototype, archived)

| ID  | Area                              | v1 status   |
|-----|-----------------------------------|-------------|
| M1  | Local TUI foundation              | shipped     |
| M2  | Sparks                            | shipped     |
| M3  | Projects                          | partial — long-section editing still shells out to `$EDITOR` |
| M4  | Dashboard tracking + charts       | shipped, charts are unicode hacks |
| M5  | AI-ready architecture             | shipped     |
| M6  | Real Anthropic provider           | shipped, no in-app onboarding |
| M7  | Quiz mode                         | shipped     |
| M8  | Artifact pipeline UI              | shipped     |
| M9  | Skills system                     | shipped, hardcoded constants |
| M10 | AI-aware tracking                 | shipped     |

Everything below "v1 milestones" is the v2 plan.

---

## v2 milestones

### M11 — Real settings & first-run setup

**Why:** C1, C5, L6 in FIXES.md. The current settings are four
unintelligible rows with no API-key field and no test action.

Deliverables:
- First-run setup wizard (separate `tea.Program`) covering: workspace
  path, theme, API key, default skill, first spark.
- Settings modal accessible from anywhere with `,`. Sectioned layout:
  Workspace, Appearance, AI provider, Skills, Tracking, Mouse.
- API-key input with masking + "Test connection" button calling
  `Provider.Ping()`.
- Model picker (Haiku 4.5 / Sonnet 4.6 / Opus 4.7).
- Tracking thresholds editable: words, touch window, session idle,
  streak grace.
- Status bar errors expandable on `?` to show full text.
- Sectioned `config.toml` with v1 flat-form back-compat reader.

Done when:
- A first-run user can paste a key, click Test, and see green ✓.
- Settings shows description text alongside every row.
- `go test ./internal/config/...` and `./internal/tui/modals/settings/...` pass.

### M12 — Responsive layout

**Why:** C2, L8 in FIXES.md. Hard caps make the UI a letterbox on big
terminals.

Deliverables:
- Remove `maxAppWidth`/`maxAppHeight` constants from `internal/tui/root.go`.
- Lower minimum to 50×16 with a graceful "current 38×12, need 50×16"
  message below.
- Implement breakpoint helpers in `internal/tui/components/`:
  `IsNarrow(w int) bool`, `IsMedium(w int) bool`, `IsWide(w int) bool`,
  `IsUltraWide(w int) bool`.
- Logo: cache by width, suppress between widths 50–80, render full
  block ≥ 80.
- Chrome strip and mode bar stretch to terminal width.
- Test the same surface at 50×16, 80×24, 120×36, 220×60 — every
  breakpoint renders without truncation.

Done when:
- A 200×60 terminal shows content edge-to-edge with no whitespace
  letterbox.
- Resizing smoothly reflows panes; no flicker.

### M13 — Pulse rebuild with ntcharts

**Why:** C3 in FIXES.md. "Charts" are unicode bars; user explicitly
asked for "REAL GRAPHS."

Deliverables:
- Add `github.com/NimbleMarkets/ntcharts` to `go.mod`.
- Rename `internal/tui/screens/dashboard/` → `internal/tui/surfaces/pulse/`.
- Five panels per `tracking.md`: Today (4 cards), Words this week
  (BarChart), Activity heatmap (Heatmap, calendar grid), 12-week trend
  (Sparkline), Active projects pipeline & velocity (custom row
  renderer).
- New stats functions: `Last12WeeksWords`, `ProjectVelocity`,
  `PipelineStage`.
- Debounced rescanner (`internal/tracker/watcher.go`) running on
  2-second idle during the session.
- Two new event types: `stage_advanced`, `edit_approved`.

Done when:
- All five panels render with real ntcharts canvases on a wide
  terminal.
- Stats unit tests cover the three new functions.
- The dashboard answers the four Pulse questions on first glance.

### M14 — Workspace surface (spark+project unified, embedded AI)

**Why:** C4, L2, L3 in FIXES.md. The tab-based AI is "stupid chat
nobody uses."

Deliverables:
- New surface `internal/tui/surfaces/workspace/` replacing v1's
  Sparks, Projects, AI tabs.
- Three columns at width ≥ 120: items rail, detail pane, AI panel.
  Two columns 80–119, AI as a `i`-toggled drawer. Single column 50–79,
  detail in main, AI behind a full-width overlay.
- Items rail unifies sparks and promoted projects, glyph-distinguished.
- Inline section editor using `bubbles/textarea` for project.md
  sections — `$EDITOR` shell-out becomes a fallback (still reachable
  via `o`/`O`).
- Markdown body parsed via `goldmark` to a real AST so section
  preview/edit doesn't lose code blocks or list spacing.
- Diff preview component for `<edit>` blocks (markdown-aware).
- Sparks promote inline: cursor stays on the same row, glyph changes,
  AI panel switches into Mentor mode automatically.

Done when:
- Selecting a spark or project on the rail reads the file once, hands
  it to the detail pane, populates the AI panel context.
- A user can promote, answer 3 quizzes, and approve an edit without
  switching tabs (because tabs no longer exist).
- `go test ./internal/tui/surfaces/workspace/...` passes.

### M15 — Filesystem-backed skills, editable prompts, sessions, test connection

**Why:** C6, L4, L5 in FIXES.md. Skills are 5 hardcoded constants;
prompts are inline Go strings; conversations vanish on quit.

Deliverables:
- `internal/storage/markdown/skills.go` — load `.sparkle/skills/*.md`
  into `[]domain.Skill` (struct, not const).
- Seed the 5 v1 skills as Markdown files on first run.
- `internal/storage/markdown/sessions.go` — append per-turn JSONL
  to `.sparkle/sessions/<project_id>.jsonl`; load last N turns on
  project select.
- `.sparkle/prompts/system.md` — read on every `BuildSystemPrompt`
  call; built-in default fallback.
- `Provider.Ping(ctx)` method on both Mock and Anthropic providers.
- Quiz widget gracefully handles 2, 3, 5+ choices.
- Migration: on first v2 launch, seed `skills/` and `prompts/` if
  absent; never overwrite.

Done when:
- A user can author `.sparkle/skills/research-paper.md` and see it in
  the Settings skill picker.
- Editing `.sparkle/prompts/system.md` changes the next AI call's
  system prompt.
- Killing and relaunching Sparkle shows the previous conversation
  history on the project's AI panel.

### M16 — Guided onboarding & contextual prompts

**Why:** C7, L7 in FIXES.md. The app never asks anything; it's a stub
for note-taking.

Deliverables:
- First-run setup wizard (M11) extended with a guided first spark:
  capture title, AI fires its first clarify question immediately,
  user answers, AI signals stage-complete, user advances.
- Workspace empty state ("no sparks yet") shows a clear CTA: "Press n
  to capture your first spark — Sparkle will help shape it from
  there."
- Workspace with stale projects (`DaysSinceActive > 7`) surfaces a
  banner above the AI panel: "It's been 9 days. Resume where we left
  off?" with `r` to send the resume prompt.
- Help overlay (`?`) is **context-aware** — different keys per surface
  per modal stack frame, not a single static screen.
- New keystroke `g` on Workspace: "Get me unstuck" — sends a stock
  prompt to the AI: "I don't know what to do next. What is the
  smallest concrete thing I should do right now?"

Done when:
- A new user launches Sparkle, runs the wizard, captures a spark, and
  the AI immediately asks them a clarifying question — no menus, no
  tab juggling, no "where do I go next" moment.
- An existing user with a 14-day-stale project sees a banner offering
  to resume.
- `?` shows the right keys for the current focus context.

---

## v2 sequencing

The 6 milestones are independent enough to ship in any order, but the
recommended sequence is:

1. **M11 (Settings + setup)** — unblocks user onboarding and AI
   connectivity, the most painful pain point.
2. **M12 (Responsive)** — kills the letterbox, low-risk plumbing.
3. **M13 (Pulse + ntcharts)** — visible payoff, validates the chart
   stack early.
4. **M14 (Workspace + AI panel)** — the biggest UX shift; depends on
   M12 for layout primitives.
5. **M15 (Skills + sessions + prompts)** — depends on M11 for the
   Settings picker.
6. **M16 (Guided)** — final polish; depends on everything else.

## v2 definition of done

See [`v2-vision.md`](v2-vision.md). The cut criteria:

- First run wizards a real user from cold start to "I have a spark
  and an AI question to answer" in under 60 seconds.
- 60×20, 100×30, 180×54 terminals all show coherent layouts.
- Pulse panels are ntcharts canvases.
- Workspace shows rail + detail + AI side-by-side at width ≥ 120.
- Settings has API-key field and a working Test connection button.
- Skills loaded from `.sparkle/skills/*.md`.
- AI conversations persist across launches.
- `go test ./...` passes.

## v3 (tentative, post-v2)

Out of scope for now, listed only to make boundaries clear:

- Multi-workspace switching from inside the app
- AI tool-use / function calling
- Cloud sync (probably never)
- Team workspaces (probably never)
- Plugin system

## Hard exclusions (still and forever in v1)

- Cloud sync, team accounts, real-time multi-user editing
- Plugin marketplace
- Pomodoro timer, kanban board
- AI fine-tuning, training data export
- Database backend
- Mobile / web ports
