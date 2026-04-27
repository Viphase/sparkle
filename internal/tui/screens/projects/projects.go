package projects

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
func (m Model) Title() string                             { return "Projects" }

func (m Model) View(width, height int) string {
	header := theme.ApplyGrad("✦ Projects", m.theme.GradientFrom, m.theme.GradientTo, true)
	body := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		lipgloss.NewStyle().Foreground(m.theme.Foreground).Render("No projects yet."),
		"",
		lipgloss.NewStyle().Foreground(m.theme.Subtle).Italic(true).
			Render("Coming in M3: project list, editable fields, two-pane layout."),
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, body)
}
