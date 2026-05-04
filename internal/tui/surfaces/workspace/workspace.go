// Package workspace implements the unified Workspace surface: items rail +
// detail pane + embedded AI panel. This is the v2 replacement for the
// separate Sparks, Projects, and AI tabs.
package workspace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	guide "github.com/viphase/sparkle/internal/ai"
	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/storage/markdown"
	"github.com/viphase/sparkle/internal/tui/components/layout"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/theme"
)

// railItem is a unified rail entry representing either a spark or a project.
type railItem struct {
	kind      string // "spark" or "project"
	id        string
	title     string
	status    string
	daysSince int
}

func (r railItem) glyph() string {
	if r.kind == "project" {
		return "●"
	}
	return "✦"
}

// aiPanelState tracks what the embedded AI panel is doing.
type aiPanelState int

const (
	aiStateIdle    aiPanelState = iota
	aiStateWaiting              // request in flight
	aiStateError                // last call failed
)

type aiCompletionMsg struct {
	text    string
	quizzes []domain.Quiz
	err     error
}

type sessionLoadedMsg struct {
	messages []domain.Message
}

// sparkCapturedMsg is fired after a successful n-capture write to disk.
type sparkCapturedMsg struct {
	spark domain.Spark
	err   error
}

// detailLoadedMsg carries the full body of the selected spark or project.
type detailLoadedMsg struct {
	title   string
	body    string
	itemID  string
	kind    string
}

// Model is the v2 Workspace surface.
type Model struct {
	theme    theme.Theme
	provider guide.Provider
	workDir  string
	skill    domain.Skill

	// Rail state
	items  []railItem
	cursor int

	// Detail pane
	detailTitle   string
	detailBody    string // full loaded body text
	detailScroll  int
	detailKind    string // "spark" or "project"
	detailID      string // ID of the currently displayed item

	// Inline editing (L3)
	editing      bool
	editor       textarea.Model
	editOriginal string // body before the edit began (for esc-to-revert)

	// Inline spark capture (n)
	capturing   bool
	captureIn   textinput.Model

	// AI panel
	aiMessages        []domain.Message
	aiInput           textinput.Model
	aiState           aiPanelState
	aiErrText         string
	aiVisible         bool // toggled with 'i' in medium layouts
	aiContext         domain.ProjectContext
	aiScroll          int
	currentProjectID  string // for session tracking

	// Quiz state: shown inline in the AI panel when the provider returns one.
	pendingQuiz *domain.Quiz
	quizCursor  int

	// Layout
	lastW int
	lastH int
}

func New(t theme.Theme, workDir string, provider ...guide.Provider) *Model {
	ti := textinput.New()
	ti.Placeholder = "Ask anything about this project…"
	ti.CharLimit = 500

	cap := textinput.New()
	cap.Placeholder = "What's the spark? (one short title)"
	cap.CharLimit = 120

	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.CharLimit = 0 // unlimited
	ta.SetWidth(60)
	ta.SetHeight(10)

	m := &Model{
		theme:     t,
		workDir:   workDir,
		aiVisible: false,
		editor:    ta,
		captureIn: cap,
	}
	if len(provider) > 0 && provider[0] != nil {
		m.provider = provider[0]
	}
	m.aiInput = ti
	return m
}

// SetSkill updates the active AI skill without rebuilding the model.
func (m *Model) SetSkill(slug string) {
	m.skill = domain.Skill(slug)
}

// SetProvider swaps the AI provider (called when the API key changes).
func (m *Model) SetProvider(p guide.Provider) {
	m.provider = p
}

func (m *Model) Title() string { return "Workspace" }

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch v := msg.(type) {
	case msgs.SparksLoadedMsg:
		m.rebuildRail(v.Items, m.projectItems())
	case msgs.ProjectsLoadedMsg:
		m.rebuildRailFromProjects(v.Items)
	case msgs.ProjectContextMsg:
		// Promoted spark: switch AI context, show panel, fire first question.
		m.aiContext = domain.ProjectContext{
			ProjectID: v.Project.ID,
			Title:     v.Project.Title,
			Status:    v.Project.Status,
		}
		m.currentProjectID = v.Project.ID
		m.aiMessages = nil
		m.pendingQuiz = nil
		m.aiScroll = 0
		m.aiVisible = true
		m.aiState = aiStateWaiting
		return m, tea.Batch(
			m.loadSessionCmd(v.Project.ID),
			m.sendToAICmd(nil, true),
		)
	case msgs.ThemeChangedMsg:
		m.theme = theme.ByName(v.ThemeName)
	case msgs.SkillChangedMsg:
		m.skill = domain.Skill(v.Skill)
	case msgs.APIKeyChangedMsg:
		if v.Key != "" {
			m.provider = guide.NewAnthropicProvider(v.Key, v.Model)
		} else {
			m.provider = nil
		}
	case sessionLoadedMsg:
		if len(v.messages) > 0 {
			m.aiMessages = v.messages
		}
		// If no history exists, fire the initial AI question now.
		if len(m.aiMessages) == 0 && m.aiState == aiStateWaiting {
			return m, m.sendToAICmd(nil, true)
		}
		m.aiState = aiStateIdle
	case aiCompletionMsg:
		m.aiState = aiStateIdle
		if v.err != nil {
			m.aiErrText = v.err.Error()
			m.aiState = aiStateError
		} else {
			if strings.TrimSpace(v.text) != "" {
				msg := domain.Message{Role: domain.MessageRoleAssistant, Content: strings.TrimSpace(v.text)}
				m.aiMessages = append(m.aiMessages, msg)
				m.appendSessionTurnCmd(msg)
			}
			if len(v.quizzes) > 0 {
				m.pendingQuiz = &v.quizzes[0]
				m.quizCursor = 0
				m.aiInput.Blur()
			}
			m.aiScroll = 999999
		}
	case sparkCapturedMsg:
		if v.err != nil {
			return m, func() tea.Msg { return msgs.ErrorMsg{Source: "capture", Err: v.err} }
		}
		// Insert at top of rail and select.
		newItem := railItem{kind: "spark", id: v.spark.ID, title: v.spark.Title, status: string(v.spark.Status)}
		m.items = append([]railItem{newItem}, m.items...)
		m.cursor = 0
		m.detailID = v.spark.ID
		m.detailKind = "spark"
		m.detailTitle = "✦  " + v.spark.Title
		m.detailBody = ""
		return m, func() tea.Msg { return msgs.StatusMsg{Text: "✦ captured"} }
	case detailLoadedMsg:
		m.detailTitle = v.title
		m.detailBody = v.body
		m.detailKind = v.kind
		m.detailID = v.itemID
		m.detailScroll = 0
	case tea.KeyMsg:
		return m.handleKey(v)
	case tea.MouseMsg:
		return m.handleMouse(v)
	}
	// Forward to editor when editing.
	if m.editing {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}
	// Forward to ai input when focused.
	if m.aiInput.Focused() {
		var cmd tea.Cmd
		m.aiInput, cmd = m.aiInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleKey(k tea.KeyMsg) (*Model, tea.Cmd) {
	// Capture modal takes priority when active.
	if m.capturing {
		switch k.String() {
		case "enter":
			title := strings.TrimSpace(m.captureIn.Value())
			if title == "" {
				m.capturing = false
				m.captureIn.Blur()
				return m, nil
			}
			m.captureIn.SetValue("")
			m.captureIn.Blur()
			m.capturing = false
			return m, m.captureSparkCmd(title)
		case "esc":
			m.captureIn.SetValue("")
			m.captureIn.Blur()
			m.capturing = false
			return m, nil
		default:
			var cmd tea.Cmd
			m.captureIn, cmd = m.captureIn.Update(k)
			return m, cmd
		}
	}

	// L3: Editor input takes priority when editing.
	if m.editing {
		switch k.String() {
		case "ctrl+s":
			return m.commitEdit()
		case "esc":
			// Revert to original content.
			m.detailBody = m.editOriginal
			m.editing = false
			m.editor.Blur()
			return m, nil
		default:
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(k)
			return m, cmd
		}
	}

	// Quiz input takes priority when a quiz is pending.
	if m.pendingQuiz != nil {
		return m.handleQuizKey(k)
	}

	if m.aiInput.Focused() {
		switch k.String() {
		case "enter":
			text := strings.TrimSpace(m.aiInput.Value())
			if text == "" {
				return m, nil
			}
			m.aiInput.SetValue("")
			m.aiInput.Blur()
			userMsg := domain.Message{Role: domain.MessageRoleUser, Content: text}
			m.aiMessages = append(m.aiMessages, userMsg)
			m.aiState = aiStateWaiting
			saveCmd := m.appendSessionTurnCmdAsync(userMsg)
			return m, tea.Batch(saveCmd, m.sendToAICmd(m.aiMessages, false))
		case "esc":
			m.aiInput.Blur()
			return m, nil
		default:
			var cmd tea.Cmd
			m.aiInput, cmd = m.aiInput.Update(k)
			return m, cmd
		}
	}

	switch k.String() {
	case "esc":
		// Close AI panel or blur input if open, otherwise a no-op.
		if m.aiVisible {
			m.aiVisible = false
			m.aiInput.Blur()
		}
	case "j", "down":
		if m.cursor < len(m.items)-1 {
			m.cursor++
			return m, m.selectCurrentCmd()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			return m, m.selectCurrentCmd()
		}
	case "i":
		m.aiVisible = !m.aiVisible
	case "a":
		m.aiVisible = true
		if !m.aiInput.Focused() {
			m.aiInput.Focus()
			return m, textinput.Blink
		}
	case "e":
		// L3: enter inline edit mode for the current item's body.
		if m.detailID != "" {
			m.editOriginal = m.detailBody
			m.editor.SetValue(m.detailBody)
			m.editor.Focus()
			m.editing = true
			return m, textarea.Blink
		}
	case "n":
		// Inline capture: prompt for a one-line title and persist as spark.
		m.capturing = true
		m.captureIn.Focus()
		return m, textinput.Blink
	case "g":
		// "Get me unstuck" — special stock prompt.
		unstuck := "I don't know what to do next. What is the smallest concrete thing I should do right now?"
		userMsg := domain.Message{Role: domain.MessageRoleUser, Content: unstuck}
		m.aiMessages = append(m.aiMessages, userMsg)
		m.aiState = aiStateWaiting
		m.aiVisible = true
		saveCmd := m.appendSessionTurnCmdAsync(userMsg)
		return m, tea.Batch(saveCmd, m.sendToAICmd(m.aiMessages, false))
	case "enter":
		if len(m.items) > 0 {
			m.aiVisible = true
			m.aiInput.Focus()
			return m, textinput.Blink
		}
	// Detail pane scroll.
	case "J":
		m.detailScroll += 3
	case "K":
		if m.detailScroll >= 3 {
			m.detailScroll -= 3
		} else {
			m.detailScroll = 0
		}
	}
	return m, nil
}

func (m *Model) handleQuizKey(k tea.KeyMsg) (*Model, tea.Cmd) {
	quiz := m.pendingQuiz
	switch k.String() {
	case "up", "k":
		if m.quizCursor > 0 {
			m.quizCursor--
		}
	case "down", "j":
		if m.quizCursor+1 < len(quiz.Choices) {
			m.quizCursor++
		}
	case "enter":
		return m.submitQuiz(m.quizCursor)
	case "esc":
		m.pendingQuiz = nil
		m.aiInput.Focus()
		return m, textinput.Blink
	default:
		for i, choice := range quiz.Choices {
			if choice.Key == k.String() {
				return m.submitQuiz(i)
			}
		}
	}
	return m, nil
}

func (m *Model) submitQuiz(idx int) (*Model, tea.Cmd) {
	quiz := m.pendingQuiz
	if idx < 0 || idx >= len(quiz.Choices) {
		return m, nil
	}
	choice := quiz.Choices[idx]
	answer := choice.Key + ") " + choice.Text
	m.pendingQuiz = nil
	userMsg := domain.Message{Role: domain.MessageRoleUser, Content: answer}
	m.aiMessages = append(m.aiMessages, userMsg)
	m.aiState = aiStateWaiting
	saveCmd := m.appendSessionTurnCmdAsync(userMsg)
	return m, tea.Batch(saveCmd, m.sendToAICmd(m.aiMessages, false))
}

func (m *Model) handleMouse(msg tea.MouseMsg) (*Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseWheelDown:
		m.aiScroll += 2
	case tea.MouseWheelUp:
		m.aiScroll -= 2
		if m.aiScroll < 0 {
			m.aiScroll = 0
		}
	}
	return m, nil
}

// selectCurrentCmd loads detail for the currently highlighted rail item.
func (m *Model) selectCurrentCmd() tea.Cmd {
	if m.cursor >= len(m.items) {
		return nil
	}
	item := m.items[m.cursor]
	m.detailTitle = ""
	m.detailBody = ""
	m.detailID = item.id
	m.detailKind = item.kind
	m.detailScroll = 0
	m.editing = false
	m.aiContext = domain.ProjectContext{
		Title:  item.title,
		Status: domain.ProjectStatus(item.status),
	}

	if item.kind == "project" {
		m.currentProjectID = item.id
		// Reset AI conversation and load persisted session.
		m.aiMessages = nil
		m.pendingQuiz = nil
		m.aiState = aiStateWaiting
		return tea.Batch(
			m.loadDetailCmd(item),
			m.loadSessionCmd(item.id),
		)
	}
	m.currentProjectID = ""
	m.aiMessages = nil
	m.pendingQuiz = nil
	return m.loadDetailCmd(item)
}

// loadDetailCmd asynchronously reads a spark or project body from disk.
func (m *Model) loadDetailCmd(item railItem) tea.Cmd {
	if m.workDir == "" {
		return nil
	}
	workDir := m.workDir
	return func() tea.Msg {
		store := markdown.NewStore(workDir)
		switch item.kind {
		case "spark":
			sp, err := store.LoadSpark(item.id)
			if err != nil {
				return detailLoadedMsg{title: item.glyph() + "  " + item.title,
					body: fmt.Sprintf("(could not load: %v)", err), itemID: item.id, kind: item.kind}
			}
			body := sp.Description
			if len(sp.Tags) > 0 {
				body += "\n\ntags: " + strings.Join(sp.Tags, ", ")
			}
			body += fmt.Sprintf("\n\nstatus: %s", sp.Status)
			return detailLoadedMsg{title: "✦  " + sp.Title, body: body, itemID: item.id, kind: item.kind}
		case "project":
			p, err := store.LoadProject(item.id)
			if err != nil {
				return detailLoadedMsg{title: item.glyph() + "  " + item.title,
					body: fmt.Sprintf("(could not load: %v)", err), itemID: item.id, kind: item.kind}
			}
			body := buildProjectSummary(p)
			return detailLoadedMsg{title: "●  " + p.Title, body: body, itemID: item.id, kind: item.kind}
		}
		return detailLoadedMsg{title: item.glyph() + "  " + item.title, itemID: item.id, kind: item.kind}
	}
}

// buildProjectSummary constructs a readable summary of a project for the
// detail pane, pulling real sections from project.md via goldmark (L2).
func buildProjectSummary(p domain.Project) string {
	var sb strings.Builder
	if desc := markdown.BodySection(p.Body, "Description"); desc != "" {
		sb.WriteString("Description\n")
		sb.WriteString(strings.Repeat("─", 12) + "\n")
		sb.WriteString(desc)
		sb.WriteString("\n\n")
	}
	if ta := p.TargetAudience; ta != "" {
		sb.WriteString("Target Audience\n")
		sb.WriteString(strings.Repeat("─", 15) + "\n")
		sb.WriteString(ta)
		sb.WriteString("\n\n")
	}
	if rm := markdown.BodySection(p.Body, "Roadmap"); rm != "" {
		sb.WriteString("Roadmap\n")
		sb.WriteString(strings.Repeat("─", 7) + "\n")
		sb.WriteString(rm)
		sb.WriteString("\n\n")
	}
	if len(p.Tags) > 0 {
		sb.WriteString("tags: " + strings.Join(p.Tags, ", ") + "\n")
	}
	sb.WriteString("status: " + string(p.Status))
	return strings.TrimSpace(sb.String())
}

// captureSparkCmd persists a freshly captured spark to disk.
func (m *Model) captureSparkCmd(title string) tea.Cmd {
	if m.workDir == "" {
		return func() tea.Msg {
			return sparkCapturedMsg{err: fmt.Errorf("no workspace configured")}
		}
	}
	workDir := m.workDir
	return func() tea.Msg {
		now := time.Now()
		sp := domain.Spark{
			ID:          fmt.Sprintf("%d-%s", now.Unix(), slugify(title)),
			Title:       title,
			Description: "",
			Status:      domain.SparkStatusNew,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		store := markdown.NewStore(workDir)
		if err := store.SaveSpark(sp); err != nil {
			return sparkCapturedMsg{err: err}
		}
		return sparkCapturedMsg{spark: sp}
	}
}

// slugify reduces a free-text title to an ID-friendly fragment.
func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_':
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	if out == "" {
		out = "spark"
	}
	if len(out) > 32 {
		out = out[:32]
	}
	return out
}

// commitEdit saves the editor content to disk and exits editing mode.
func (m *Model) commitEdit() (*Model, tea.Cmd) {
	content := m.editor.Value()
	m.editing = false
	m.editor.Blur()
	m.detailBody = content // optimistically update in memory

	if m.workDir == "" || m.detailID == "" {
		return m, nil
	}
	workDir := m.workDir
	id := m.detailID
	kind := m.detailKind
	return m, func() tea.Msg {
		store := markdown.NewStore(workDir)
		switch kind {
		case "spark":
			sp, err := store.LoadSpark(id)
			if err != nil {
				return msgs.ErrorMsg{Source: "edit-spark", Err: err}
			}
			sp.Description = content
			if err := store.SaveSpark(sp); err != nil {
				return msgs.ErrorMsg{Source: "save-spark", Err: err}
			}
		case "project":
			p, err := store.LoadProject(id)
			if err != nil {
				return msgs.ErrorMsg{Source: "edit-project", Err: err}
			}
			p.Body = content
			if err := store.SaveProject(p); err != nil {
				return msgs.ErrorMsg{Source: "save-project", Err: err}
			}
		}
		return msgs.StatusMsg{Text: "saved"}
	}
}

func (m *Model) projectItems() []railItem {
	var out []railItem
	for _, item := range m.items {
		if item.kind == "project" {
			out = append(out, item)
		}
	}
	return out
}

func (m *Model) rebuildRail(sparks []domain.Spark, projects []railItem) {
	var items []railItem
	for _, s := range sparks {
		if s.Status == domain.SparkStatusArchived {
			continue
		}
		items = append(items, railItem{kind: "spark", id: s.ID, title: s.Title, status: string(s.Status)})
	}
	items = append(items, projects...)
	m.items = items
	if m.cursor >= len(m.items) && len(m.items) > 0 {
		m.cursor = len(m.items) - 1
	}
}

func (m *Model) rebuildRailFromProjects(projects []domain.Project) {
	var sparks []railItem
	for _, item := range m.items {
		if item.kind == "spark" {
			sparks = append(sparks, item)
		}
	}
	var projItems []railItem
	for _, p := range projects {
		projItems = append(projItems, railItem{
			kind:   "project",
			id:     p.ID,
			title:  p.Title,
			status: string(p.Status),
		})
	}
	m.items = append(sparks, projItems...)
	if m.cursor >= len(m.items) && len(m.items) > 0 {
		m.cursor = len(m.items) - 1
	}
}

// sendToAICmd sends the current message history to the AI provider.
// If initial is true, an empty messages slice triggers the cold-start quiz.
func (m *Model) sendToAICmd(messages []domain.Message, initial bool) tea.Cmd {
	if messages == nil {
		messages = m.aiMessages
	}
	msgsCopy := make([]domain.Message, len(messages))
	copy(msgsCopy, messages)
	ctx := m.aiContext
	provider := m.provider
	skill := m.skill
	return func() tea.Msg {
		p := provider
		if p == nil {
			p = guide.NewMockProvider()
		}
		reqCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		resp, err := p.Complete(reqCtx, domain.CompletionRequest{
			Messages: msgsCopy,
			Context:  ctx,
			Mode:     domain.ModeClarify,
			Skill:    skill,
		})
		if err != nil {
			return aiCompletionMsg{err: err}
		}
		return aiCompletionMsg{text: resp.Text, quizzes: resp.Quizzes}
	}
}

// loadSessionCmd asynchronously reads the persisted conversation for projectID.
func (m *Model) loadSessionCmd(projectID string) tea.Cmd {
	if m.workDir == "" || projectID == "" {
		return func() tea.Msg { return sessionLoadedMsg{} }
	}
	workDir := m.workDir
	return func() tea.Msg {
		msgs, err := markdown.LoadSession(workDir, projectID)
		if err != nil {
			// Not fatal — start with empty history.
			return sessionLoadedMsg{}
		}
		return sessionLoadedMsg{messages: msgs}
	}
}

// appendSessionTurnCmdAsync persists a single message turn in the background.
// Errors are silently dropped since session writes are best-effort.
func (m *Model) appendSessionTurnCmdAsync(msg domain.Message) tea.Cmd {
	if m.workDir == "" || m.currentProjectID == "" {
		return nil
	}
	workDir := m.workDir
	projectID := m.currentProjectID
	return func() tea.Msg {
		_ = markdown.AppendSessionTurn(workDir, projectID, msg)
		return nil
	}
}

// appendSessionTurnCmd is a fire-and-forget call (used when we don't need a
// tea.Cmd return value in the Update hot path).
func (m *Model) appendSessionTurnCmd(msg domain.Message) {
	if m.workDir == "" || m.currentProjectID == "" {
		return
	}
	_ = markdown.AppendSessionTurn(m.workDir, m.currentProjectID, msg)
}

func (m *Model) InForm() bool {
	return m.aiInput.Focused() || m.pendingQuiz != nil || m.editing || m.capturing
}

// IsEditing reports whether the inline body editor is open. Root uses this to
// surface "ctrl+s save · esc cancel" in the status bar while the user is
// editing.
func (m *Model) IsEditing() bool { return m.editing }

// View renders the workspace at the given dimensions using the breakpoint layout.
func (m *Model) View(width, height int) string {
	m.lastW, m.lastH = width, height
	t := m.theme

	railW := layout.RailWidth(width)
	aiW := layout.AIPanelWidth(width)

	if layout.IsWide(width) && m.aiVisible {
		// Three-column: rail | detail | AI
		detailW := width - railW - aiW - 2
		if detailW < 1 {
			detailW = 1
		}
		rail := m.renderRail(railW, height)
		detail := m.renderDetail(detailW, height)
		ai := m.renderAIPanel(aiW, height)
		sep := theme.Fg(t, t.Border).Render(strings.Repeat("│\n", height-1) + "│")
		return lipgloss.JoinHorizontal(lipgloss.Top, rail, sep, detail, sep, ai)
	}

	if layout.IsWide(width) {
		// Wide, AI hidden — two columns with hint
		detailW := width - railW - 1
		if detailW < 1 {
			detailW = 1
		}
		rail := m.renderRail(railW, height)
		detail := m.renderDetail(detailW, height)
		sep := theme.Fg(t, t.Border).Render(strings.Repeat("│\n", height-1) + "│")
		return lipgloss.JoinHorizontal(lipgloss.Top, rail, sep, detail)
	}

	if layout.IsMedium(width) {
		if m.aiVisible {
			detailW := width - railW - 1
			rail := m.renderRail(railW, height)
			ai := m.renderAIPanel(detailW, height)
			sep := theme.Fg(t, t.Border).Render(strings.Repeat("│\n", height-1) + "│")
			return lipgloss.JoinHorizontal(lipgloss.Top, rail, sep, ai)
		}
		detailW := width - railW - 1
		rail := m.renderRail(railW, height)
		detail := m.renderDetail(detailW, height)
		sep := theme.Fg(t, t.Border).Render(strings.Repeat("│\n", height-1) + "│")
		return lipgloss.JoinHorizontal(lipgloss.Top, rail, sep, detail)
	}

	// Narrow: single column
	if m.aiVisible {
		return m.renderAIPanel(width, height)
	}
	return m.renderDetail(width, height)
}

func (m *Model) renderRail(width, height int) string {
	t := m.theme
	title := theme.ApplyGradOn("ꕤ  items", t.GradientFrom, t.GradientTo, t.Background, true)

	lines := []string{title, ""}
	for i, item := range m.items {
		glyph := theme.Fg(t, t.Accent).Render(item.glyph())
		label := item.title
		maxLabelW := width - 6
		if maxLabelW < 4 {
			maxLabelW = 4
		}
		if len([]rune(label)) > maxLabelW {
			label = string([]rune(label)[:maxLabelW-1]) + "…"
		}

		var line string
		if i == m.cursor {
			bar := theme.Fg(t, t.Accent).Bold(true).Render("▌")
			txt := theme.Fg(t, t.Primary).Bold(true).Render(fmt.Sprintf(" %s %s", item.glyph(), label))
			line = bar + txt
		} else {
			line = "  " + glyph + " " + theme.Fg(t, t.Foreground).Render(label)
		}
		line = theme.Base(t).Width(width).Render(line)
		lines = append(lines, line)
	}

	if len(m.items) == 0 {
		lines = append(lines,
			"",
			theme.Fg(t, t.Muted).Render("  no items yet"),
			theme.Fg(t, t.Subtle).Render("  press n to"),
			theme.Fg(t, t.Subtle).Render("  capture a spark"),
		)
	}

	hint := theme.Base(t).Width(width).Render(
		theme.Fg(t, t.Subtle).Render("n new · e edit · i AI"),
	)
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	content = theme.Base(t).Width(width).Height(height - 1).Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, content, hint)
}

func (m *Model) renderDetail(width, height int) string {
	t := m.theme

	// Capture prompt (n) — modal-style, takes over the detail pane.
	if m.capturing {
		title := theme.ApplyGradOn("ꕤ  capture a spark", t.GradientFrom, t.GradientTo, t.Background, true)
		m.captureIn.Width = width - 6
		input := theme.Base(t).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocus).
			Width(width - 4).
			Padding(0, 1).
			Render(m.captureIn.View())
		hint := theme.Fg(t, t.Subtle).Italic(true).
			Render("enter  save   ·   esc  cancel")
		block := lipgloss.JoinVertical(lipgloss.Left, title, "", input, "", hint)
		return theme.Place(t, width, height, lipgloss.Center, lipgloss.Center, block)
	}

	// Empty state — nothing selected.
	if m.detailID == "" {
		empty := lipgloss.JoinVertical(lipgloss.Center,
			theme.Fg(t, t.Muted).Bold(true).Render("select an item"),
			"",
			theme.Fg(t, t.Subtle).Render("j/k  navigate   ·   enter  focus AI"),
			theme.Fg(t, t.Subtle).Render("n  new spark   ·   e  edit   ·   g  unstuck"),
		)
		return theme.Place(t, width, height, lipgloss.Center, lipgloss.Center, empty)
	}

	// Loading state — item selected but body not yet loaded.
	if m.detailTitle == "" {
		spinner := theme.Fg(t, t.Muted).Render("  loading…")
		return theme.Base(t).Width(width).Height(height).Render(spinner)
	}

	title := theme.ApplyGradOn(m.detailTitle, t.GradientFrom, t.GradientTo, t.Background, true)
	hintH := 1
	titleH := lipgloss.Height(title)

	// L3: inline editing mode — show textarea.
	if m.editing {
		m.editor.SetWidth(width - 4)
		m.editor.SetHeight(height - titleH - hintH - 2)
		editorView := theme.Base(t).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocus).
			Width(width - 2).
			Render(m.editor.View())
		hint := theme.Base(t).Width(width).Render(
			theme.Fg(t, t.Subtle).Italic(true).Render("ctrl+s  save   ·   esc  cancel"),
		)
		return theme.Base(t).Width(width).Height(height).Render(
			lipgloss.JoinVertical(lipgloss.Left, title, "", editorView, hint),
		)
	}

	// Normal read view — scrollable body with 2-space left indent for breathing room.
	bodyW := width - 4 // narrower wrap so the 2-char indent doesn't overflow
	if bodyW < 10 {
		bodyW = 10
	}
	bodyH := height - titleH - hintH - 2
	if bodyH < 1 {
		bodyH = 1
	}

	wrapped := wordWrap(m.detailBody, bodyW)
	lines := strings.Split(wrapped, "\n")
	maxScroll := len(lines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.detailScroll > maxScroll {
		m.detailScroll = maxScroll
	}
	if m.detailScroll < 0 {
		m.detailScroll = 0
	}
	visible := lines[m.detailScroll:]
	if len(visible) > bodyH {
		visible = visible[:bodyH]
	}
	// Indent each non-empty line by 2 spaces.
	indented := make([]string, len(visible))
	for i, l := range visible {
		if l != "" {
			indented[i] = "  " + l
		} else {
			indented[i] = l
		}
	}
	bodyText := theme.Fg(t, t.Foreground).Render(strings.Join(indented, "\n"))
	bodyBlock := theme.Base(t).Width(width).Height(bodyH).Render(bodyText)

	hint := theme.Base(t).Width(width).Render(
		theme.Fg(t, t.Subtle).Render("e  edit  (ctrl+s saves)  ·  i  AI  ·  a  ask  ·  J/K  scroll"),
	)
	return theme.Base(t).Width(width).Height(height).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", bodyBlock, hint),
	)
}

func (m *Model) renderAIPanel(width, height int) string {
	t := m.theme
	header := theme.ApplyGradOn("ꕤ  AI mentor", t.GradientFrom, t.GradientTo, t.Background, true)

	// Context pill — shows the current project/spark name.
	ctxPill := ""
	if m.aiContext.Title != "" {
		pill := fmt.Sprintf(" %s ", m.aiContext.Title)
		maxPillW := width - 4
		if maxPillW < 4 {
			maxPillW = 4
		}
		if len([]rune(pill)) > maxPillW {
			pill = string([]rune(pill)[:maxPillW-1]) + "… "
		}
		ctxPill = theme.Fg(t, t.Accent).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocus).
			Render(pill)
	}

	// Input bar.
	m.aiInput.Width = width - 4
	inputBar := theme.Base(t).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus).
		Width(width - 2).
		Render(m.aiInput.View())
	inputH := lipgloss.Height(inputBar)

	// Quiz widget (if pending).
	quizWidget := m.renderQuizWidget(width - 2)
	quizH := 0
	if quizWidget != "" {
		quizH = lipgloss.Height(quizWidget) + 1
	}

	// Status line.
	statusLine := ""
	switch m.aiState {
	case aiStateWaiting:
		statusLine = theme.Fg(t, t.Accent).Render("ꕤ  thinking…")
	case aiStateError:
		e := m.aiErrText
		if len([]rune(e)) > width-4 {
			e = string([]rune(e)[:width-7]) + "…"
		}
		statusLine = theme.Fg(t, t.Danger).Render("✗ " + e)
	}

	headerH := lipgloss.Height(header)
	ctxH := 0
	if ctxPill != "" {
		ctxH = lipgloss.Height(ctxPill)
	}
	statusH := 0
	if statusLine != "" {
		statusH = 1
	}
	hintH := 1

	histH := height - headerH - ctxH - inputH - quizH - statusH - hintH - 1
	if histH < 1 {
		histH = 1
	}

	history := m.renderHistory(width, histH)

	parts := []string{header}
	if ctxPill != "" {
		parts = append(parts, ctxPill)
	}
	parts = append(parts, history)
	if quizWidget != "" {
		parts = append(parts, "", quizWidget)
	}
	if statusLine != "" {
		parts = append(parts, statusLine)
	}
	parts = append(parts, inputBar)

	hintText := "enter  send  ·  esc  back  ·  i  hide"
	if m.pendingQuiz != nil {
		hintText = "a-z / ↑↓  choose  ·  enter  confirm  ·  esc  skip"
	}
	hint := theme.Base(t).Width(width).Render(theme.Fg(t, t.Subtle).Render(hintText))
	parts = append(parts, hint)

	return theme.Base(t).Width(width).Height(height).Render(
		lipgloss.JoinVertical(lipgloss.Left, parts...),
	)
}

func (m *Model) renderHistory(width, height int) string {
	t := m.theme
	if len(m.aiMessages) == 0 {
		if m.aiState == aiStateWaiting {
			return theme.Base(t).Width(width).Height(height).Render(
				theme.Fg(t, t.Muted).Render("  ꕤ  thinking…"),
			)
		}
		return theme.Base(t).Width(width).Height(height).Render(
			theme.Fg(t, t.Muted).Render("  (conversation starts here)"),
		)
	}

	var lines []string
	for _, msg := range m.aiMessages {
		var prefix, content string
		switch msg.Role {
		case domain.MessageRoleUser:
			prefix = theme.Fg(t, t.Primary).Bold(true).Render("you  ")
			content = theme.Fg(t, t.Foreground).Render(msg.Content)
		case domain.MessageRoleAssistant:
			prefix = theme.Fg(t, t.Accent).Bold(true).Render("ꕤ    ")
			content = theme.Fg(t, t.Foreground).Render(msg.Content)
		default:
			continue
		}
		msgW := width - 6
		if msgW < 10 {
			msgW = 10
		}
		wrapped := wordWrap(content, msgW)
		msgLines := strings.Split(wrapped, "\n")
		for i, l := range msgLines {
			if i == 0 {
				lines = append(lines, prefix+l)
			} else {
				lines = append(lines, "     "+l)
			}
		}
		lines = append(lines, "")
	}

	// Clamp scroll.
	maxScroll := len(lines) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.aiScroll > maxScroll {
		m.aiScroll = maxScroll
	}
	if m.aiScroll < 0 {
		m.aiScroll = 0
	}

	visible := lines[m.aiScroll:]
	if len(visible) > height {
		visible = visible[:height]
	}

	return theme.Base(t).Width(width).Height(height).Render(
		strings.Join(visible, "\n"),
	)
}

func (m *Model) renderQuizWidget(width int) string {
	if m.pendingQuiz == nil {
		return ""
	}
	quiz := m.pendingQuiz
	t := m.theme

	questionStyle := theme.Fg(t, t.Foreground).Bold(true)
	question := questionStyle.Render(wordWrap(quiz.Question, max(16, width-4)))

	var choiceLines []string
	for i, choice := range quiz.Choices {
		if i == m.quizCursor {
			bullet := theme.Fg(t, t.Primary).Bold(true).Render("▌ " + choice.Key + ")")
			val := theme.Fg(t, t.Primary).Bold(true).Render("  " + choice.Text)
			choiceLines = append(choiceLines, bullet+val)
		} else {
			label := theme.Fg(t, t.Muted).Render("  " + choice.Key + ")")
			val := theme.Fg(t, t.Foreground).Render("  " + choice.Text)
			choiceLines = append(choiceLines, label+val)
		}
	}

	header := theme.Fg(t, t.Accent).Bold(true).Render("quiz")
	body := lipgloss.JoinVertical(lipgloss.Left,
		header, "", question, "", strings.Join(choiceLines, "\n"),
	)
	return theme.Base(t).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Width(width).
		Padding(0, 1).
		Render(body)
}

// wordWrap wraps text to maxWidth runes per line.
func wordWrap(text string, maxWidth int) string {
	if maxWidth < 1 {
		return text
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	var b strings.Builder
	lineLen := 0
	for i, word := range words {
		wl := len([]rune(word))
		if i == 0 {
			b.WriteString(word)
			lineLen = wl
		} else if lineLen+1+wl > maxWidth {
			b.WriteString("\n")
			b.WriteString(word)
			lineLen = wl
		} else {
			b.WriteString(" ")
			b.WriteString(word)
			lineLen += 1 + wl
		}
	}
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
