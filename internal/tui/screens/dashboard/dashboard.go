package dashboard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/components/logo"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/theme"
)

type Model struct {
	theme        theme.Theme
	sparkCount   int
	activeCount  int
	archivedSeen int
}

func New(t theme.Theme) screens.Screen { return Model{theme: t} }

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Title() string { return "Dashboard" }

func (m Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	if loaded, ok := msg.(msgs.SparksLoadedMsg); ok {
		m.sparkCount = 0
		m.activeCount = 0
		m.archivedSeen = 0
		for _, s := range loaded.Items {
			m.sparkCount++
			switch s.Status {
			case domain.SparkStatusArchived:
				m.archivedSeen++
			case domain.SparkStatusNew, domain.SparkStatusQuestioning:
				m.activeCount++
			}
		}
	}
	return m, nil
}

func (m Model) View(width, height int) string {
	logoBlock := logo.Render(m.theme, min(width-8, 64))

	stat := func(label string, n int, color lipgloss.Color) string {
		num := lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%d", n))
		lbl := lipgloss.NewStyle().Foreground(m.theme.Muted).Render(label)
		return lipgloss.JoinVertical(lipgloss.Center, num, lbl)
	}

	stats := lipgloss.JoinHorizontal(lipgloss.Top,
		stat("sparks", m.sparkCount, m.theme.Primary),
		spacer(6),
		stat("active", m.activeCount, m.theme.Accent),
		spacer(6),
		stat("archived", m.archivedSeen, m.theme.Subtle),
	)

	hint := lipgloss.NewStyle().
		Foreground(m.theme.Subtle).
		Italic(true).
		Render("press 2 or tab to head into Sparks · n to capture a new one")

	block := lipgloss.JoinVertical(lipgloss.Center, logoBlock, "", stats, "", hint)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, block)
}

func spacer(n int) string {
	return lipgloss.NewStyle().Width(n).Render("")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
