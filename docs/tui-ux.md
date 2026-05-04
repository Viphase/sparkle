# Sparkle TUI / UX Design — v2

Read [`v2-vision.md`](v2-vision.md) and [`product-spec.md`](product-spec.md)
first. This file specifies the visual language, layout, and
responsiveness contract.

## UX goals

1. **Seamless** — the user reaches every action from the surface they
   are already on. No tab-switching to ask the AI a question.
2. **Polished** — the app feels finished at any width. No truncated
   labels, no letterbox dead zones, no floating boxes on big screens.
3. **Fast** — keystrokes feel immediate; renders are debounced; no
   re-layout flash on resize.
4. **Discoverable** — affordances are visible. Every keystroke that
   does something is shown somewhere on the screen or under `?`.

## Top-level layout

```
┌────────────────────────────────────────────────────────────────────────┐
│ ꕤ  Sparkle    workspace · ~/sparkle              theme · pastel-dark   │  ← chrome (1 row)
│ [ 1 Workspace ]  2 Pulse                                       , • ?   │  ← mode bar (1 row)
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│                          ── view content ──                            │
│                                                                        │
├────────────────────────────────────────────────────────────────────────┤
│ ↑ status bar: latest action / error / hint                             │  ← status (1 row)
└────────────────────────────────────────────────────────────────────────┘
```

The chrome line shows `ꕤ Sparkle`, the active workspace, and the active
theme. The mode bar shows the two surfaces and a context indicator
(`,` for settings, `•` for unsaved state, `?` for help). The status bar
is one row.

There is **no maximum app width or height**. The chrome and mode bar
stretch to the terminal's full width. The view content fills the
remaining height.

## Responsive breakpoints

| Breakpoint | Width        | Layout                                     |
|------------|--------------|--------------------------------------------|
| narrow     | 50–79        | single column; AI panel collapses to drawer |
| medium     | 80–119       | two columns (rail + detail); AI panel below detail or via toggle |
| wide       | 120–179      | three columns (rail + detail + AI panel)   |
| ultrawide  | 180+         | three columns + Pulse mini sidebar         |

Below 50×16 the app shows a single line: "Sparkle needs at least 50×16
columns; current 38×12." Above 50×16 every screen renders.

The breakpoints kick in per-screen via `View(width, height int)`. Do
not implement a global width cap.

## Workspace view

```
┌── 1 Workspace ─────────────────────────────────────────────────────────┐
│ ┌─ items ──────────┐ ┌─ project: Sparkle ─────────────┬─ AI mentor ───┐│
│ │ ✦  Sparkle       │ │ Title    Sparkle              │ stage clarify ││
│ │ ✦  taproot       │ │ Status   active               │ artifacts 4/7 ││
│ │ ◌  ssbot         │ │ Audience solo developers      │               ││
│ │ ★  novel-tracker │ │ GitHub   github.com/v/sparkle │ AI: who is the││
│ │ ─ archived ─     │ │ Tags     go, tui, ai          │ next user…?   ││
│ │ ◌  oldidea       │ │                               │ a) ...        ││
│ │                  │ │ # Description                 │ b) ...        ││
│ │ + n  new spark   │ │ One-paragraph blurb here…     │ c) ...        ││
│ │   /  search      │ │                               │ d) something  ││
│ └──────────────────┘ │ # Architecture                │   else        ││
│                      │ Empty — ask AI to draft       │ ─ input ──    ││
│                      └───────────────────────────────┴───────────────┘│
└────────────────────────────────────────────────────────────────────────┘
```

### Items rail

Width: 24 columns (narrow), 28 (medium), 32 (wide).

Rows:
- Sparks (most recently edited first), grouped by status. ✦ = active
  spark, ◌ = clarifying, ★ = promoted/project, ▣ = archived (greyed).
- Footer always shows `n new`, `/ search`, `1` to focus list.

A "spark" and a "project" appear in the same list. Promotion does not
remove the spark — it changes its glyph from ✦ to ★ and the right
pane swaps to the project surface.

### Detail pane

Width: fills the space between rail and AI panel.

Top: structured fields (title, status, audience, github, tags),
inline-editable. Press `e` to focus a field, `enter` to save, `esc`
to cancel.

Bottom: rendered project.md with section headers as anchors. Press a
section letter (`d` description, `r` roadmap, `a` architecture, `t`
target audience, `o` open questions, `f` features, `p` problem) to
open the inline textarea on that section. Enter twice (or `ctrl+s`) to
save; `esc` to cancel.

The whole detail pane scrolls with `j`/`k`/`pgup`/`pgdn`/`g`/`G`.

### AI mentor panel

Width: 36 columns (wide), 0 (medium — collapses to a toggle), 0
(narrow).

When collapsed (medium / narrow), `i` toggles a full-width AI drawer
that overlays the detail pane.

Top: pipeline indicator. Six stages as `● clarify → ○ structure → ○
challenge → · architect → · expand → · finalize`. Active stage is
filled, visited stages are checkmarks, unvisited are dots.

Below: artifact bar. `artifacts 4/7  ✓ desc  ✓ arch  · roadmap  …`.

Below: messages, scrolled. Each AI turn shows `AI` in accent; user
turns show `You` in primary.

Below: quiz widget when active. Letters `a`–`f` select; `↑↓` move; `enter`
submits.

Below: `<edit>` review overlay when an edit is proposed. Diff preview,
`y` approve, `n` reject, `tab`/`shift+tab` cycles between proposed
edits.

Bottom: input. `enter` sends; `esc` clears; `tab` cycles mode.

## Pulse view

```
┌── 2 Pulse ─────────────────────────────────────────────────────────────┐
│  Today                                                                 │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐                                   │
│  │  840 │ │   3  │ │ 12d  │ │  4/7 │                                   │
│  │words │ │files │ │streak│ │active│                                   │
│  └──────┘ └──────┘ └──────┘ └──────┘                                   │
│                                                                        │
│  ┌─ words this week ────────────────────────────────────────────────┐  │
│  │   ▆ ▇ █ ▅ ▃ ▂ ▁                  ← ntcharts bars                 │  │
│  │   M T W T F S S                                                  │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                        │
│  ┌─ activity · last 30 days ────────────────────────────────────────┐  │
│  │   M T W T F S S                  ← ntcharts heatmap (calendar)  │  │
│  │   · · ■ ■ ■ · ·                                                  │  │
│  │   ■ ■ ■ ■ ■ · ·                                                  │  │
│  │   ■ ■ ■ · ■ · ·                                                  │  │
│  │   ■ ■ ■ ■ ■ ■ ·                                                  │  │
│  │   ■ ■ ■ ■                                                        │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                        │
│  ┌─ active projects · pipeline & velocity ──────────────────────────┐  │
│  │ Sparkle        clarify→structure→challenge→architect→expand→fin │  │
│  │                ●  ●  ●  ○  ·  ·         260 words/wk · streak 12│  │
│  │ taproot        ●  ○  ·  ·  ·  ·         40 words/wk · streak 0  │  │
│  │ novel-tracker  ●  ●  ●  ●  ●  ●         ▔▔▔▔▔▔ ✓ shipped       │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────┘
```

The four hero cards never stack vertically above width 60. Below 60 they
become four short rows.

The bar chart and heatmap use `ntcharts`. The pipeline rows use a
custom row renderer reading `domain.AllModes()`.

## Modals

### Settings modal (`,`)

Reachable from anywhere; opens centered, 80% width × 70% height,
darkens the background.

Sections:
- **Workspace** — root path (read-only, points at "use --workspace flag
  to switch"), config path
- **Appearance** — theme picker (live preview), font hints
- **AI provider** — provider toggle (mock / claude), API-key input
  (masked), model picker (Haiku 4.5 / Sonnet 4.6 / Opus 4.7), default
  skill picker, **Test connection** button
- **Skills** — list of available skills (built-in + on-disk), edit/duplicate
- **Tracking** — words threshold, file-touch window, active-day cutoff
- **Mouse** — toggle

Every row has a label, a value control, and a one-sentence description.

### Help overlay (`?`)

Reachable from anywhere; opens centered. Lists the keyboard shortcuts
for the current surface plus globals. Press any key to dismiss.

### Spark capture (`n`)

Single-input modal centered on the screen. `enter` saves, `esc`
cancels. Saves wherever the user was; does not navigate away.

### Edit review (proposed by AI)

Opens inline in the AI panel (wide) or as a full-width drawer
(medium/narrow). Renders a diff-style preview. `y` approves and writes;
`n` rejects; `tab` cycles between multiple proposals.

## Visual language

### Theme tokens

The theme defines: `background`, `foreground`, `surface`, `subtle`,
`muted`, `primary`, `accent`, `success`, `warning`, `danger`, `info`,
`border`, `borderFocus`, `gradientFrom`, `gradientTo`.

Three themes ship: `pastel-dark` (default), `pastel-light`, `nova`.
Plus user-authored themes loaded from `.sparkle/themes/*.toml`.

### Components (canonical, all in `internal/tui/components/`)

| Component  | Purpose                                              |
|------------|------------------------------------------------------|
| `chrome`   | Top app strip with logo + workspace + theme indicator |
| `modebar`  | Workspace / Pulse switcher + context glyphs          |
| `statusbar`| Bottom row, errors, hints, status                    |
| `rail`     | Left list pane with grouped, scrollable items        |
| `card`     | Bordered hero card with number + label               |
| `chart`    | ntcharts wrappers: `BarChart`, `Heatmap`, `Sparkline`|
| `pipeline` | 6-stage pipeline row renderer                        |
| `input`    | Bordered text input                                  |
| `textarea` | Multi-line markdown editor                           |
| `modal`    | Centered overlay with backdrop                       |
| `diff`     | Markdown-aware diff renderer for edit review         |

No component should hardcode colors; pull from `theme.Theme`.

## Mouse contract

- Mode bar tabs are clickable.
- Items in the rail are clickable.
- Settings modal rows are clickable, including value controls.
- Charts in Pulse are NOT clickable (out of scope).
- Wheel scrolls focused pane.

If `mouse_enabled = false` in config, mouse is disabled; everything
remains keyboard-reachable.

## Performance

- The dashboard logo is rendered once and cached, keyed on width.
- ntcharts canvases are created once per panel resize and reused on
  data updates.
- Tracking rescans run on a 2-second-idle debounce, never inside
  `Update`.
- Status bar messages auto-dismiss after 5 seconds (info) or persist
  until acknowledged (error).

## Accessibility

- All affordances reachable by keyboard.
- Focus state visible (border switches to `borderFocus`).
- Color is never the only signal — every status pill has a glyph
  alongside the color.
- High-contrast mode = the `nova` theme. Truecolor not required;
  themes degrade to 16 colors when the terminal lacks truecolor (use
  `lipgloss` adaptive colors).

## What "polished" means here

A reviewer should be able to open Sparkle on a 220×60 terminal and:
1. See the chrome span the full width.
2. See the Workspace's three columns naturally distribute.
3. Resize the terminal smoothly: panes reflow, no flicker, no
   truncated borders.
4. Press `,` and see a settings modal that reads like a settings
   screen, not a list of cryptic keys.
5. Press `2` and see Pulse with real bar/heatmap charts that actually
   look like charts, not unicode art.
