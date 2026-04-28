// Package statusbar renders the bottom status bar and tracks the latest
// info or error message surfaced via msgs.StatusMsg / msgs.ErrorMsg.
package statusbar

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/viphase/sparkle/internal/tui/theme"
)

const (
	defaultHint = "tab nav · 1-5 tabs · ? keys · q quit"
	helpHint    = "tab/shift+tab tabs · 1-5 jump · n new spark · e edit · a archive · / search · h archived · g/G first/last · q quit"
)

type Model struct {
	theme theme.Theme
	info  string
	err   string
	help  bool
}

func New(t theme.Theme) Model {
	return Model{theme: t, info: defaultHint}
}

func (m Model) WithTheme(t theme.Theme) Model { m.theme = t; return m }

func (m Model) ToggleHelp() Model {
	m.help = !m.help
	m.err = ""
	return m
}

func (m Model) SetInfo(s string) Model {
	m.info = s
	m.err = ""
	m.help = false
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
	m.help = false
	return m
}

func (m Model) Clear() Model {
	m.info = defaultHint
	m.err = ""
	m.help = false
	return m
}

func (m Model) View(width int) string {
	bar := theme.Base(m.theme).
		Padding(0, 2).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(m.theme.Border)
	if width > 0 {
		bar = bar.Width(width).MaxWidth(width)
	}
	if m.err != "" {
		badge := theme.Fg(m.theme, m.theme.Danger).
			Bold(true).
			Padding(0, 1).
			Render(" error ")
		return bar.Render(m.truncate(width, badge+" "+
			theme.Fg(m.theme, m.theme.Danger).Render(m.err)))
	}
	text := m.info
	if m.help {
		text = helpHint
	}
	return bar.Foreground(m.theme.Muted).Render(m.truncate(width, text))
}

func (m Model) truncate(width int, text string) string {
	if width <= 4 {
		return text
	}
	return ansi.Truncate(text, width-4, "…")
}
