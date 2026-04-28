package ai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	guide "github.com/viphase/sparkle/internal/ai"
	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/theme"
)

type completionMsg struct {
	text  string
	edits []domain.ProposedEdit
	err   error
}

type editApprovalMsg struct {
	edit    domain.ProposedEdit
	workDir string
	err     error
}

// editMode tracks whether the screen is in the approve/reject flow.
type editMode int

const (
	editModeNone     editMode = iota
	editModeReviewing // showing a pending edit for review
)

type Model struct {
	theme         theme.Theme
	provider      guide.Provider
	providerName  string // "mock" or "claude"
	input         textinput.Model
	messages      []domain.Message
	context       domain.ProjectContext
	workDir       string // workspace root for writing approved edits
	scroll        int
	waiting       bool
	errText       string
	mode          domain.Mode
	pendingEdits  []domain.ProposedEdit
	editIdx       int
	editViewMode  editMode
}

// New creates the AI screen. Pass an optional guide.Provider to override the
// default mock.
func New(t theme.Theme, providers ...guide.Provider) screens.Screen {
	return newModel(t, "", providers...)
}

// NewWithWorkDir creates the AI screen and sets the workspace root so approved
// edits can be written to disk.
func NewWithWorkDir(t theme.Theme, workDir string, providers ...guide.Provider) screens.Screen {
	return newModel(t, workDir, providers...)
}

func newModel(t theme.Theme, workDir string, providers ...guide.Provider) *Model {
	var provider guide.Provider = guide.NewMockProvider()
	providerName := "mock"
	if len(providers) > 0 && providers[0] != nil {
		provider = providers[0]
		providerName = "claude"
	}

	ti := textinput.New()
	ti.CharLimit = 500
	ti.Prompt = "› "
	ti.PromptStyle = theme.Fg(t, t.Primary)
	ti.TextStyle = theme.Fg(t, t.Foreground)
	ti.Cursor.Style = theme.Fg(t, t.Accent)
	ti.Placeholder = "Ask about audience, architecture, roadmap, or risk"
	ti.PlaceholderStyle = theme.Fg(t, t.Subtle)
	ti.Focus()

	return &Model{
		theme:        t,
		provider:     provider,
		providerName: providerName,
		input:        ti,
		workDir:      workDir,
		mode:         domain.ModeClarify,
		messages: []domain.Message{
			{
				Role:    domain.MessageRoleAssistant,
				Content: "I am Sparkle's AI guide. Ask a project question and I will help shape it into concrete work.",
			},
		},
	}
}

func (m *Model) Init() tea.Cmd  { return textinput.Blink }
func (m *Model) Title() string  { return "AI" }
func (m *Model) InForm() bool   { return m.input.Focused() && m.editViewMode == editModeNone }

func (m *Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.ThemeChangedMsg:
		m.theme = theme.ByName(msg.ThemeName)
		m.input.PromptStyle = theme.Fg(m.theme, m.theme.Primary)
		m.input.TextStyle = theme.Fg(m.theme, m.theme.Foreground)
		m.input.Cursor.Style = theme.Fg(m.theme, m.theme.Accent)
		m.input.PlaceholderStyle = theme.Fg(m.theme, m.theme.Subtle)
		return m, nil
	case msgs.ProjectsLoadedMsg:
		m.context = projectContext(msg.Items)
		return m, nil
	case completionMsg:
		m.waiting = false
		if msg.err != nil {
			m.errText = msg.err.Error()
			return m, nil
		}
		m.errText = ""
		if strings.TrimSpace(msg.text) != "" {
			m.messages = append(m.messages, domain.Message{
				Role:    domain.MessageRoleAssistant,
				Content: strings.TrimSpace(msg.text),
			})
		}
		m.scroll = 0
		if len(msg.edits) > 0 {
			m.pendingEdits = append(m.pendingEdits, msg.edits...)
			m.editViewMode = editModeReviewing
			m.editIdx = 0
		}
		return m, nil
	case editApprovalMsg:
		if msg.err != nil {
			m.errText = fmt.Sprintf("write failed: %s", msg.err)
		} else {
			m.messages = append(m.messages, domain.Message{
				Role:    domain.MessageRoleAssistant,
				Content: fmt.Sprintf("✓ Wrote %s", msg.edit.Path),
			})
		}
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	case tea.MouseMsg:
		return m.updateMouse(msg)
	}
	return m, nil
}

func (m *Model) updateKey(msg tea.KeyMsg) (screens.Screen, tea.Cmd) {
	// Edit approval/rejection flow takes priority.
	if m.editViewMode == editModeReviewing {
		return m.updateEditReview(msg)
	}

	switch msg.String() {
	case "up":
		m.scrollMessages(1)
		return m, nil
	case "pgup":
		m.scrollMessages(5)
		return m, nil
	case "down":
		m.scrollMessages(-1)
		return m, nil
	case "pgdown":
		m.scrollMessages(-5)
		return m, nil
	case "tab":
		// Cycle through modes.
		m.cycleMode(1)
		return m, nil
	case "shift+tab":
		m.cycleMode(-1)
		return m, nil
	case "esc":
		if m.input.Value() != "" {
			m.input.SetValue("")
		} else if m.input.Focused() {
			m.input.Blur()
		} else {
			m.input.Focus()
			return m, textinput.Blink
		}
		m.errText = ""
		return m, nil
	case "enter":
		if m.waiting {
			return m, nil
		}
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			return m, nil
		}
		m.messages = append(m.messages, domain.Message{Role: domain.MessageRoleUser, Content: text})
		m.input.SetValue("")
		m.input.Focus()
		m.waiting = true
		m.scroll = 0
		m.errText = ""
		return m, m.completeCmd(m.messages, m.context, m.mode)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) updateEditReview(msg tea.KeyMsg) (screens.Screen, tea.Cmd) {
	if m.editIdx >= len(m.pendingEdits) {
		m.editViewMode = editModeNone
		m.pendingEdits = nil
		return m, nil
	}
	edit := m.pendingEdits[m.editIdx]
	switch msg.String() {
	case "y", "enter":
		// Approve: write the file, advance.
		m.editIdx++
		if m.editIdx >= len(m.pendingEdits) {
			m.editViewMode = editModeNone
			m.pendingEdits = nil
		}
		return m, writeEditCmd(edit, m.workDir)
	case "n", "esc":
		// Reject: skip this edit, advance.
		m.messages = append(m.messages, domain.Message{
			Role:    domain.MessageRoleAssistant,
			Content: fmt.Sprintf("✗ Skipped edit to %s", edit.Path),
		})
		m.editIdx++
		if m.editIdx >= len(m.pendingEdits) {
			m.editViewMode = editModeNone
			m.pendingEdits = nil
		}
		return m, nil
	case "tab":
		// Peek at next edit without deciding.
		if m.editIdx+1 < len(m.pendingEdits) {
			m.editIdx++
		}
		return m, nil
	case "shift+tab":
		if m.editIdx > 0 {
			m.editIdx--
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) updateMouse(msg tea.MouseMsg) (screens.Screen, tea.Cmd) {
	event := tea.MouseEvent(msg)
	switch event.Button {
	case tea.MouseButtonWheelUp:
		m.scrollMessages(3)
	case tea.MouseButtonWheelDown:
		m.scrollMessages(-3)
	}
	return m, nil
}

func (m *Model) cycleMode(delta int) {
	all := domain.AllModes()
	for i, mo := range all {
		if mo == m.mode {
			next := (i + delta + len(all)) % len(all)
			m.mode = all[next]
			return
		}
	}
	m.mode = domain.ModeClarify
}

func (m *Model) scrollMessages(delta int) {
	m.scroll += delta
	if m.scroll < 0 {
		m.scroll = 0
	}
}

func (m *Model) completeCmd(messages []domain.Message, ctx domain.ProjectContext, mode domain.Mode) tea.Cmd {
	provider := m.provider
	copied := append([]domain.Message(nil), messages...)
	return func() tea.Msg {
		if provider == nil {
			return completionMsg{err: fmt.Errorf("no AI provider configured")}
		}
		resp, err := provider.Complete(context.Background(), domain.CompletionRequest{
			Messages: copied,
			Context:  ctx,
			Mode:     mode,
		})
		if err != nil {
			return completionMsg{err: err}
		}
		return completionMsg{text: resp.Text, edits: resp.ProposedEdits}
	}
}

func writeEditCmd(edit domain.ProposedEdit, workDir string) tea.Cmd {
	return func() tea.Msg {
		if workDir == "" {
			return editApprovalMsg{edit: edit, err: fmt.Errorf("no workspace directory set")}
		}
		path := filepath.Join(workDir, filepath.FromSlash(edit.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return editApprovalMsg{edit: edit, err: err}
		}
		if err := os.WriteFile(path, []byte(edit.Content), 0o644); err != nil {
			return editApprovalMsg{edit: edit, err: err}
		}
		return editApprovalMsg{edit: edit, workDir: workDir}
	}
}

// ─── View ──────────────────────────────────────────────────────────────────

func (m *Model) View(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	// Edit-review overlay takes the whole content area.
	if m.editViewMode == editModeReviewing && len(m.pendingEdits) > 0 {
		return m.editReviewView(width, height)
	}

	header := theme.ApplyGradOn("AI Guide", m.theme.GradientFrom, m.theme.GradientTo, m.theme.Background, true)
	providerLine := m.providerLine()
	modeLine := m.modeBar(min(width-8, 72))

	messageW := max(24, min(width-8, 72))
	compact := height < 24
	contextPanel := m.renderContext(messageW, compact)

	input := m.renderInput(messageW)
	hint := m.hintLine()
	hintRendered := theme.Fg(m.theme, m.theme.Subtle).Render(hint)

	gaps := 0
	if !compact {
		gaps = 3
	}
	fixedH := lipgloss.Height(header) + lipgloss.Height(providerLine) +
		lipgloss.Height(modeLine) + lipgloss.Height(contextPanel) +
		lipgloss.Height(input) + lipgloss.Height(hintRendered) + gaps
	messageH := max(1, height-fixedH-2)
	messages := m.renderMessages(messageW, messageH)

	parts := []string{header, providerLine, modeLine}
	if !compact {
		parts = append(parts, "")
	}
	parts = append(parts, contextPanel)
	if !compact {
		parts = append(parts, "")
	}
	parts = append(parts, messages)
	if !compact {
		parts = append(parts, "")
	}
	parts = append(parts, input, hintRendered)
	if m.errText != "" {
		parts = append(parts, theme.Fg(m.theme, m.theme.Danger).Render(m.errText))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return theme.Place(m.theme, width, height, lipgloss.Center, lipgloss.Center, body)
}

func (m *Model) hintLine() string {
	if m.waiting {
		return "waiting for AI response…"
	}
	if m.input.Focused() {
		return "enter send  tab mode  ↑↓ scroll  esc clear"
	}
	return "↑↓ scroll  tab mode  esc focus  q quit"
}

func (m *Model) providerLine() string {
	var label string
	switch m.providerName {
	case "claude":
		label = "claude · real provider"
	default:
		label = "mock provider · local only"
	}
	color := m.theme.Success
	if m.providerName != "claude" {
		color = m.theme.Muted
	}
	return theme.Fg(m.theme, color).Render(label)
}

func (m *Model) modeBar(width int) string {
	all := domain.AllModes()
	var parts []string
	for _, mo := range all {
		label := mo.Label()
		if mo == m.mode {
			parts = append(parts, theme.Fg(m.theme, m.theme.Primary).Bold(true).
				Background(m.theme.Surface).
				Padding(0, 1).
				Render(label))
		} else {
			parts = append(parts, theme.Fg(m.theme, m.theme.Subtle).Padding(0, 1).Render(label))
		}
	}
	bar := strings.Join(parts, " ")
	return theme.Base(m.theme).Width(width).Render(bar)
}

func (m *Model) editReviewView(width, height int) string {
	if m.editIdx >= len(m.pendingEdits) {
		return ""
	}
	edit := m.pendingEdits[m.editIdx]

	header := theme.ApplyGradOn("Proposed Edit", m.theme.GradientFrom, m.theme.GradientTo, m.theme.Background, true)
	counter := theme.Fg(m.theme, m.theme.Muted).Render(
		fmt.Sprintf("edit %d of %d", m.editIdx+1, len(m.pendingEdits)))
	pathLine := theme.Fg(m.theme, m.theme.Accent).Bold(true).Render(edit.Path)
	desc := theme.Fg(m.theme, m.theme.Foreground).Render(edit.Description)

	previewW := min(width-8, 72)
	previewH := max(4, height-12)
	preview := m.renderEditPreview(edit.Content, previewW, previewH)

	hint := theme.Fg(m.theme, m.theme.Subtle).Render(
		"y / enter  approve & write    n / esc  reject    tab  next edit")

	body := lipgloss.JoinVertical(lipgloss.Left,
		header, counter, "", pathLine, desc, "", preview, "", hint,
	)
	return theme.Place(m.theme, width, height, lipgloss.Center, lipgloss.Center, body)
}

func (m *Model) renderEditPreview(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
		lines = append(lines, theme.Fg(m.theme, m.theme.Muted).Render("…"))
	}
	var rendered []string
	for _, l := range lines {
		rendered = append(rendered, theme.Fg(m.theme, m.theme.Foreground).Render(l))
	}
	return theme.Base(m.theme).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderFocus).
		Width(width).
		Padding(0, 1).
		Render(strings.Join(rendered, "\n"))
}

func (m *Model) renderContext(width int, compact bool) string {
	title := strings.TrimSpace(m.context.Title)
	if title == "" {
		title = "no project context loaded"
	}
	status := string(m.context.Status)
	if status == "" {
		status = "none"
	}
	audience := strings.TrimSpace(m.context.TargetAudience)
	if audience == "" {
		audience = "not set"
	}

	if compact {
		line := "context " + title + " · " + status
		return theme.Fg(m.theme, m.theme.Muted).Render(truncate(line, width))
	}

	lines := []string{
		theme.Fg(m.theme, m.theme.Subtle).Render("context"),
		theme.Fg(m.theme, m.theme.Foreground).Render(truncate(title, max(8, width-4))),
		theme.Fg(m.theme, m.theme.Muted).Render("status " + status + " · audience " + truncate(audience, max(8, width-22))),
	}
	return theme.Base(m.theme).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Width(width).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

func (m *Model) renderMessages(width, height int) string {
	if height < 1 {
		height = 1
	}
	rows := m.messageRows(width)
	if m.waiting {
		rows = append(rows, theme.Fg(m.theme, m.theme.Muted).Render("AI thinking…"))
	}
	if len(rows) == 0 {
		rows = append(rows, theme.Fg(m.theme, m.theme.Muted).Render("No messages yet."))
	}
	visible := visibleRows(rows, height, m.scroll)
	return theme.Base(m.theme).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Width(width).
		Height(height).
		Padding(0, 1).
		Render(strings.Join(visible, "\n"))
}

func (m *Model) messageRows(width int) []string {
	var rows []string
	for _, msg := range m.messages {
		label := "AI"
		color := m.theme.Accent
		if msg.Role == domain.MessageRoleUser {
			label = "You"
			color = m.theme.Primary
		}
		prefix := theme.Fg(m.theme, color).Bold(true).Render(label + " ")
		content := wrapText(strings.TrimSpace(msg.Content), max(12, width-8))
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if i == 0 {
				rows = append(rows, prefix+theme.Fg(m.theme, m.theme.Foreground).Render(line))
			} else {
				rows = append(rows, strings.Repeat(" ", 4)+theme.Fg(m.theme, m.theme.Foreground).Render(line))
			}
		}
	}
	return rows
}

func visibleRows(rows []string, height int, scroll int) []string {
	if height >= len(rows) {
		return rows
	}
	maxScroll := len(rows) - height
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	start := len(rows) - height - scroll
	return rows[start : start+height]
}

func (m *Model) renderInput(width int) string {
	border := m.theme.Border
	if m.input.Focused() {
		border = m.theme.BorderFocus
	}
	return theme.Base(m.theme).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(width).
		Padding(0, 1).
		Render(m.input.View())
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func projectContext(projects []domain.Project) domain.ProjectContext {
	if len(projects) == 0 {
		return domain.ProjectContext{}
	}
	p := projects[0]
	for _, candidate := range projects {
		if candidate.Status == domain.ProjectStatusActive {
			p = candidate
			break
		}
	}
	return domain.ProjectContext{
		ProjectID:      p.ID,
		Title:          p.Title,
		Status:         p.Status,
		Description:    bodySection(p.Body, "Description"),
		Architecture:   bodySection(p.Body, "Architecture"),
		TargetAudience: firstNonEmpty(p.TargetAudience, bodySection(p.Body, "Target Audience")),
		Roadmap:        bodySection(p.Body, "Roadmap"),
	}
}

func bodySection(body, heading string) string {
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func wrapText(s string, width int) string {
	if width < 1 {
		width = 1
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	var line string
	for _, word := range words {
		if line == "" {
			line = word
			continue
		}
		if len([]rune(line))+1+len([]rune(word)) > width {
			lines = append(lines, line)
			line = word
			continue
		}
		line += " " + word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func truncate(s string, maxLen int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= maxLen {
		return string(runes)
	}
	if maxLen <= 1 {
		return string(runes[:max(0, maxLen)])
	}
	return string(runes[:maxLen-1]) + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
