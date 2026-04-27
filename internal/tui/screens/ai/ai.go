package ai

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/theme"
)

type Model struct{ theme theme.Theme }

func New(t theme.Theme) screens.Screen                    { return Model{theme: t} }
func (m Model) Init() tea.Cmd                             { return nil }
func (m Model) Update(_ tea.Msg) (screens.Screen, tea.Cmd) { return m, nil }
func (m Model) Title() string                             { return "AI" }

func (m Model) View(width, height int) string {
	header := theme.ApplyGrad("✦ AI Guide", m.theme.GradientFrom, m.theme.GradientTo, true)
	body := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		lipgloss.NewStyle().Foreground(m.theme.Foreground).Render("Provider: not configured."),
		"",
		lipgloss.NewStyle().Foreground(m.theme.Subtle).Italic(true).
			Render("Coming in M5: chat with mock provider · M6: real provider behind the same interface."),
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, body)
}
