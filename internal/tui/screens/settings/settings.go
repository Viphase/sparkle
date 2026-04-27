package settings

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/config"
	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/theme"
	"github.com/viphase/sparkle/internal/workspace"
)

type Model struct {
	theme  theme.Theme
	config config.Config
	ws     workspace.Workspace
}

func New(t theme.Theme, ws workspace.Workspace) screens.Screen {
	return Model{theme: t, ws: ws, config: config.Defaults()}
}

func (m Model) Init() tea.Cmd                             { return nil }
func (m Model) Update(_ tea.Msg) (screens.Screen, tea.Cmd) { return m, nil }
func (m Model) Title() string                             { return "Settings" }

func (m Model) View(width, height int) string {
	header := theme.ApplyGrad("✦ Settings", m.theme.GradientFrom, m.theme.GradientTo, true)

	root := m.ws.Root
	if root == "" {
		root = "(none selected)"
	}

	rowStyle := lipgloss.NewStyle().Foreground(m.theme.Foreground)
	keyStyle := lipgloss.NewStyle().Foreground(m.theme.Muted).Width(14)

	rows := []string{
		rowStyle.Render(keyStyle.Render("theme") + m.theme.Name),
		rowStyle.Render(keyStyle.Render("workspace") + root),
		rowStyle.Render(keyStyle.Render("words threshold") + intStr(m.config.WordsThreshold)),
	}

	hint := lipgloss.NewStyle().Foreground(m.theme.Subtle).Italic(true).
		Render("Coming next: theme picker (pastel-dark · pastel-light · nova), workspace switching, TOML config.")

	body := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		lipgloss.JoinVertical(lipgloss.Left, rows...),
		"",
		hint,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, body)
}

func intStr(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
