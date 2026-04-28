# Sparkle TUI and UX Design

## UX Goal

Sparkle should feel polished, sharp, and fast.

It is not just a CRUD terminal app. UX is a major goal.

## Interaction Model

Keyboard-first with mouse support.

Keyboard:
- `tab` / `shift+tab`: switch major sections
- `j/k` or arrow keys: move
- `enter`: open/select
- `esc`: back/close modal
- `/`: search
- `n`: new spark/project/task
- `e`: edit selected item
- `a`: archive/unarchive selected spark inside Sparks
- `c`: clear active search
- `?`: show extra footer keys
- `?`: help
- `q`: quit

Mouse:
- clickable tabs
- selectable cards/list items
- scroll in long views
- clickable buttons where practical

Keyboard and mouse should dispatch the same domain-level intents.

## Visual Style

Use Lip Gloss centrally.

No hardcoded colors scattered through views.

Use:
- rounded cards
- generous spacing
- pleasant borders
- cool blue/cyan accents
- clear focus states
- readable typography through terminal styling

## Themes

### pastel-light

Light background, pastel accents, not pure grayscale.

### pastel-dark

Dark background, muted blue/cyan accents, not pure grayscale.

### nova

Playful, high-distinction, colorful, still readable.

## Screen Layouts

### Dashboard

Suggested sections:
- top status bar
- stats cards
- active project cards
- recent sparks
- tracking panel (see "Dashboard Tracking" below)
- shortcut footer

Workspace switching lives in Settings, not on the dashboard. v1 assumes one active workspace at a time.

### Sparks Bubble

Should feel like a playful idea board.

Use cards or bubbles:
- title
- short description
- tags
- status
- age/last updated

Actions:
- new
- edit
- archive
- promote
- search by title, description, status, and tags

### Project Workspace

Use two-pane layout when width allows:
- left: project list
- right: selected project summary

Project detail can show tabs:
- overview
- architecture
- audience
- roadmap
- notes
- AI

### Dashboard Tracking

Tracking lives as a panel inside the Dashboard, not a separate top-level screen.

Prioritize charts:
- daily consistency
- weekly activity
- word trend
- streak card
- milestone progress

### AI Screen

Chat-like layout:
- context panel
- messages
- suggested actions
- proposed file changes
- approve/reject controls

## Responsive TUI

Handle:
- narrow terminals
- short terminals
- large terminals
- no mouse support
- no truecolor support

Degrade gracefully.
