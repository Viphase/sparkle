package settings

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/config"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/theme"
	"github.com/viphase/sparkle/internal/workspace"
)

const (
	rowTheme  = 0
	rowWords  = 1
	rowMouse  = 2
	numRows   = 3
)

type Model struct {
	theme    theme.Theme
	config   config.Config
	ws       workspace.Workspace
	cursor   int
	themeIdx int
}

func New(t theme.Theme, ws workspace.Workspace, cfg config.Config) screens.Screen {
	idx := 0
	for i, p := range theme.AllPalettes() {
		if p.Name == cfg.Theme {
			idx = i
			break
		}
	}
	return &Model{
		theme:    t,
		config:   cfg,
		ws:       ws,
		themeIdx: idx,
	}
}

func (m *Model) Init() tea.Cmd { return nil }
func (m *Model) Title() string { return "Settings" }

func (m *Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.ThemeChangedMsg:
		m.theme = theme.ByName(msg.ThemeName)
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
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

func (m *Model) handleKey(msg tea.KeyMsg) (screens.Screen, tea.Cmd) {
	palettes := theme.AllPalettes()
	switch msg.String() {
	case "j", "down":
		if m.cursor < numRows-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "left", "h":
		switch m.cursor {
		case rowTheme:
			m.themeIdx = (m.themeIdx - 1 + len(palettes)) % len(palettes)
			newTheme := palettes[m.themeIdx]
			m.config.Theme = newTheme.Name
			return m, tea.Batch(m.saveCmd(), themeCmd(newTheme.Name))
		case rowWords:
			if m.config.WordsThreshold > 1 {
				m.config.WordsThreshold--
				return m, m.saveCmd()
			}
		case rowMouse:
			m.config.MouseEnabled = !m.config.MouseEnabled
			return m, tea.Batch(m.saveCmd(), mouseCmd(m.config.MouseEnabled))
		}
	case "right", "l":
		switch m.cursor {
		case rowTheme:
			m.themeIdx = (m.themeIdx + 1) % len(palettes)
			newTheme := palettes[m.themeIdx]
			m.config.Theme = newTheme.Name
			return m, tea.Batch(m.saveCmd(), themeCmd(newTheme.Name))
		case rowWords:
			m.config.WordsThreshold++
			return m, m.saveCmd()
		case rowMouse:
			m.config.MouseEnabled = !m.config.MouseEnabled
			return m, tea.Batch(m.saveCmd(), mouseCmd(m.config.MouseEnabled))
		}
	}
	return m, nil
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
		return msgs.StatusMsg{Text: "saved · .sparkle/config.toml"}
	}
}

func themeCmd(name string) tea.Cmd {
	return func() tea.Msg { return msgs.ThemeChangedMsg{ThemeName: name} }
}

func mouseCmd(enabled bool) tea.Cmd {
	return func() tea.Msg { return msgs.MouseToggledMsg{Enabled: enabled} }
}

func (m *Model) View(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	header := theme.ApplyGradOn("Settings", m.theme.GradientFrom, m.theme.GradientTo, m.theme.Background, true)

	palettes := theme.AllPalettes()
	themeNames := make([]string, len(palettes))
	for i, p := range palettes {
		themeNames[i] = p.Name
	}
	themeVal := themeNames[m.themeIdx]
	wordsVal := fmt.Sprintf("%d words", m.config.WordsThreshold)
	mouseVal := "on"
	if !m.config.MouseEnabled {
		mouseVal = "off"
	}

	root := m.ws.Root
	if root == "" {
		root = "(none)"
	}
	wsLine := theme.Fg(m.theme, m.theme.Subtle).Render("workspace  " + root)
	hint := theme.Fg(m.theme, m.theme.Subtle).Italic(true).
		Render("↑↓ / jk  navigate   ←→  change   saves automatically")

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.renderRow(rowTheme, "theme", themeVal),
		m.renderRow(rowWords, "words threshold", wordsVal),
		m.renderRow(rowMouse, "mouse support", mouseVal),
		"",
		wsLine,
		"",
		hint,
	)

	return theme.Place(m.theme, width, height, lipgloss.Center, lipgloss.Center, body)
}

func (m *Model) renderRow(row int, key, value string) string {
	const keyWidth = 18
	selected := m.cursor == row

	keyStyle := theme.Fg(m.theme, m.theme.Muted).Width(keyWidth)
	valStyle := theme.Fg(m.theme, m.theme.Foreground)
	arrowStyle := theme.Fg(m.theme, m.theme.Subtle)
	cur := "  "

	if selected {
		cur = theme.Fg(m.theme, m.theme.Accent).Bold(true).Render("▌ ")
		keyStyle = keyStyle.Foreground(m.theme.Primary)
		valStyle = valStyle.Foreground(m.theme.Primary).Bold(true)
		arrowStyle = theme.Fg(m.theme, m.theme.Primary)
	}

	arrows := arrowStyle.Render(" ← ") + valStyle.Render(value) + arrowStyle.Render(" →")
	return cur + keyStyle.Render(key) + arrows
}
