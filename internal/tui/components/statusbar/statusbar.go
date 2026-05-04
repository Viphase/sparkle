// Package statusbar renders the bottom status bar and tracks the latest
// info or error message surfaced via msgs.StatusMsg / msgs.ErrorMsg.
package statusbar

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/viphase/sparkle/internal/tui/theme"
)

const (
	defaultHint = "tab  switch surface  ·  ?  help  ·  q  quit"
	helpHint    = "tab / shift+tab  switch  ·  1-3  jump  ·  ?  this help  ·  q  quit"
)

type Model struct {
	theme theme.Theme
	info  string
	err   string
	help  bool
	// hint is the contextual key reference for the current surface.
	// It is shown after a separator when info == "" or matches defaultHint,
	// so transient status messages (set via SetInfo) take precedence but
	// surface keys come back automatically afterwards.
	hint string
}

func New(t theme.Theme) Model {
	return Model{theme: t, info: defaultHint}
}

// SetHint replaces the surface-specific key hint. Pass the empty string to
// fall back to the global default. This is sticky — it survives until the next
// SetHint call, while SetInfo only displays a transient message.
func (m Model) SetHint(s string) Model { m.hint = s; return m }

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

// HasError reports whether an error is currently shown.
func (m Model) HasError() bool { return m.err != "" }

// ClearError dismisses the current error and returns to the default hint.
func (m Model) ClearError() Model {
	m.err = ""
	m.info = defaultHint
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
		dismiss := theme.Fg(m.theme, m.theme.Subtle).Render("  esc to dismiss")
		msg := badge + " " + theme.Fg(m.theme, m.theme.Danger).Render(m.err) + dismiss
		return bar.Render(m.truncate(width, msg))
	}
	var text string
	switch {
	case m.help:
		text = helpHint
	case m.info != "" && m.info != defaultHint && m.hint != "":
		text = m.info + "    " + theme.Fg(m.theme, m.theme.Subtle).Render(m.hint)
	case m.hint != "":
		text = m.hint
	default:
		text = m.info
	}
	return bar.Foreground(m.theme.Muted).Render(m.truncate(width, text))
}

func (m Model) truncate(width int, text string) string {
	if width <= 4 {
		return text
	}
	return ansi.Truncate(text, width-4, "…")
}
