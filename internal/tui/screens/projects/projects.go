package projects

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/theme"
)

// Loader is the storage interface the projects screen needs.
type Loader interface {
	ListProjects() ([]domain.Project, error)
	SaveProject(domain.Project) error
	ProjectPath(id string) string  // path to project.md, used by open-in-editor
	NotesPath(id string) string    // path to notes.md, used by open-in-editor
	DeleteProject(id string) error // permanently removes a project directory
}

// pane tracks which side has keyboard focus.
type pane int

const (
	paneList pane = iota
	paneDetail
)

// Detail field indices.
const (
	fldTitle    = 0
	fldStatus   = 1
	fldGitHub   = 2
	fldAudience = 3
	fldTags     = 4
	numFields   = 5
)

const listPaneW = 30 // fixed width of the left list pane

type fieldMeta struct {
	label string
	value string
}

// Model implements the Projects screen (M3 two-pane layout).
type Model struct {
	theme  theme.Theme
	loader Loader

	items  []domain.Project
	cursor int
	offset int
	listH  int
	loaded bool

	activePane    pane
	detailField   int
	inputActive   bool
	input         textinput.Model
	deleteConfirm bool

	now func() time.Time
}

func New(t theme.Theme, loader Loader) screens.Screen {
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Prompt = ""
	ti.PromptStyle = theme.Fg(t, t.Primary)
	ti.TextStyle = theme.Fg(t, t.Foreground)
	ti.Cursor.Style = theme.Fg(t, t.Accent)
	ti.PlaceholderStyle = theme.Fg(t, t.Subtle)

	return &Model{
		theme:  t,
		loader: loader,
		input:  ti,
		now:    time.Now,
	}
}

func (m *Model) Init() tea.Cmd { return nil }
func (m *Model) Title() string { return "Projects" }

// InForm reports whether a text input is active (used by root to pass keys through).
func (m *Model) InForm() bool { return m.inputActive }

func (m *Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.ThemeChangedMsg:
		m.theme = theme.ByName(msg.ThemeName)
		return m, nil
	case msgs.ProjectsLoadedMsg:
		m.items = msg.Items
		m.loaded = true
		m.clampCursor()
		return m, nil
	case tea.MouseMsg:
		return m.updateMouse(msg)
	case tea.KeyMsg:
		if m.inputActive {
			return m.updateInput(msg)
		}
		if m.activePane == paneDetail {
			return m.updateDetail(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

// updateMouse handles wheel scrolling and click-to-select in the list pane.
// Y is content-relative (root already subtracted the tab-bar height).
func (m *Model) updateMouse(msg tea.MouseMsg) (screens.Screen, tea.Cmd) {
	if m.inputActive {
		return m, nil // text input active — ignore mouse
	}
	// Compute pane widths exactly as View() does.
	lw := listPaneW
	// We don't have access to width here, but lw=30 is the default.
	// Clicks in X < lw are in the list pane; X > lw are in the detail pane.
	inListPane := msg.X < lw

	switch msg.Type {
	case tea.MouseWheelUp:
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
	case tea.MouseWheelDown:
		if m.cursor+1 < len(m.items) {
			m.cursor++
			m.ensureCursorVisible()
		}
	case tea.MouseLeft:
		if inListPane {
			// List pane layout (no box padding):
			//   row 0 → header "Projects"
			//   row 1 → blank
			//   row 2 → first item
			itemRow := msg.Y - 2
			if itemRow >= 0 && m.offset+itemRow < len(m.items) {
				m.cursor = m.offset + itemRow
				m.activePane = paneList
				m.ensureCursorVisible()
			}
		} else {
			// Clicking in the detail pane switches focus to it.
			if len(m.items) > 0 {
				m.activePane = paneDetail
			}
		}
	}
	return m, nil
}

func (m *Model) updateList(key tea.KeyMsg) (screens.Screen, tea.Cmd) {
	if key.String() != "D" {
		m.deleteConfirm = false
	}
	switch key.String() {
	case "j", "down":
		if m.cursor+1 < len(m.items) {
			m.cursor++
			m.ensureCursorVisible()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
	case "g", "home":
		m.cursor = 0
		m.ensureCursorVisible()
	case "G", "end":
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
			m.ensureCursorVisible()
		}
	case "enter", "l":
		if len(m.items) > 0 {
			m.activePane = paneDetail
			m.detailField = 0
		}
	case "o":
		p, ok := m.selectedProject()
		if !ok {
			return m, nil
		}
		return m, m.openProjectInEditorCmd(p)
	case "O":
		p, ok := m.selectedProject()
		if !ok {
			return m, nil
		}
		return m, m.openNotesInEditorCmd(p)
	case "D":
		p, ok := m.selectedProject()
		if !ok {
			return m, nil
		}
		if m.deleteConfirm {
			m.deleteConfirm = false
			return m, m.deleteProjectCmd(p)
		}
		m.deleteConfirm = true
		return m, nil
	}
	return m, nil
}

func (m *Model) updateDetail(key tea.KeyMsg) (screens.Screen, tea.Cmd) {
	if key.String() != "D" {
		m.deleteConfirm = false
	}
	switch key.String() {
	case "esc", "h":
		m.activePane = paneList
	case "j", "down":
		if m.detailField < numFields-1 {
			m.detailField++
		}
	case "k", "up":
		if m.detailField > 0 {
			m.detailField--
		}
	case "e", "enter":
		if m.detailField == fldStatus {
			return m, nil // status uses ←→
		}
		p, ok := m.selectedProject()
		if !ok {
			return m, nil
		}
		return m.startEdit(p)
	case "left":
		if m.detailField == fldStatus {
			return m.cycleStatus(-1)
		}
	case "right":
		if m.detailField == fldStatus {
			return m.cycleStatus(+1)
		}
	case "o":
		p, ok := m.selectedProject()
		if !ok {
			return m, nil
		}
		return m, m.openProjectInEditorCmd(p)
	case "O":
		p, ok := m.selectedProject()
		if !ok {
			return m, nil
		}
		return m, m.openNotesInEditorCmd(p)
	case "D":
		p, ok := m.selectedProject()
		if !ok {
			return m, nil
		}
		if m.deleteConfirm {
			m.deleteConfirm = false
			return m, m.deleteProjectCmd(p)
		}
		m.deleteConfirm = true
		return m, nil
	}
	return m, nil
}

func (m *Model) cycleStatus(dir int) (screens.Screen, tea.Cmd) {
	p, ok := m.selectedProject()
	if !ok {
		return m, nil
	}
	statuses := domain.AllProjectStatuses()
	idx := 0
	for i, s := range statuses {
		if s == p.Status {
			idx = i
			break
		}
	}
	idx = (idx + dir + len(statuses)) % len(statuses)
	p.Status = statuses[idx]
	p.UpdatedAt = m.now().UTC()
	m.items[m.cursor] = p // optimistic update
	return m, m.saveProjectCmd(p)
}

func (m *Model) startEdit(p domain.Project) (screens.Screen, tea.Cmd) {
	var val, placeholder string
	switch m.detailField {
	case fldTitle:
		val = p.Title
		placeholder = "Project title"
	case fldGitHub:
		val = p.GitHubURL
		placeholder = "https://github.com/..."
	case fldAudience:
		val = p.TargetAudience
		placeholder = "Who is this for?"
	case fldTags:
		val = strings.Join(p.Tags, ", ")
		placeholder = "tag1, tag2, ..."
	}
	m.input.Placeholder = placeholder
	m.input.SetValue(val)
	m.input.CursorEnd()
	m.input.Focus()
	m.inputActive = true
	return m, textinput.Blink
}

func (m *Model) updateInput(key tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.inputActive = false
		m.input.Blur()
		return m, nil
	case "enter":
		p, ok := m.selectedProject()
		if !ok {
			m.inputActive = false
			m.input.Blur()
			return m, nil
		}
		val := strings.TrimSpace(m.input.Value())
		switch m.detailField {
		case fldTitle:
			p.Title = val
		case fldGitHub:
			p.GitHubURL = val
		case fldAudience:
			p.TargetAudience = val
		case fldTags:
			p.Tags = parseTags(val)
		}
		p.UpdatedAt = m.now().UTC()
		m.items[m.cursor] = p // optimistic
		m.inputActive = false
		m.input.Blur()
		return m, m.saveProjectCmd(p)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)
	return m, cmd
}

// openProjectInEditorCmd suspends the TUI, opens project.md in $EDITOR, then
// reloads the project list so any edits are reflected immediately.
func (m *Model) openProjectInEditorCmd(p domain.Project) tea.Cmd {
	if m.loader == nil {
		return func() tea.Msg {
			return msgs.ErrorMsg{Source: "open-project", Err: fmt.Errorf("no storage configured")}
		}
	}
	return m.openPathInEditorCmd("project", m.loader.ProjectPath(p.ID))
}

// openNotesInEditorCmd opens notes.md for the selected project. Notes are
// regular Markdown on disk and are never overwritten by SaveProject.
func (m *Model) openNotesInEditorCmd(p domain.Project) tea.Cmd {
	if m.loader == nil {
		return func() tea.Msg {
			return msgs.ErrorMsg{Source: "open-notes", Err: fmt.Errorf("no storage configured")}
		}
	}
	return m.openPathInEditorCmd("notes", m.loader.NotesPath(p.ID))
}

func (m *Model) openPathInEditorCmd(source, path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	loader := m.loader
	c := exec.Command(editor, path) //nolint:gosec
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return msgs.ErrorMsg{Source: "editor-" + source, Err: err}
		}
		items, listErr := loader.ListProjects()
		if listErr != nil {
			return msgs.ErrorMsg{Source: "list-projects", Err: listErr}
		}
		return msgs.ProjectsLoadedMsg{Items: items}
	})
}

func (m *Model) saveProjectCmd(p domain.Project) tea.Cmd {
	loader := m.loader
	return func() tea.Msg {
		if loader == nil {
			return msgs.ErrorMsg{Source: "save-project", Err: fmt.Errorf("no storage configured")}
		}
		if err := loader.SaveProject(p); err != nil {
			return msgs.ErrorMsg{Source: "save-project", Err: err}
		}
		items, err := loader.ListProjects()
		if err != nil {
			return msgs.ErrorMsg{Source: "list-projects", Err: err}
		}
		return msgs.ProjectsLoadedMsg{Items: items}
	}
}

func (m *Model) deleteProjectCmd(p domain.Project) tea.Cmd {
	loader := m.loader
	cursor := m.cursor
	return func() tea.Msg {
		if loader == nil {
			return msgs.ErrorMsg{Source: "delete-project", Err: fmt.Errorf("no storage configured")}
		}
		if err := loader.DeleteProject(p.ID); err != nil {
			return msgs.ErrorMsg{Source: "delete-project", Err: err}
		}
		items, err := loader.ListProjects()
		if err != nil {
			return msgs.ErrorMsg{Source: "list-projects", Err: err}
		}
		_ = cursor // cursor will be clamped on ProjectsLoadedMsg
		return msgs.ProjectsLoadedMsg{Items: items}
	}
}

func (m *Model) selectedProject() (domain.Project, bool) {
	if len(m.items) == 0 || m.cursor < 0 || m.cursor >= len(m.items) {
		return domain.Project{}, false
	}
	return m.items[m.cursor], true
}

func (m *Model) clampCursor() {
	if len(m.items) == 0 {
		m.cursor = 0
		m.offset = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
}

func (m *Model) ensureCursorVisible() {
	m.clampCursor()
	if m.listH <= 0 {
		m.listH = 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.listH {
		m.offset = m.cursor - m.listH + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
	if maxOff := max(0, len(m.items)-m.listH); m.offset > maxOff {
		m.offset = maxOff
	}
}

// ─────────────────────────── View ───────────────────────────

func (m *Model) View(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	if !m.loaded || len(m.items) == 0 {
		return m.emptyView(width, height)
	}

	lw := listPaneW // 30 by default
	if width < 80 {
		lw = 20
	}
	if width < 60 {
		lw = width / 3
	}
	rw := width - lw - 1
	if rw < 10 {
		rw = 10
	}

	listPane := m.renderListPane(lw, height)
	divider := m.renderDivider(height)
	detailPane := m.renderDetailPane(rw, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, divider, detailPane)
}

func (m *Model) emptyView(width, height int) string {
	if !m.loaded {
		body := lipgloss.JoinVertical(lipgloss.Center,
			theme.ApplyGradOn("Projects", m.theme.GradientFrom, m.theme.GradientTo, m.theme.Background, true),
			"",
			theme.Fg(m.theme, m.theme.Muted).Render("Loading projects…"),
		)
		return theme.Place(m.theme, width, height, lipgloss.Center, lipgloss.Center, body)
	}
	header := theme.ApplyGradOn("Projects", m.theme.GradientFrom, m.theme.GradientTo, m.theme.Background, true)
	body := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		theme.Fg(m.theme, m.theme.Foreground).Render("No projects yet."),
		"",
		theme.Fg(m.theme, m.theme.Subtle).Italic(true).Render("promote a spark with p to create your first project"),
	)
	return theme.Place(m.theme, width, height, lipgloss.Center, lipgloss.Center, body)
}

func (m *Model) renderListPane(width, height int) string {
	m.listH = max(1, height-3)
	m.ensureCursorVisible()

	header := theme.ApplyGradOn("Projects", m.theme.GradientFrom, m.theme.GradientTo, m.theme.Background, true)
	rows := []string{header, ""}

	start := m.offset
	end := min(len(m.items), start+m.listH)
	for i, p := range m.items[start:end] {
		rows = append(rows, m.renderListRow(i+start, p, width))
	}
	if start > 0 || end < len(m.items) {
		rows = append(rows, theme.Fg(m.theme, m.theme.Subtle).
			Render(fmt.Sprintf("%d-%d of %d", start+1, end, len(m.items))))
	}

	footerColor := m.theme.Subtle
	if m.activePane == paneList {
		footerColor = m.theme.Muted
	}
	footer := "j/k  enter open  D delete"
	if m.deleteConfirm {
		footer = theme.Fg(m.theme, m.theme.Danger).Render("D again to confirm delete — any other key cancels")
	} else {
		footer = theme.Fg(m.theme, footerColor).Render(footer)
	}
	rows = append(rows, "", footer)

	content := strings.Join(rows, "\n")
	return theme.Base(m.theme).Width(width).Height(height).Render(content)
}

func (m *Model) renderListRow(i int, p domain.Project, width int) string {
	name := p.Title
	if name == "" {
		name = p.ID
	}
	maxW := width - 5
	if maxW < 1 {
		maxW = 1
	}
	runes := []rune(name)
	if len(runes) > maxW {
		name = string(runes[:maxW-1]) + "…"
	}

	titleStyle := theme.Fg(m.theme, m.theme.Foreground)
	cur := "  "

	if i == m.cursor {
		cur = theme.Fg(m.theme, m.theme.Accent).Bold(true).Render("▌ ")
		if m.activePane == paneList {
			titleStyle = titleStyle.Foreground(m.theme.Primary).Bold(true)
		}
	}

	dot := theme.Fg(m.theme, m.statusColor(p.Status)).Render("●")
	return cur + dot + " " + titleStyle.Render(name)
}

func (m *Model) renderDivider(height int) string {
	lines := make([]string, height)
	for i := range lines {
		lines[i] = "│"
	}
	content := strings.Join(lines, "\n")
	return theme.Fg(m.theme, m.theme.Border).Width(1).Height(height).Render(content)
}

func (m *Model) renderDetailPane(width, height int) string {
	p, ok := m.selectedProject()
	if !ok {
		return theme.Base(m.theme).Width(width).Height(height).Padding(0, 1).Render(
			theme.Fg(m.theme, m.theme.Subtle).Italic(true).Render("select a project to view details"),
		)
	}

	displayTitle := p.Title
	if displayTitle == "" {
		displayTitle = p.ID
	}
	titleRunes := []rune(displayTitle)
	if len(titleRunes) > width-4 {
		displayTitle = string(titleRunes[:width-5]) + "…"
	}
	titleLine := theme.ApplyGradOn(displayTitle, m.theme.GradientFrom, m.theme.GradientTo, m.theme.Background, true)

	rows := []string{titleLine, ""}

	fields := buildFields(p)
	const labelW = 10
	for i, f := range fields {
		selected := m.activePane == paneDetail && i == m.detailField
		cur := "  "
		labelStyle := theme.Fg(m.theme, m.theme.Muted).Width(labelW)
		valStyle := theme.Fg(m.theme, m.theme.Foreground)

		if selected {
			cur = theme.Fg(m.theme, m.theme.Accent).Bold(true).Render("▌ ")
			labelStyle = labelStyle.Foreground(m.theme.Primary)
			valStyle = valStyle.Foreground(m.theme.Primary).Bold(true)
		}

		var valStr string
		switch {
		case m.inputActive && selected:
			valStr = m.input.View()
		case i == fldStatus:
			valStr = m.renderStatusField(p.Status, selected)
		default:
			valStr = valStyle.Render(f.value)
		}

		rows = append(rows, cur+labelStyle.Render(f.label)+valStr)
	}

	if !p.CreatedAt.IsZero() {
		rows = append(rows, "")
		rows = append(rows, "  "+theme.Fg(m.theme, m.theme.Subtle).Render(
			"created "+p.CreatedAt.Local().Format("2006-01-02"),
		))
	}

	rows = append(rows, "")
	rows = append(rows, m.renderSectionPreview(p, width)...)

	rows = append(rows, "")
	rows = append(rows, theme.Fg(m.theme, m.theme.Subtle).Render(m.detailHint()))

	content := strings.Join(rows, "\n")
	return theme.Base(m.theme).Width(width).Height(height).Padding(0, 1).Render(content)
}

func (m *Model) renderStatusField(s domain.ProjectStatus, selected bool) string {
	valStyle := theme.Fg(m.theme, m.theme.Foreground)
	arrowStyle := theme.Fg(m.theme, m.theme.Subtle)
	if selected {
		valStyle = valStyle.Foreground(m.theme.Primary).Bold(true)
		arrowStyle = arrowStyle.Foreground(m.theme.Primary)
	}
	return arrowStyle.Render("← ") + valStyle.Render(string(s)) + arrowStyle.Render(" →")
}

func (m *Model) detailHint() string {
	if m.inputActive {
		return "enter save  esc cancel"
	}
	if m.deleteConfirm {
		return theme.Fg(m.theme, m.theme.Danger).Render("D again to confirm delete — any other key cancels")
	}
	if m.activePane == paneDetail {
		if m.detailField == fldStatus {
			return "← → change status  j/k select  o project.md  O notes.md  D delete  esc back"
		}
		return "e edit  j/k select  ← → status  o project.md  O notes.md  D delete  esc back"
	}
	return "enter/l open detail  o project.md  O notes.md  D delete"
}

func (m *Model) statusColor(s domain.ProjectStatus) lipgloss.Color {
	switch s {
	case domain.ProjectStatusActive:
		return m.theme.Success
	case domain.ProjectStatusDraft:
		return m.theme.Info
	case domain.ProjectStatusPaused:
		return m.theme.Warning
	case domain.ProjectStatusCompleted:
		return m.theme.Primary
	case domain.ProjectStatusArchived:
		return m.theme.Subtle
	}
	return m.theme.Muted
}

// ─────────────────────────── helpers ───────────────────────────

func buildFields(p domain.Project) []fieldMeta {
	return []fieldMeta{
		{label: "Title", value: nonEmpty(p.Title, p.ID)},
		{label: "Status", value: string(p.Status)},
		{label: "GitHub", value: nonEmpty(p.GitHubURL, "(not set)")},
		{label: "Audience", value: nonEmpty(p.TargetAudience, "(not set)")},
		{label: "Tags", value: nonEmpty(strings.Join(p.Tags, ", "), "(none)")},
	}
}

func (m *Model) renderSectionPreview(p domain.Project, width int) []string {
	body := strings.TrimSpace(p.Body)
	if body == "" {
		return []string{
			theme.Fg(m.theme, m.theme.Subtle).Render("Sections"),
			"  " + theme.Fg(m.theme, m.theme.Muted).Render("open project.md to fill description, architecture, and roadmap"),
		}
	}

	sectionNames := []string{"Description", "Architecture", "Roadmap"}
	rows := []string{theme.Fg(m.theme, m.theme.Subtle).Render("Sections")}
	valueWidth := width - 18
	if valueWidth < 16 {
		valueWidth = 16
	}
	for _, name := range sectionNames {
		preview := sectionPreview(body, name)
		if preview == "" {
			preview = "(empty)"
		}
		preview = truncateRunes(preview, valueWidth)
		label := theme.Fg(m.theme, m.theme.Muted).Width(14).Render(name)
		val := theme.Fg(m.theme, m.theme.Foreground).Render(preview)
		rows = append(rows, "  "+label+val)
	}
	rows = append(rows, "  "+theme.Fg(m.theme, m.theme.Muted).Render("Notes")+"        "+
		theme.Fg(m.theme, m.theme.Foreground).Render("O opens notes.md"))
	return rows
}

func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func parseTags(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func sectionPreview(body, heading string) string {
	lines := strings.Split(body, "\n")
	var parts []string
	inSection := false
	want := "# " + strings.ToLower(strings.TrimSpace(heading))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(trimmed, "# ") {
			if inSection {
				break
			}
			if lower == want {
				inSection = true
			}
			continue
		}
		if inSection && trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, " ")
}

func truncateRunes(s string, maxLen int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= maxLen {
		return string(runes)
	}
	if maxLen <= 1 {
		return string(runes[:max(0, maxLen)])
	}
	return string(runes[:maxLen-1]) + "…"
}
