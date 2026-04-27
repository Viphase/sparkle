package sparks

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/theme"
)

// Saver is the slice of storage the sparks screen needs. The markdown.Store
// satisfies it; a fake satisfies it in tests.
type Saver interface {
	SaveSpark(domain.Spark) error
	ListSparks() ([]domain.Spark, error)
}

type mode int

const (
	modeList mode = iota
	modeForm
)

type Model struct {
	theme  theme.Theme
	saver  Saver
	items  []domain.Spark
	cursor int
	loaded bool

	mode  mode
	input textinput.Model

	now func() time.Time // injectable for tests
}

func New(t theme.Theme, saver Saver) screens.Screen {
	ti := textinput.New()
	ti.Placeholder = "Spark title"
	ti.CharLimit = 120
	ti.Prompt = "✦ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(t.Foreground)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(t.Accent)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(t.Subtle)

	return &Model{
		theme: t,
		saver: saver,
		input: ti,
		now:   time.Now,
	}
}

func (m *Model) Init() tea.Cmd { return nil }
func (m *Model) Title() string { return "Sparks" }

// InForm reports whether the sparks screen is currently in title-entry mode.
// The root model uses this to decide whether to consume top-level keys (q,
// tab, 1-6) or yield them to the input field.
func (m *Model) InForm() bool { return m.mode == modeForm }

func (m *Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.SparksLoadedMsg:
		m.items = msg.Items
		m.loaded = true
		if m.cursor >= len(m.items) {
			m.cursor = 0
			if len(m.items) > 0 {
				m.cursor = 0
			}
		}
		return m, nil
	case tea.KeyMsg:
		if m.mode == modeForm {
			return m.updateForm(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m *Model) updateList(key tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch key.String() {
	case "n":
		m.mode = modeForm
		m.input.Reset()
		m.input.Focus()
		return m, textinput.Blink
	case "j", "down":
		if m.cursor+1 < len(m.items) {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
		}
	}
	return m, nil
}

func (m *Model) updateForm(key tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.mode = modeList
		m.input.Blur()
		return m, nil
	case "enter":
		title := strings.TrimSpace(m.input.Value())
		if title == "" {
			return m, nil
		}
		m.mode = modeList
		m.input.Blur()
		return m, m.saveSparkCmd(title)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)
	return m, cmd
}

// saveSparkCmd builds a tea.Cmd that creates the spark, persists it, then
// reloads the list. Errors flow through the unified ErrorMsg envelope.
func (m *Model) saveSparkCmd(title string) tea.Cmd {
	saver := m.saver
	now := m.now().UTC()
	sp := domain.Spark{
		ID:        domain.NewSparkID(now),
		Title:     title,
		Status:    domain.SparkStatusNew,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return func() tea.Msg {
		if saver == nil {
			return msgs.ErrorMsg{Source: "save-spark", Err: fmt.Errorf("no storage configured")}
		}
		if err := saver.SaveSpark(sp); err != nil {
			return msgs.ErrorMsg{Source: "save-spark", Err: err}
		}
		items, err := saver.ListSparks()
		if err != nil {
			return msgs.ErrorMsg{Source: "list-sparks", Err: err}
		}
		return msgs.SparksLoadedMsg{Items: items}
	}
}

func (m *Model) View(width, height int) string {
	box := lipgloss.NewStyle().Padding(1, 2).Width(width).Height(height)
	header := theme.ApplyGrad("✦ Sparks", m.theme.GradientFrom, m.theme.GradientTo, true)

	if m.mode == modeForm {
		return box.Render(m.formView(width, height, header))
	}

	if !m.loaded {
		return box.Render(centerCard(width, height, m.theme,
			header,
			lipgloss.NewStyle().Foreground(m.theme.Muted).Render("Loading sparks…"),
		))
	}

	if len(m.items) == 0 {
		return box.Render(centerCard(width, height, m.theme,
			header,
			lipgloss.NewStyle().Foreground(m.theme.Foreground).Render("No sparks yet."),
			lipgloss.NewStyle().Foreground(m.theme.Subtle).Italic(true).
				Render("press n to capture your first one"),
		))
	}

	rows := []string{header, ""}
	for i, sp := range m.items {
		rows = append(rows, m.renderRow(i, sp, width-4))
	}
	rows = append(rows, "",
		lipgloss.NewStyle().Foreground(m.theme.Subtle).
			Render(fmt.Sprintf("%d sparks  ·  n new  ·  j/k move  ·  esc back", len(m.items))),
	)

	return box.Render(strings.Join(rows, "\n"))
}

func (m *Model) renderRow(i int, sp domain.Spark, w int) string {
	title := strings.TrimSpace(sp.Title)
	if title == "" {
		title = "(untitled)"
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(m.statusColor(sp.Status)).
		Width(13)
	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Foreground)
	cursor := "  "

	if i == m.cursor {
		cursor = lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true).Render("▌ ")
		titleStyle = titleStyle.Foreground(m.theme.Primary).Bold(true)
	}
	statusCell := statusStyle.Render(string(sp.Status))
	titleCell := titleStyle.Render(title)

	return cursor + statusCell + "  " + titleCell
}

func (m *Model) statusColor(s domain.SparkStatus) lipgloss.Color {
	switch s {
	case domain.SparkStatusNew:
		return m.theme.Info
	case domain.SparkStatusQuestioning:
		return m.theme.Warning
	case domain.SparkStatusPromoted:
		return m.theme.Success
	case domain.SparkStatusArchived:
		return m.theme.Subtle
	}
	return m.theme.Muted
}

func (m *Model) formView(width, height int, header string) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderFocus).
		Padding(1, 2).
		Width(min(width-8, 56))

	prompt := lipgloss.NewStyle().Foreground(m.theme.Subtle).
		Render("Capture a spark — enter to save, esc to cancel")

	card := border.Render(lipgloss.JoinVertical(lipgloss.Left,
		prompt,
		"",
		m.input.View(),
	))

	block := lipgloss.JoinVertical(lipgloss.Center, header, "", card)
	return lipgloss.Place(width-4, height-2, lipgloss.Center, lipgloss.Center, block)
}

func centerCard(width, height int, t theme.Theme, header string, lines ...string) string {
	body := append([]string{header, ""}, lines...)
	block := lipgloss.JoinVertical(lipgloss.Center, body...)
	return lipgloss.Place(width-4, height-2, lipgloss.Center, lipgloss.Center, block)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
