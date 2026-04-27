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

// Saver is the slice of storage the sparks screen needs. *markdown.Store
// satisfies it; tests use a fake.
type Saver interface {
	SaveSpark(domain.Spark) error
	LoadSpark(id string) (domain.Spark, error)
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

	mode      mode
	input     textinput.Model
	editingID string // empty when creating, set when editing

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

// InForm reports whether the screen is currently in title-entry mode.
func (m *Model) InForm() bool { return m.mode == modeForm }

func (m *Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.SparksLoadedMsg:
		m.items = msg.Items
		m.loaded = true
		if m.cursor >= len(m.items) {
			m.cursor = 0
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
		m.editingID = ""
		m.input.Reset()
		m.input.Focus()
		return m, textinput.Blink
	case "e":
		if len(m.items) == 0 {
			return m, nil
		}
		sp := m.items[m.cursor]
		m.mode = modeForm
		m.editingID = sp.ID
		m.input.SetValue(sp.Title)
		m.input.CursorEnd()
		m.input.Focus()
		return m, textinput.Blink
	case "a":
		if len(m.items) == 0 {
			return m, nil
		}
		return m, m.toggleArchiveCmd(m.items[m.cursor])
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
		m.editingID = ""
		m.input.Blur()
		return m, nil
	case "enter":
		title := strings.TrimSpace(m.input.Value())
		if title == "" {
			return m, nil
		}
		m.mode = modeList
		m.input.Blur()
		cmd := m.saveSparkCmd(title)
		m.editingID = ""
		return m, cmd
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)
	return m, cmd
}

// saveSparkCmd creates or updates a spark, then re-lists. When editingID is
// set, the existing spark is loaded so its description/tags/created_at survive
// the title edit.
func (m *Model) saveSparkCmd(title string) tea.Cmd {
	saver := m.saver
	editingID := m.editingID
	now := m.now().UTC()
	return func() tea.Msg {
		if saver == nil {
			return msgs.ErrorMsg{Source: "save-spark", Err: fmt.Errorf("no storage configured")}
		}
		var sp domain.Spark
		if editingID != "" {
			existing, err := saver.LoadSpark(editingID)
			if err != nil {
				return msgs.ErrorMsg{Source: "load-spark", Err: err}
			}
			existing.Title = title
			existing.UpdatedAt = now
			sp = existing
		} else {
			sp = domain.Spark{
				ID:        domain.NewSparkID(now),
				Title:     title,
				Status:    domain.SparkStatusNew,
				CreatedAt: now,
				UpdatedAt: now,
			}
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

// toggleArchiveCmd flips a spark's archived state. Archived → new; anything
// else → archived. UpdatedAt is bumped.
func (m *Model) toggleArchiveCmd(sp domain.Spark) tea.Cmd {
	saver := m.saver
	now := m.now().UTC()
	if sp.Status == domain.SparkStatusArchived {
		sp.Status = domain.SparkStatusNew
	} else {
		sp.Status = domain.SparkStatusArchived
	}
	sp.UpdatedAt = now
	return func() tea.Msg {
		if saver == nil {
			return msgs.ErrorMsg{Source: "archive-spark", Err: fmt.Errorf("no storage configured")}
		}
		if err := saver.SaveSpark(sp); err != nil {
			return msgs.ErrorMsg{Source: "archive-spark", Err: err}
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
		return box.Render(centerCard(width, height,
			header,
			lipgloss.NewStyle().Foreground(m.theme.Muted).Render("Loading sparks…"),
		))
	}

	if len(m.items) == 0 {
		return box.Render(centerCard(width, height,
			header,
			lipgloss.NewStyle().Foreground(m.theme.Foreground).Render("No sparks yet."),
			lipgloss.NewStyle().Foreground(m.theme.Subtle).Italic(true).
				Render("press n to capture your first one"),
		))
	}

	rows := []string{header, ""}
	for i, sp := range m.items {
		rows = append(rows, m.renderRow(i, sp))
	}
	rows = append(rows, "",
		lipgloss.NewStyle().Foreground(m.theme.Subtle).
			Render(fmt.Sprintf("%d sparks  ·  n new  ·  e edit  ·  a archive  ·  j/k move", len(m.items))),
	)

	return box.Render(strings.Join(rows, "\n"))
}

func (m *Model) renderRow(i int, sp domain.Spark) string {
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
	if sp.Status == domain.SparkStatusArchived {
		titleStyle = titleStyle.Foreground(m.theme.Subtle).Strikethrough(true)
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

	promptText := "Capture a spark — enter to save, esc to cancel"
	if m.editingID != "" {
		promptText = "Edit spark — enter to save, esc to cancel"
	}
	prompt := lipgloss.NewStyle().Foreground(m.theme.Subtle).Render(promptText)

	card := border.Render(lipgloss.JoinVertical(lipgloss.Left,
		prompt,
		"",
		m.input.View(),
	))

	block := lipgloss.JoinVertical(lipgloss.Center, header, "", card)
	return lipgloss.Place(width-4, height-2, lipgloss.Center, lipgloss.Center, block)
}

func centerCard(width, height int, header string, lines ...string) string {
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
