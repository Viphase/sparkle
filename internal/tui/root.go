package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/tui/theme"
)

type Root struct {
	theme  theme.Theme
	width  int
	height int
}

func NewRoot() Root {
	return Root{theme: theme.PastelDark()}
}

func (r Root) Init() tea.Cmd { return nil }

func (r Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return r, tea.Quit
		}
	}
	return r, nil
}

func (r Root) View() string {
	title := lipgloss.NewStyle().
		Foreground(r.theme.Primary).
		Bold(true).
		Padding(1, 2).
		Render("Sparkle")

	subtitle := lipgloss.NewStyle().
		Foreground(r.theme.Foreground).
		Padding(0, 2).
		Render("Turn rough sparks into structured projects.")

	stage := lipgloss.NewStyle().
		Foreground(r.theme.Accent).
		Padding(1, 2).
		Render("Milestone 1 — local TUI foundation")

	footer := lipgloss.NewStyle().
		Foreground(r.theme.Muted).
		Padding(1, 2).
		Render("press q to quit")

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, stage, footer)
}
