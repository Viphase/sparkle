package ai

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/theme"
)

type Model struct{ theme theme.Theme }

func New(t theme.Theme) screens.Screen                     { return Model{theme: t} }
func (m Model) Init() tea.Cmd                              { return nil }
func (m Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	if tc, ok := msg.(msgs.ThemeChangedMsg); ok {
		m.theme = theme.ByName(tc.ThemeName)
	}
	return m, nil
}
func (m Model) Title() string                              { return "AI" }

func (m Model) View(width, height int) string {
	header := theme.ApplyGradOn("AI Guide", m.theme.GradientFrom, m.theme.GradientTo, m.theme.Background, true)
	body := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		theme.Fg(m.theme, m.theme.Foreground).Render("Provider: not configured."),
		"",
		theme.Fg(m.theme, m.theme.Subtle).Italic(true).
			Render("Coming in M5: chat with mock provider · M6: real provider behind the same interface."),
	)
	return theme.Place(m.theme, width, height, lipgloss.Center, lipgloss.Center, body)
}
