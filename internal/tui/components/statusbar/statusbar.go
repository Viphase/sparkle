// Package statusbar renders the bottom status bar and tracks the latest
// info or error message surfaced via msgs.StatusMsg / msgs.ErrorMsg.
package statusbar

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/tui/theme"
)

const defaultHint = "tab next  ·  shift+tab prev  ·  1-6 jump  ·  n new  ·  q quit"

type Model struct {
	theme theme.Theme
	info  string
	err   string
}

func New(t theme.Theme) Model {
	return Model{theme: t, info: defaultHint}
}

func (m Model) WithTheme(t theme.Theme) Model { m.theme = t; return m }

func (m Model) SetInfo(s string) Model {
	m.info = s
	m.err = ""
	return m
}

func (m Model) SetError(source string, err error) Model {
	if err == nil {
		return m
	}
	if source == "" {
		m.err = err.Error()
	} else {
		m.err = source + ": " + err.Error()
	}
	return m
}

func (m Model) Clear() Model {
	m.info = defaultHint
	m.err = ""
	return m
}

func (m Model) View(width int) string {
	bar := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(m.theme.Border)
	if width > 0 {
		bar = bar.Width(width)
	}
	if m.err != "" {
		badge := lipgloss.NewStyle().
			Foreground(m.theme.Background).
			Background(m.theme.Danger).
			Bold(true).
			Padding(0, 1).
			Render(" error ")
		return bar.Render(badge + " " +
			lipgloss.NewStyle().Foreground(m.theme.Danger).Render(m.err))
	}
	return bar.Foreground(m.theme.Muted).Render(m.info)
}
