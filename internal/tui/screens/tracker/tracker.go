package tracker

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
func (m Model) Title() string                             { return "Tracker" }

func (m Model) View(width, height int) string {
	header := theme.ApplyGrad("✦ Tracker", m.theme.GradientFrom, m.theme.GradientTo, true)
	body := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		lipgloss.NewStyle().Foreground(m.theme.Foreground).Render("No activity yet."),
		"",
		lipgloss.NewStyle().Foreground(m.theme.Subtle).Italic(true).
			Render("Coming in M4: consistency chart, streak, weekly velocity."),
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, body)
}
