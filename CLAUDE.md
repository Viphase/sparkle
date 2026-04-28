# CLAUDE.md — Sparkle Project Control Map

You are working on **Sparkle**, a local-first Go TUI app for turning rough project sparks into structured, trackable project workspaces.

Keep this file short. Use the linked docs when a task needs detail.

## Product Goal

Sparkle helps a user capture a short idea — a "spark" — and develop it into a structured project with project notes, architecture, target audience, GitHub link, milestones, and automatic work tracking.

Personal-first. Team support may come later, but v1 is for one user managing fewer than 100 projects across multiple workspaces.

## Required Stack

- Go
- Bubble Tea for TUI architecture
- Lip Gloss for styling
- Bubbles for reusable TUI components
- ntcharts or another Bubble Tea-compatible charting library
- Markdown-first local storage
- clean architecture
- tests from the start

## Non-Negotiable Rules

1. Keep domain logic independent from Bubble Tea.
2. Do not block inside Bubble Tea `Update`.
3. Use `tea.Cmd` for file I/O, scanning, and future API calls.
4. Store user data in readable Markdown.
5. Never silently overwrite important user files.
6. Keep the app fast for fewer than 100 projects.
7. Make the TUI polished, keyboard-friendly, and mouse-friendly.
8. Run tests before declaring implementation complete.
9. Use this ascii symbol as a logo: ꕤ

## Primary Resources

Read FIXES.md to know the current issues

Read these files when relevant:

- Product spec: `docs/product-spec.md`
- Architecture: `docs/architecture.md`
- Storage format: `docs/storage-format.md`
- Tracking: `docs/tracking.md`
- AI guide design: `docs/ai-guide.md`
- TUI/UX design: `docs/tui-ux.md`
- Roadmap: `docs/roadmap.md`
- Testing strategy: `docs/testing.md`

## Expected First Implementation Order

1. Create Go module and folder structure.
2. Implement domain models.
3. Implement Markdown storage.
4. Implement app shell and navigation.
5. Implement sparks.
6. Implement project workspace.
7. Implement tracker and charts.
8. Add mock AI provider and AI screen.
9. Write README roadmap.
10. Ensure `go test ./...` passes.

## Definition of Done

- `go test ./...` passes.
- `go run ./cmd/sparkle` launches the TUI.
- User can create/select workspace.
- User can create a spark.
- User can promote a spark into a project.
- User can view/edit project fields.
- Dashboard shows spark/project counts.
- Tracker shows at least one chart.
- Themes can be switched.
- README contains roadmap and architecture overview.
- AI provider interface and mock provider exist.
