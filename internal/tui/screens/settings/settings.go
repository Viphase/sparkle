package settings

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
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/theme"
	"github.com/viphase/sparkle/internal/workspace"
)

// Row indices within the flat navigation list.
const (
	rowAPIKey   = 0
	rowModel    = 1
	rowTestConn = 2
	rowTheme    = 3
	rowWords    = 4
	rowSkill    = 5
	rowMouse    = 6
	numRows     = 7
)

// Available Anthropic models in display order.
var availableModels = []string{
	"claude-haiku-4-5",
	"claude-sonnet-4-6",
	"claude-opus-4-7",
}

type pingState int

const (
	pingIdle    pingState = iota
	pingTesting           // request in-flight
	pingOK                // last test succeeded
	pingFailed            // last test failed
)

type pingDoneMsg struct{ err error }

type Model struct {
	theme    theme.Theme
	config   config.Config
	ws       workspace.Workspace
	cursor   int
	themeIdx int
	skillIdx int
	modelIdx int

	// Filesystem-backed skill definitions. When non-empty these replace the
	// hardcoded domain.AllSkills() fallback in the skill picker.
	skillDefs []domain.SkillDef

	// API key editing
	keyInput   textinput.Model
	editingKey bool

	// Test connection state
	ping    pingState
	pingMsg string

	lastH int
}

func New(t theme.Theme, ws workspace.Workspace, cfg config.Config) screens.Screen {
	themeIdx := 0
	for i, p := range theme.AllPalettes() {
		if p.Name == cfg.Theme {
			themeIdx = i
			break
		}
	}
	modelIdx := 0
	for i, m := range availableModels {
		if m == cfg.AIModel {
			modelIdx = i
			break
		}
	}

	ti := textinput.New()
	ti.Placeholder = "sk-ant-..."
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.CharLimit = 200
	ti.Width = 40

	m := &Model{
		theme:    t,
		config:   cfg,
		ws:       ws,
		themeIdx: themeIdx,
		modelIdx: modelIdx,
		keyInput: ti,
	}
	// Set initial skillIdx based on config.
	m.skillIdx = m.findSkillIdx(cfg.ActiveSkill)
	return m
}

func (m *Model) Init() tea.Cmd { return nil }
func (m *Model) Title() string { return "Settings" }

func (m *Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.ThemeChangedMsg:
		m.theme = theme.ByName(msg.ThemeName)
		return m, nil
	case msgs.SkillDefsLoadedMsg:
		m.skillDefs = msg.Skills
		// Re-sync skillIdx to the current config value after defs load.
		m.skillIdx = m.findSkillIdx(m.config.ActiveSkill)
		return m, nil
	case pingDoneMsg:
		if msg.err != nil {
			m.ping = pingFailed
			m.pingMsg = msg.err.Error()
		} else {
			m.ping = pingOK
			m.pingMsg = "connection OK"
		}
		return m, nil
	case tea.KeyMsg:
		if m.editingKey {
			return m.handleKeyInput(msg)
		}
		return m.handleKey(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	// Forward to textinput when editing (handles blink, etc.)
	if m.editingKey {
		var cmd tea.Cmd
		m.keyInput, cmd = m.keyInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleKeyInput(msg tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.editingKey = false
		m.keyInput.Blur()
		m.config.AnthropicAPIKey = m.keyInput.Value()
		m.ping = pingIdle
		m.pingMsg = ""
		return m, tea.Batch(m.saveCmd(), apiKeyCmd(m.config.AnthropicAPIKey, m.config.AIModel))
	case "esc":
		m.editingKey = false
		m.keyInput.Blur()
		if m.config.AnthropicAPIKey != "" {
			m.keyInput.SetValue(m.config.AnthropicAPIKey)
		} else {
			m.keyInput.SetValue("")
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.keyInput, cmd = m.keyInput.Update(msg)
	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) (screens.Screen, tea.Cmd) {
	palettes := theme.AllPalettes()
	skills := m.effectiveSkills()
	switch msg.String() {
	case "j", "down":
		if m.cursor < numRows-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		switch m.cursor {
		case rowAPIKey:
			m.editingKey = true
			m.keyInput.SetValue(m.config.AnthropicAPIKey)
			m.keyInput.Focus()
			return m, textinput.Blink
		case rowTestConn:
			m.ping = pingTesting
			m.pingMsg = ""
			return m, m.pingCmd()
		}
	case "left", "h":
		switch m.cursor {
		case rowTheme:
			m.themeIdx = (m.themeIdx - 1 + len(palettes)) % len(palettes)
			m.config.Theme = palettes[m.themeIdx].Name
			return m, tea.Batch(m.saveCmd(), themeCmd(m.config.Theme))
		case rowWords:
			if m.config.WordsThreshold > 1 {
				m.config.WordsThreshold--
				return m, m.saveCmd()
			}
		case rowMouse:
			m.config.MouseEnabled = !m.config.MouseEnabled
			return m, tea.Batch(m.saveCmd(), mouseCmd(m.config.MouseEnabled))
		case rowSkill:
			m.skillIdx = (m.skillIdx - 1 + len(skills)) % len(skills)
			m.config.ActiveSkill = skills[m.skillIdx].Slug
			return m, tea.Batch(m.saveCmd(), skillCmd(m.config.ActiveSkill))
		case rowModel:
			m.modelIdx = (m.modelIdx - 1 + len(availableModels)) % len(availableModels)
			m.config.AIModel = availableModels[m.modelIdx]
			return m, tea.Batch(m.saveCmd(), apiKeyCmd(m.config.AnthropicAPIKey, m.config.AIModel))
		}
	case "right", "l":
		switch m.cursor {
		case rowTheme:
			m.themeIdx = (m.themeIdx + 1) % len(palettes)
			m.config.Theme = palettes[m.themeIdx].Name
			return m, tea.Batch(m.saveCmd(), themeCmd(m.config.Theme))
		case rowWords:
			m.config.WordsThreshold++
			return m, m.saveCmd()
		case rowMouse:
			m.config.MouseEnabled = !m.config.MouseEnabled
			return m, tea.Batch(m.saveCmd(), mouseCmd(m.config.MouseEnabled))
		case rowSkill:
			m.skillIdx = (m.skillIdx + 1) % len(skills)
			m.config.ActiveSkill = skills[m.skillIdx].Slug
			return m, tea.Batch(m.saveCmd(), skillCmd(m.config.ActiveSkill))
		case rowModel:
			m.modelIdx = (m.modelIdx + 1) % len(availableModels)
			m.config.AIModel = availableModels[m.modelIdx]
			return m, tea.Batch(m.saveCmd(), apiKeyCmd(m.config.AnthropicAPIKey, m.config.AIModel))
		}
	}
	return m, nil
}

func (m *Model) handleMouse(msg tea.MouseMsg) (screens.Screen, tea.Cmd) {
	switch msg.Type {
	case tea.MouseWheelDown:
		if m.cursor < numRows-1 {
			m.cursor++
		}
	case tea.MouseWheelUp:
		if m.cursor > 0 {
			m.cursor--
		}
	}
	return m, nil
}

// pingCmd launches a background test-connection check using the current key.
func (m *Model) pingCmd() tea.Cmd {
	key := m.config.ResolvedAPIKey()
	model := m.config.AIModel
	return func() tea.Msg {
		if key == "" {
			return pingDoneMsg{err: fmt.Errorf("no API key — set one above first")}
		}
		provider := guideai.NewAnthropicProvider(key, model)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return pingDoneMsg{err: provider.Ping(ctx)}
	}
}

func (m *Model) saveCmd() tea.Cmd {
	root := m.ws.Root
	cfg := m.config
	return func() tea.Msg {
		if root == "" {
			return msgs.StatusMsg{Text: "no workspace — config not saved"}
		}
		if err := config.Save(root, cfg); err != nil {
			return msgs.ErrorMsg{Source: "settings", Err: err}
		}
		return msgs.StatusMsg{Text: "saved"}
	}
}

func themeCmd(name string) tea.Cmd {
	return func() tea.Msg { return msgs.ThemeChangedMsg{ThemeName: name} }
}

func mouseCmd(enabled bool) tea.Cmd {
	return func() tea.Msg { return msgs.MouseToggledMsg{Enabled: enabled} }
}

func skillCmd(slug string) tea.Cmd {
	return func() tea.Msg { return msgs.SkillChangedMsg{Skill: slug} }
}

func apiKeyCmd(key, model string) tea.Cmd {
	return func() tea.Msg { return msgs.APIKeyChangedMsg{Key: key, Model: model} }
}

// effectiveSkills returns the skill list for the picker. Prefers filesystem-
// loaded definitions but falls back to the hardcoded domain constants when
// no .sparkle/skills/ files have been loaded yet.
func (m *Model) effectiveSkills() []domain.SkillDef {
	none := domain.SkillDef{
		Slug:        "",
		Label:       "none",
		Description: "generic project guidance, no specialisation",
	}
	if len(m.skillDefs) > 0 {
		return append([]domain.SkillDef{none}, m.skillDefs...)
	}
	// Fallback: synthesise SkillDef values from the hardcoded Skill constants.
	builtins := domain.AllSkills()
	defs := make([]domain.SkillDef, 0, len(builtins))
	for _, s := range builtins {
		defs = append(defs, domain.SkillDef{
			Slug:        string(s),
			Label:       s.Label(),
			Description: s.Description(),
		})
	}
	return defs
}

// findSkillIdx returns the index of slug in effectiveSkills(), defaulting to 0.
func (m *Model) findSkillIdx(slug string) int {
	for i, s := range m.effectiveSkills() {
		if s.Slug == slug {
			return i
		}
	}
	return 0
}

func (m *Model) View(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	m.lastH = height

	t := m.theme
	header := theme.ApplyGradOn("ꕤ  Settings", t.GradientFrom, t.GradientTo, t.Background, true)

	// Section: AI Provider
	aiHeader := sectionHeader(t, "AI Provider")
	keyDisplay := m.keyDisplay()
	apiKeyRow := m.renderRow(rowAPIKey, "api key", keyDisplay,
		"paste your Anthropic key (sk-ant-…)")
	if m.editingKey {
		apiKeyRow = m.renderEditRow("api key", m.keyInput.View())
	}
	modelRow := m.renderRow(rowModel, "model", availableModels[m.modelIdx],
		"Haiku is fastest, Opus is most capable")
	testRow := m.renderTestRow()

	// Section: Appearance
	appearHeader := sectionHeader(t, "Appearance")
	palettes := theme.AllPalettes()
	themeRow := m.renderRow(rowTheme, "theme", palettes[m.themeIdx].Name,
		"visual colour palette for the whole app")

	// Section: Tracking
	trackHeader := sectionHeader(t, "Tracking")
	wordsRow := m.renderRow(rowWords, "words threshold",
		fmt.Sprintf("%d words", m.config.WordsThreshold),
		"minimum new words to count a writing session")

	// Section: Input
	inputHeader := sectionHeader(t, "Input")
	skills := m.effectiveSkills()
	var skillLabel, skillDesc string
	if m.skillIdx < len(skills) {
		skillLabel = skills[m.skillIdx].Label
		skillDesc = skills[m.skillIdx].Description
	} else {
		skillLabel = "none"
		skillDesc = "generic project guidance"
	}
	skillRow := m.renderRow(rowSkill, "ai skill", skillLabel, skillDesc)
	mouseVal := "on"
	if !m.config.MouseEnabled {
		mouseVal = "off"
	}
	mouseRow := m.renderRow(rowMouse, "mouse", mouseVal,
		"enable mouse clicks and scroll wheel")

	wsLine := theme.Fg(t, t.Subtle).Render("workspace  " + wsRoot(m.ws))
	hint := theme.Fg(t, t.Subtle).Italic(true).
		Render("↑↓ / jk  navigate   ←→  change   enter  edit/test")

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		aiHeader,
		apiKeyRow,
		modelRow,
		testRow,
		"",
		appearHeader,
		themeRow,
		"",
		trackHeader,
		wordsRow,
		"",
		inputHeader,
		skillRow,
		mouseRow,
		"",
		wsLine,
		"",
		hint,
	)

	return theme.Place(t, width, height, lipgloss.Left, lipgloss.Top, body)
}

func sectionHeader(t theme.Theme, title string) string {
	return theme.Fg(t, t.Accent).Bold(true).Render("── " + strings.ToUpper(title))
}

func (m *Model) keyDisplay() string {
	if m.config.AnthropicAPIKey != "" {
		k := m.config.AnthropicAPIKey
		if len(k) > 8 {
			k = k[:4] + strings.Repeat("•", len(k)-8) + k[len(k)-4:]
		}
		return k
	}
	return "not set"
}

func wsRoot(ws workspace.Workspace) string {
	if ws.Root == "" {
		return "(none)"
	}
	return ws.Root
}

func (m *Model) renderRow(row int, key, value, desc string) string {
	const keyWidth = 18
	const descPad = 22 // key + gap before desc starts
	selected := m.cursor == row

	t := m.theme
	keyStyle := theme.Fg(t, t.Muted).Width(keyWidth)
	valStyle := theme.Fg(t, t.Foreground)
	arrowStyle := theme.Fg(t, t.Subtle)
	descStyle := theme.Fg(t, t.Subtle).Italic(true)
	cur := "  "

	if selected {
		cur = theme.Fg(t, t.Accent).Bold(true).Render("▌ ")
		keyStyle = keyStyle.Foreground(t.Primary)
		valStyle = valStyle.Foreground(t.Primary).Bold(true)
		arrowStyle = theme.Fg(t, t.Primary)
		descStyle = theme.Fg(t, t.Muted).Italic(true)
	}

	// Rows that cycle with arrows.
	cycleRows := map[int]bool{rowTheme: true, rowModel: true, rowSkill: true, rowMouse: true}
	var mainLine string
	if cycleRows[row] {
		arrows := arrowStyle.Render(" ← ") + valStyle.Render(value) + arrowStyle.Render(" →")
		mainLine = cur + keyStyle.Render(key) + arrows
	} else {
		mainLine = cur + keyStyle.Render(key) + valStyle.Render(value)
	}
	if desc == "" || !selected {
		return mainLine
	}
	// Show description on next line when selected.
	descLine := strings.Repeat(" ", descPad) + descStyle.Render(desc)
	return lipgloss.JoinVertical(lipgloss.Left, mainLine, descLine)
}

func (m *Model) renderEditRow(key, inputView string) string {
	const keyWidth = 18
	t := m.theme
	cur := theme.Fg(t, t.Accent).Bold(true).Render("▌ ")
	keyStyle := theme.Fg(t, t.Primary).Width(keyWidth)
	return cur + keyStyle.Render(key) + inputView
}

func (m *Model) renderTestRow() string {
	const keyWidth = 18
	selected := m.cursor == rowTestConn
	t := m.theme

	keyStyle := theme.Fg(t, t.Muted).Width(keyWidth)
	cur := "  "
	if selected {
		cur = theme.Fg(t, t.Accent).Bold(true).Render("▌ ")
		keyStyle = keyStyle.Foreground(t.Primary)
	}

	var status string
	switch m.ping {
	case pingIdle:
		status = theme.Fg(t, t.Subtle).Render("[enter to test]")
	case pingTesting:
		status = theme.Fg(t, t.Accent).Render("testing…")
	case pingOK:
		status = theme.Fg(t, t.Success).Render("✓ " + m.pingMsg)
	case pingFailed:
		errMsg := m.pingMsg
		if len(errMsg) > 60 {
			errMsg = errMsg[:57] + "…"
		}
		status = theme.Fg(t, t.Danger).Render("✗ " + errMsg)
	}

	return cur + keyStyle.Render("test connection") + status
}
