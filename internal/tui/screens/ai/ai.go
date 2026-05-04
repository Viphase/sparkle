package ai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	guide "github.com/viphase/sparkle/internal/ai"
	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tracker"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/theme"
)

type completionMsg struct {
	text          string
	edits         []domain.ProposedEdit
	quizzes       []domain.Quiz
	stageComplete bool
	err           error
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
	lastH         int // last rendered height (for mouse hit-testing)
	lastW         int // last rendered width (for mouse hit-testing)

	// Quiz state: when the AI returns a quiz, it's shown as an interactive
	// widget. The user picks an option with a letter key or arrow + enter.
	pendingQuizzes []domain.Quiz
	quizIdx        int // index into pendingQuizzes for the current quiz
	quizCursor     int // selected choice within the current quiz (0-based)

	// Pipeline / artifact tracking (M8).
	visitedStages map[domain.Mode]bool // stages the user has been to
	stageAdvise   bool                 // AI signalled stage-complete last turn
	artifacts     domain.ArtifactStatus

	// Active skill (M9) — injectable prompt specialisation.
	skill domain.Skill

	// Tracking data (M10) — kept so the AI context can include progress stats.
	allEvents map[string][]domain.TrackingEvent
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
		theme:         t,
		provider:      provider,
		providerName:  providerName,
		input:         ti,
		workDir:       workDir,
		mode:          domain.ModeClarify,
		visitedStages: map[domain.Mode]bool{domain.ModeClarify: true},
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
func (m *Model) InForm() bool {
	// Block global keys (q, tab, etc.) while editing text, reviewing an edit,
	// or answering a quiz so the user's keystrokes go to the right handler.
	return (m.input.Focused() || m.hasPendingQuiz()) && m.editViewMode == editModeNone
}

func (m *Model) hasPendingQuiz() bool {
	return len(m.pendingQuizzes) > 0 && m.quizIdx < len(m.pendingQuizzes)
}

func (m *Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.ThemeChangedMsg:
		m.theme = theme.ByName(msg.ThemeName)
		m.input.PromptStyle = theme.Fg(m.theme, m.theme.Primary)
		m.input.TextStyle = theme.Fg(m.theme, m.theme.Foreground)
		m.input.Cursor.Style = theme.Fg(m.theme, m.theme.Accent)
		m.input.PlaceholderStyle = theme.Fg(m.theme, m.theme.Subtle)
		return m, nil
	case msgs.SkillChangedMsg:
		m.skill = domain.Skill(msg.Skill)
		return m, nil
	case msgs.TrackingLoadedMsg:
		m.allEvents = msg.AllEvents
		// Refresh tracking fields in current context if a project is loaded.
		if m.context.ProjectID != "" {
			m.context = enrichContextWithTracking(m.context, msg.AllEvents)
		}
		return m, nil
	case msgs.ProjectsLoadedMsg:
		m.context = projectContext(msg.Items)
		return m, nil
	case msgs.ProjectContextMsg:
		// Build context from the newly promoted project.
		m.context = domain.ProjectContext{
			ProjectID:      msg.Project.ID,
			Title:          msg.Project.Title,
			Status:         msg.Project.Status,
			TargetAudience: msg.Project.TargetAudience,
			Description:    bodySection(msg.Project.Body, "Description"),
			Architecture:   bodySection(msg.Project.Body, "Architecture"),
			Roadmap:        bodySection(msg.Project.Body, "Roadmap"),
		}
		m.artifacts = domain.ArtifactStatusFromContext(m.context)
		// Enrich with tracking data if we already have events.
		if m.allEvents != nil {
			m.context = enrichContextWithTracking(m.context, m.allEvents)
		}
		// Reset conversation for this project and auto-trigger the AI.
		m.messages = []domain.Message{
			{Role: domain.MessageRoleAssistant, Content: "I'm ready to help shape \"" + msg.Project.Title + "\" into a concrete project. Let me start with a few key questions."},
		}
		m.scroll = 0
		m.pendingQuizzes = nil
		m.quizIdx = 0
		m.stageAdvise = false
		m.visitedStages = map[domain.Mode]bool{domain.ModeClarify: true}
		m.mode = domain.ModeClarify
		m.errText = ""
		m.waiting = true
		return m, m.completeCmd(m.messages, m.context, domain.ModeClarify)
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
		// Track stage-complete signal from the AI.
		m.stageAdvise = msg.stageComplete
		// Update artifact status from context (written edits may have enriched it).
		m.artifacts = domain.ArtifactStatusFromContext(m.context)
		if len(msg.edits) > 0 {
			m.pendingEdits = append(m.pendingEdits, msg.edits...)
			m.editViewMode = editModeReviewing
			m.editIdx = 0
		}
		if len(msg.quizzes) > 0 {
			m.pendingQuizzes = append(m.pendingQuizzes, msg.quizzes...)
			m.quizCursor = 0
			// Blur the text input — keyboard focus moves to the quiz widget.
			m.input.Blur()
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
	// Edit approval flow takes priority.
	if m.editViewMode == editModeReviewing {
		return m.updateEditReview(msg)
	}
	// Quiz widget takes priority over free-text input.
	if m.hasPendingQuiz() {
		return m.updateQuiz(msg)
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

// updateQuiz handles keyboard input while a quiz widget is showing.
func (m *Model) updateQuiz(msg tea.KeyMsg) (screens.Screen, tea.Cmd) {
	quiz := m.pendingQuizzes[m.quizIdx]
	switch msg.String() {
	case "up", "k":
		if m.quizCursor > 0 {
			m.quizCursor--
		}
	case "down", "j":
		if m.quizCursor+1 < len(quiz.Choices) {
			m.quizCursor++
		}
	case "enter":
		return m.submitQuizChoice(quiz, m.quizCursor)
	case "esc":
		// Dismiss quiz without answering; restore text input.
		return m.advanceQuiz()
	default:
		// Letter keys (a, b, c, d, …) map directly to choice keys.
		key := msg.String()
		for i, choice := range quiz.Choices {
			if choice.Key == key {
				return m.submitQuizChoice(quiz, i)
			}
		}
	}
	return m, nil
}

// advanceQuiz moves past the current quiz. If no more quizzes remain the
// pending list is cleared and the text input regains focus.
func (m *Model) advanceQuiz() (screens.Screen, tea.Cmd) {
	m.quizIdx++
	if !m.hasPendingQuiz() {
		m.pendingQuizzes = nil
		m.quizIdx = 0
		m.input.Focus()
		return m, textinput.Blink
	}
	m.quizCursor = 0
	return m, nil
}

// submitQuizChoice records the selected answer as a user message and sends it
// to the AI provider, then advances past the current quiz.
func (m *Model) submitQuizChoice(quiz domain.Quiz, idx int) (screens.Screen, tea.Cmd) {
	if idx < 0 || idx >= len(quiz.Choices) {
		return m, nil
	}
	choice := quiz.Choices[idx]
	answer := choice.Key + ") " + choice.Text

	// Advance quiz state first so the widget disappears immediately.
	m.quizIdx++
	m.quizCursor = 0
	if !m.hasPendingQuiz() {
		m.pendingQuizzes = nil
		m.quizIdx = 0
	}

	// Append choice as user message and send to AI.
	m.messages = append(m.messages, domain.Message{
		Role:    domain.MessageRoleUser,
		Content: answer,
	})
	m.waiting = true
	m.scroll = 0
	m.errText = ""
	m.input.Focus()
	return m, m.completeCmd(m.messages, m.context, m.mode)
}

func (m *Model) updateMouse(msg tea.MouseMsg) (screens.Screen, tea.Cmd) {
	event := tea.MouseEvent(msg)
	switch event.Button {
	case tea.MouseButtonWheelUp:
		m.scrollMessages(3)
	case tea.MouseButtonWheelDown:
		m.scrollMessages(-3)
	}
	switch msg.Type {
	case tea.MouseWheelUp:
		m.scrollMessages(3)
	case tea.MouseWheelDown:
		m.scrollMessages(-3)
	case tea.MouseLeft:
		// Mode bar is in the top portion of the content area (rows 0-5).
		if msg.Y <= 5 {
			m.mode = m.modeAtX(msg.X, m.lastW)
		}
	}
	return m, nil
}

// modeAtX returns the Mode corresponding to a click at the given X coordinate
// in the mode bar row. Falls back to the current mode if the click misses.
func (m *Model) modeAtX(clickX, contentW int) domain.Mode {
	all := domain.AllModes()
	messageW := max(24, min(contentW-8, 72))
	if contentW < 60 {
		messageW = max(24, contentW-4)
	}
	// Body is centered: bodyX = (contentW - messageW) / 2
	bodyX := (contentW - messageW) / 2
	relX := clickX - bodyX
	if relX < 0 {
		return m.mode
	}
	x := 0
	for _, mo := range all {
		label := " " + mo.Label() + " " // Padding(0,1)
		labelW := len([]rune(label))
		if relX >= x && relX < x+labelW {
			// Record visit and clear advise.
			if m.visitedStages == nil {
				m.visitedStages = make(map[domain.Mode]bool)
			}
			m.visitedStages[mo] = true
			m.stageAdvise = false
			return mo
		}
		x += labelW + 1 // +1 for the space separator
	}
	return m.mode
}

func (m *Model) cycleMode(delta int) {
	all := domain.AllModes()
	for i, mo := range all {
		if mo == m.mode {
			next := (i + delta + len(all)) % len(all)
			m.mode = all[next]
			if m.visitedStages == nil {
				m.visitedStages = make(map[domain.Mode]bool)
			}
			m.visitedStages[m.mode] = true
			m.stageAdvise = false // dismiss advise when user acts on it
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
	skill := m.skill
	copied := append([]domain.Message(nil), messages...)
	return func() tea.Msg {
		if provider == nil {
			return completionMsg{err: fmt.Errorf("no AI provider configured")}
		}
		resp, err := provider.Complete(context.Background(), domain.CompletionRequest{
			Messages: copied,
			Context:  ctx,
			Mode:     mode,
			Skill:    skill,
		})
		if err != nil {
			return completionMsg{err: err}
		}
		return completionMsg{text: resp.Text, edits: resp.ProposedEdits, quizzes: resp.Quizzes, stageComplete: resp.StageComplete}
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
	m.lastH = height
	m.lastW = width

	// Edit-review overlay takes the whole content area.
	if m.editViewMode == editModeReviewing && len(m.pendingEdits) > 0 {
		return m.editReviewView(width, height)
	}

	header := theme.ApplyGradOn("AI Guide", m.theme.GradientFrom, m.theme.GradientTo, m.theme.Background, true)
	providerLine := m.providerLine()

	// Body width used for pipeline, mode bar, and messages.
	var bodyW int
	if width < 60 {
		bodyW = max(24, width-4)
	} else {
		bodyW = max(24, min(width-8, 72))
	}

	// Pipeline replaces the old flat mode bar when height allows.
	compact := height < 24
	var pipelineLine, modeLine string
	if compact || width < 60 {
		// Compact: fall back to the slim mode bar only.
		modeLine = m.modeBar(bodyW)
	} else {
		// Full: pipeline stage tracker + slim mode bar on separate lines.
		pipelineLine = m.renderPipeline(bodyW)
		modeLine = m.modeBar(bodyW)
	}

	messageW := bodyW
	contextPanel := m.renderContext(messageW, compact, width < 60)
	artifactLine := m.renderArtifacts(messageW)

	input := m.renderInput(messageW)
	hint := m.hintLine()
	hintColor := m.theme.Subtle
	if m.stageAdvise {
		hintColor = m.theme.Accent // highlight hint when stage-complete
	}
	hintRendered := theme.Fg(m.theme, hintColor).Render(hint)

	// Pre-render quiz widget (if any) to measure its height.
	quizWidget := m.renderQuiz(messageW)
	quizH := 0
	if quizWidget != "" {
		quizH = lipgloss.Height(quizWidget) + 1 // +1 for the gap below it
	}

	// Compute fixed height consumed by non-message elements.
	fixedH := lipgloss.Height(header) + lipgloss.Height(providerLine)
	if pipelineLine != "" {
		fixedH += lipgloss.Height(pipelineLine)
	}
	fixedH += lipgloss.Height(modeLine) + lipgloss.Height(contextPanel) +
		lipgloss.Height(artifactLine) + lipgloss.Height(input) +
		lipgloss.Height(hintRendered) + quizH

	gaps := 0
	if !compact {
		gaps = 4 // header→pipeline, pipeline→mode, context→msgs, msgs→input gaps
	}
	fixedH += gaps
	messageH := max(1, height-fixedH-2)
	messages := m.renderMessages(messageW, messageH)

	// Assemble body parts.
	parts := []string{header, providerLine}
	if pipelineLine != "" {
		parts = append(parts, pipelineLine)
	}
	parts = append(parts, modeLine)
	if !compact {
		parts = append(parts, "")
	}
	parts = append(parts, contextPanel, artifactLine)
	if !compact {
		parts = append(parts, "")
	}
	parts = append(parts, messages)
	if !compact {
		parts = append(parts, "")
	}
	if quizWidget != "" {
		parts = append(parts, quizWidget)
		if !compact {
			parts = append(parts, "")
		}
	}
	parts = append(parts, input, hintRendered)
	if m.errText != "" {
		parts = append(parts, theme.Fg(m.theme, m.theme.Danger).Render(m.errText))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return theme.Place(m.theme, width, height, lipgloss.Center, lipgloss.Center, body)
}

func (m *Model) hintLine() string {
	if m.hasPendingQuiz() {
		return "a-d / ↑↓ choose   enter confirm   esc skip"
	}
	if m.waiting {
		return "waiting for AI response…"
	}
	if m.stageAdvise {
		all := domain.AllModes()
		for i, mo := range all {
			if mo == m.mode && i+1 < len(all) {
				return "stage done · tab → " + all[i+1].Label() + "   enter send   ↑↓ scroll"
			}
		}
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
	if m.skill != domain.SkillNone {
		label += "  ·  skill: " + m.skill.Label()
	}
	color := m.theme.Success
	if m.providerName != "claude" {
		color = m.theme.Muted
	}
	return theme.Fg(m.theme, color).Render(label)
}

// renderPipeline renders the 6 pipeline stages as a visual flow indicator.
// Visited stages show a checkmark, the active stage is highlighted, and future
// stages are muted. An arrow (→) separates each stage.
func (m *Model) renderPipeline(width int) string {
	all := domain.AllModes()
	var parts []string
	for i, mo := range all {
		label := mo.Label()
		var pill string
		switch {
		case mo == m.mode:
			// Active stage: bold primary pill.
			pill = theme.Fg(m.theme, m.theme.Primary).Bold(true).
				Background(m.theme.Surface).
				Padding(0, 1).
				Render("● " + label)
		case m.visitedStages[mo]:
			// Visited stage: success checkmark, muted.
			pill = theme.Fg(m.theme, m.theme.Success).
				Padding(0, 1).
				Render("✓ " + label)
		default:
			// Future stage: very subtle.
			pill = theme.Fg(m.theme, m.theme.Subtle).
				Padding(0, 1).
				Render("· " + label)
		}
		parts = append(parts, pill)
		if i < len(all)-1 {
			arrow := theme.Fg(m.theme, m.theme.Subtle).Render("→")
			parts = append(parts, arrow)
		}
	}
	bar := strings.Join(parts, "")
	return theme.Base(m.theme).Width(width).Render(bar)
}

// renderArtifacts renders a compact one-line checklist of the 7 tracked
// project artifacts. Filled artifacts show ✓ (success), empty show · (subtle).
func (m *Model) renderArtifacts(width int) string {
	type item struct {
		label string
		done  bool
	}
	items := []item{
		{"desc", m.artifacts.Description},
		{"arch", m.artifacts.Architecture},
		{"audience", m.artifacts.TargetAudience},
		{"roadmap", m.artifacts.Roadmap},
		{"notes", m.artifacts.Notes},
		{"flaws", m.artifacts.Flaws},
		{"plan", m.artifacts.Plan},
	}
	filled := m.artifacts.FilledCount()
	header := theme.Fg(m.theme, m.theme.Subtle).Render("artifacts ")
	counter := theme.Fg(m.theme, m.theme.Muted).Render(
		fmt.Sprintf("%d/7  ", filled))

	var chips []string
	for _, it := range items {
		if it.done {
			chips = append(chips, theme.Fg(m.theme, m.theme.Success).Render("✓ "+it.label))
		} else {
			chips = append(chips, theme.Fg(m.theme, m.theme.Subtle).Render("· "+it.label))
		}
	}
	line := header + counter + strings.Join(chips, "  ")
	return theme.Base(m.theme).Width(width).Render(truncate(line, max(8, width)))
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

func (m *Model) renderContext(width int, compact bool, narrow ...bool) string {
	isNarrow := len(narrow) > 0 && narrow[0]
	_ = isNarrow // used below to drop border when narrow
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
	if isNarrow {
		// Drop the border when the window is too narrow to afford it.
		return theme.Base(m.theme).
			Width(width).
			Render(strings.Join(lines, "\n"))
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

// renderQuiz draws the interactive multiple-choice widget for the current quiz.
func (m *Model) renderQuiz(width int) string {
	if !m.hasPendingQuiz() {
		return ""
	}
	quiz := m.pendingQuizzes[m.quizIdx]

	questionStyle := theme.Fg(m.theme, m.theme.Foreground).Bold(true)
	question := questionStyle.Render(wrapText(quiz.Question, max(16, width-4)))

	var choiceLines []string
	for i, choice := range quiz.Choices {
		prefix := "  " + choice.Key + ")  "
		text := wrapText(choice.Text, max(16, width-len([]rune(prefix))-4))
		if i == m.quizCursor {
			bullet := theme.Fg(m.theme, m.theme.Primary).Bold(true).Render("▌ " + choice.Key + ")")
			val := theme.Fg(m.theme, m.theme.Primary).Bold(true).Render("  " + text)
			choiceLines = append(choiceLines, bullet+val)
		} else {
			label := theme.Fg(m.theme, m.theme.Muted).Render("  " + choice.Key + ")")
			val := theme.Fg(m.theme, m.theme.Foreground).Render("  " + text)
			choiceLines = append(choiceLines, label+val)
		}
	}

	// Counter for multi-quiz responses.
	header := theme.Fg(m.theme, m.theme.Accent).Bold(true).Render("quiz")
	if len(m.pendingQuizzes) > 1 {
		counter := theme.Fg(m.theme, m.theme.Muted).Render(
			fmt.Sprintf("  %d of %d", m.quizIdx+1, len(m.pendingQuizzes)))
		header = header + counter
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		question,
		"",
		strings.Join(choiceLines, "\n"),
	)
	return theme.Base(m.theme).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent).
		Width(width).
		Padding(0, 1).
		Render(body)
}

// ─── Helpers ───────────────────────────────────────────────────────────────

// enrichContextWithTracking adds live tracking stats for the context's project
// into the ProjectContext fields used by the AI prompt (M10).
func enrichContextWithTracking(ctx domain.ProjectContext, allEvents map[string][]domain.TrackingEvent) domain.ProjectContext {
	if ctx.ProjectID == "" || allEvents == nil {
		return ctx
	}
	events := allEvents[ctx.ProjectID]
	if len(events) == 0 {
		return ctx
	}
	now := time.Now()
	stats := tracker.Compute(events, now)
	ctx.TodayWords = stats.TodayWords
	ctx.WeekWords = stats.WeekWords
	ctx.Streak = stats.CurrentStreak
	ctx.ActiveDaysWeek = stats.ActiveDaysWeek
	if !stats.LastActive.IsZero() {
		ctx.DaysSinceActive = int(now.Sub(stats.LastActive).Hours() / 24)
	} else {
		ctx.DaysSinceActive = -1
	}
	return ctx
}

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
