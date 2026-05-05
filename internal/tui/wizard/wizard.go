// Package wizard runs the first-run setup flow as a separate tea.Program.
//
// It collects: workspace path, theme, Anthropic API key (with optional Test
// connection), default skill, and an optional first spark. The result is
// returned through Model.Result(); callers persist the config and (if any)
// the first spark themselves.
package wizard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	guideai "github.com/viphase/sparkle/internal/ai"
	"github.com/viphase/sparkle/internal/config"
	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/theme"
)

type step int

const (
	stepWorkspace step = iota
	stepTheme
	stepAPIKey
	stepSkill
	stepFirstSpark
	stepDone
)

// Result is what the caller reads after Run returns.
type Result struct {
	// Cancelled is true if the user pressed ctrl+c before finishing.
	Cancelled bool
	// WorkspacePath is the chosen workspace root (absolute or ~ form).
	WorkspacePath string
	// Config holds the chosen theme, API key, model, and active skill.
	Config config.Config
	// FirstSparkTitle is the title for the optional first spark; empty means skip.
	FirstSparkTitle string
}

type pingDoneMsg struct{ err error }

// Model is the wizard's bubbletea model.
type Model struct {
	theme theme.Theme
	step  step
	width int
	height int

	// step inputs
	wsInput    textinput.Model
	keyInput   textinput.Model
	sparkInput textinput.Model

	themeIdx int
	skillIdx int

	skills []domain.SkillDef

	// API-key test state
	pingState   string // "" | "testing" | "ok" | "failed"
	pingMessage string

	cancelled bool
	result    Result
}

// New constructs a wizard with sensible defaults.
//
// defaultWS is the proposed workspace path (typically workspace.DefaultPath()).
// skills is the available skill list (built-in + filesystem); pass nil to use
// the built-in fallback from domain.AllSkills().
func New(defaultWS string, skills []domain.SkillDef) *Model {
	t := theme.ByName(config.Defaults().Theme)

	wsTI := textinput.New()
	wsTI.Placeholder = defaultWS
	wsTI.SetValue(defaultWS)
	wsTI.Width = 50
	wsTI.Focus()

	keyTI := textinput.New()
	keyTI.Placeholder = "sk-ant-…  (paste your key, or skip)"
	keyTI.EchoMode = textinput.EchoPassword
	keyTI.EchoCharacter = '•'
	keyTI.CharLimit = 200
	keyTI.Width = 50

	sparkTI := textinput.New()
	sparkTI.Placeholder = "your first idea (or skip)"
	sparkTI.CharLimit = 120
	sparkTI.Width = 50

	if skills == nil {
		builtins := domain.AllSkills()
		skills = make([]domain.SkillDef, 0, len(builtins)+1)
		skills = append(skills, domain.SkillDef{Slug: "", Label: "none", Description: "generic project guidance"})
		for _, s := range builtins {
			skills = append(skills, domain.SkillDef{Slug: string(s), Label: s.Label(), Description: s.Description()})
		}
	}

	m := &Model{
		theme:      t,
		wsInput:    wsTI,
		keyInput:   keyTI,
		sparkInput: sparkTI,
		skills:     skills,
		result:     Result{Config: config.Defaults(), WorkspacePath: defaultWS},
	}
	return m
}

// Init satisfies tea.Model.
func (m *Model) Init() tea.Cmd { return textinput.Blink }

// Result returns the wizard's collected output. Safe to call after Run exits.
func (m *Model) Result() Result {
	if m.cancelled {
		return Result{Cancelled: true}
	}
	r := m.result
	r.WorkspacePath = strings.TrimSpace(m.wsInput.Value())
	if r.WorkspacePath == "" {
		r.WorkspacePath = m.wsInput.Placeholder
	}
	palettes := theme.AllPalettes()
	if m.themeIdx >= 0 && m.themeIdx < len(palettes) {
		r.Config.Theme = palettes[m.themeIdx].Name
	}
	r.Config.AnthropicAPIKey = strings.TrimSpace(m.keyInput.Value())
	if m.skillIdx >= 0 && m.skillIdx < len(m.skills) {
		r.Config.ActiveSkill = m.skills[m.skillIdx].Slug
	}
	r.FirstSparkTitle = strings.TrimSpace(m.sparkInput.Value())
	return r
}

// Update satisfies tea.Model. The wizard handles ctrl+c (cancel) and esc
// (back-step) globally; per-step keys advance.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case pingDoneMsg:
		if msg.err != nil {
			m.pingState = "failed"
			m.pingMessage = msg.err.Error()
		} else {
			m.pingState = "ok"
			m.pingMessage = "connection OK"
		}
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "esc":
			if m.step > 0 {
				m.step--
				m.refocus()
			}
			return m, nil
		}
		return m.handleStepKey(msg)
	}
	// Forward to active textinput for blink/paste handling.
	return m.forwardToActiveInput(msg)
}

func (m *Model) handleStepKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.step {
	case stepWorkspace:
		if msg.String() == "enter" {
			return m.advance()
		}
		return m.forwardToActiveInput(msg)
	case stepTheme:
		palettes := theme.AllPalettes()
		switch msg.String() {
		case "left", "h":
			m.themeIdx = (m.themeIdx - 1 + len(palettes)) % len(palettes)
			m.theme = palettes[m.themeIdx]
		case "right", "l":
			m.themeIdx = (m.themeIdx + 1) % len(palettes)
			m.theme = palettes[m.themeIdx]
		case "enter":
			return m.advance()
		}
		return m, nil
	case stepAPIKey:
		switch msg.String() {
		case "enter":
			return m.advance()
		case "ctrl+t":
			// Test connection
			key := strings.TrimSpace(m.keyInput.Value())
			if key == "" {
				m.pingState = "failed"
				m.pingMessage = "no key entered"
				return m, nil
			}
			m.pingState = "testing"
			m.pingMessage = ""
			model := config.Defaults().AIModel
			return m, func() tea.Msg {
				p := guideai.NewAnthropicProvider(key, model)
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				return pingDoneMsg{err: p.Ping(ctx)}
			}
		case "ctrl+s":
			// Skip key
			m.keyInput.SetValue("")
			return m.advance()
		}
		return m.forwardToActiveInput(msg)
	case stepSkill:
		switch msg.String() {
		case "left", "h":
			m.skillIdx = (m.skillIdx - 1 + len(m.skills)) % len(m.skills)
		case "right", "l":
			m.skillIdx = (m.skillIdx + 1) % len(m.skills)
		case "enter":
			return m.advance()
		}
		return m, nil
	case stepFirstSpark:
		switch msg.String() {
		case "enter":
			return m.advance()
		case "ctrl+s":
			m.sparkInput.SetValue("")
			return m.advance()
		}
		return m.forwardToActiveInput(msg)
	}
	return m, nil
}

// advance moves to the next step or finishes the wizard.
func (m *Model) advance() (tea.Model, tea.Cmd) {
	m.step++
	if m.step >= stepDone {
		return m, tea.Quit
	}
	m.refocus()
	return m, textinput.Blink
}

// refocus blurs all inputs then focuses the one for the current step.
func (m *Model) refocus() {
	m.wsInput.Blur()
	m.keyInput.Blur()
	m.sparkInput.Blur()
	switch m.step {
	case stepWorkspace:
		m.wsInput.Focus()
	case stepAPIKey:
		m.keyInput.Focus()
	case stepFirstSpark:
		m.sparkInput.Focus()
	}
}

// forwardToActiveInput proxies messages to the textinput for the active step.
func (m *Model) forwardToActiveInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.step {
	case stepWorkspace:
		m.wsInput, cmd = m.wsInput.Update(msg)
	case stepAPIKey:
		m.keyInput, cmd = m.keyInput.Update(msg)
	case stepFirstSpark:
		m.sparkInput, cmd = m.sparkInput.Update(msg)
	}
	return m, cmd
}

// View satisfies tea.Model.
func (m *Model) View() string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	t := m.theme

	header := theme.ApplyGradOn("ꕤ  Welcome to Sparkle", t.GradientFrom, t.GradientTo, t.Background, true)
	stepLabel := fmt.Sprintf("step %d of 5", int(m.step)+1)
	stepBar := theme.Fg(t, t.Subtle).Render(stepLabel)

	var body string
	switch m.step {
	case stepWorkspace:
		body = m.viewWorkspace()
	case stepTheme:
		body = m.viewTheme()
	case stepAPIKey:
		body = m.viewAPIKey()
	case stepSkill:
		body = m.viewSkill()
	case stepFirstSpark:
		body = m.viewFirstSpark()
	}

	hint := theme.Fg(t, t.Subtle).Italic(true).Render("enter  next   ·   esc  back   ·   ctrl+c  cancel")
	full := lipgloss.JoinVertical(lipgloss.Left, header, stepBar, "", body, "", hint)
	return theme.Place(t, w, h, lipgloss.Center, lipgloss.Center, full)
}

func (m *Model) viewWorkspace() string {
	t := m.theme
	q := theme.Fg(t, t.Foreground).Bold(true).Render("Where should I keep your work?")
	desc := theme.Fg(t, t.Subtle).Render("Sparkle stores sparks, projects, and config in this directory.")
	return lipgloss.JoinVertical(lipgloss.Left, q, desc, "", m.wsInput.View())
}

func (m *Model) viewTheme() string {
	t := m.theme
	q := theme.Fg(t, t.Foreground).Bold(true).Render("Pick a theme.")
	palettes := theme.AllPalettes()
	name := palettes[m.themeIdx].Name
	swatch := theme.Fg(t, t.Accent).Render("← " + name + " →")
	desc := theme.Fg(t, t.Subtle).Render("press ← / → to cycle. live preview applied.")
	return lipgloss.JoinVertical(lipgloss.Left, q, "", swatch, "", desc)
}

func (m *Model) viewAPIKey() string {
	t := m.theme
	q := theme.Fg(t, t.Foreground).Bold(true).Render("Anthropic API key (optional)")
	desc := theme.Fg(t, t.Subtle).Render("paste your key for real AI, or ctrl+s to skip and use the local mock.")
	hint := theme.Fg(t, t.Subtle).Italic(true).Render("ctrl+t  test connection   ·   ctrl+s  skip   ·   enter  continue")
	var status string
	switch m.pingState {
	case "testing":
		status = theme.Fg(t, t.Accent).Render("testing…")
	case "ok":
		status = theme.Fg(t, t.Success).Render("✓ " + m.pingMessage)
	case "failed":
		msg := m.pingMessage
		if len(msg) > 60 {
			msg = msg[:57] + "…"
		}
		status = theme.Fg(t, t.Danger).Render("✗ " + msg)
	}
	return lipgloss.JoinVertical(lipgloss.Left, q, desc, "", m.keyInput.View(), "", status, hint)
}

func (m *Model) viewSkill() string {
	t := m.theme
	q := theme.Fg(t, t.Foreground).Bold(true).Render("Default skill?")
	cur := m.skills[m.skillIdx]
	value := theme.Fg(t, t.Accent).Render("← " + cur.Label + " →")
	desc := theme.Fg(t, t.Subtle).Render(cur.Description)
	return lipgloss.JoinVertical(lipgloss.Left, q, "", value, desc)
}

func (m *Model) viewFirstSpark() string {
	t := m.theme
	q := theme.Fg(t, t.Foreground).Bold(true).Render("Capture your first spark.")
	desc := theme.Fg(t, t.Subtle).Render("type a one-line idea, or ctrl+s to skip.")
	return lipgloss.JoinVertical(lipgloss.Left, q, desc, "", m.sparkInput.View())
}
