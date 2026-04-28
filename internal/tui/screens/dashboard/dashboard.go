package dashboard

import (
	"fmt"
	"strings"
	"time"

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
	projectCount int
	activity     []int
}

func New(t theme.Theme) screens.Screen { return Model{theme: t} }

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Title() string { return "Dashboard" }

func (m Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.ThemeChangedMsg:
		m.theme = theme.ByName(msg.ThemeName)
	case msgs.SparksLoadedMsg:
		m.sparkCount = 0
		m.activeCount = 0
		m.archivedSeen = 0
		m.activity = weeklySparkActivity(msg.Items, time.Now())
		for _, s := range msg.Items {
			m.sparkCount++
			switch s.Status {
			case domain.SparkStatusArchived:
				m.archivedSeen++
			case domain.SparkStatusNew, domain.SparkStatusQuestioning:
				m.activeCount++
			}
		}
	case msgs.ProjectsLoadedMsg:
		m.projectCount = len(msg.Items)
	}
	return m, nil
}

func (m Model) View(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	logoBlock := logo.Render(m.theme, min(width-8, 64))

	stat := func(label string, n int, color lipgloss.Color) string {
		const innerWidth = 11
		num := theme.Fg(m.theme, color).Bold(true).Render(fmt.Sprintf("%d", n))
		lbl := theme.Fg(m.theme, m.theme.Muted).Render(label)
		numLine := lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, num,
			lipgloss.WithWhitespaceBackground(m.theme.Background))
		lblLine := lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, lbl,
			lipgloss.WithWhitespaceBackground(m.theme.Background))
		content := lipgloss.JoinVertical(lipgloss.Center, numLine, lblLine)
		return theme.Base(m.theme).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color).
			Width(innerWidth).
			Render(content)
	}

	stats := lipgloss.JoinHorizontal(lipgloss.Top,
		stat("sparks", m.sparkCount, m.theme.Primary),
		spacer(m.theme, 2),
		stat("active", m.activeCount, m.theme.Accent),
		spacer(m.theme, 2),
		stat("projects", m.projectCount, m.theme.Success),
		spacer(m.theme, 2),
		stat("archived", m.archivedSeen, m.theme.Subtle),
	)

	tracking := m.trackingPreview(min(width-8, 60))

	block := lipgloss.JoinVertical(lipgloss.Center, logoBlock, "", stats, "", tracking)
	return theme.Place(m.theme, width, height, lipgloss.Center, lipgloss.Center, block)
}

func spacer(t theme.Theme, n int) string {
	return theme.Base(t).Width(n).Render("")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m Model) trackingPreview(width int) string {
	if width < 24 {
		width = 24
	}

	title := theme.Fg(m.theme, m.theme.Primary).Bold(true).Render("Tracking")
	values := m.activity
	if len(values) != 7 {
		values = make([]int, 7)
	}
	labels := []string{"M", "T", "W", "T", "F", "S", "S"}
	maxValue := 1
	for _, v := range values {
		if v > maxValue {
			maxValue = v
		}
	}

	rows := make([]string, 0, len(values))
	for i, v := range values {
		barLen := 0
		if v > 0 {
			barLen = max(1, v*12/maxValue)
		}
		bar := theme.Fg(m.theme, m.theme.Accent).Render(strings.Repeat("━", barLen))
		label := theme.Fg(m.theme, m.theme.Subtle).Render(labels[i])
		value := theme.Fg(m.theme, m.theme.Muted).Render(fmt.Sprintf("%2d", v))
		rows = append(rows, label+"  "+bar+" "+value)
	}

	chart := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return theme.Base(m.theme).Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Center, title, chart),
	)
}

func weeklySparkActivity(items []domain.Spark, now time.Time) []int {
	counts := make([]int, 7)
	start := startOfWeek(now)
	end := start.AddDate(0, 0, 7)

	for _, sp := range items {
		addActivity(counts, start, end, sp.CreatedAt)
		if !sameDay(sp.UpdatedAt, sp.CreatedAt) {
			addActivity(counts, start, end, sp.UpdatedAt)
		}
	}
	return counts
}

func addActivity(counts []int, start, end time.Time, ts time.Time) {
	if ts.IsZero() {
		return
	}
	ts = ts.In(start.Location())
	if ts.Before(start) || !ts.Before(end) {
		return
	}
	idx := int(ts.Sub(start).Hours() / 24)
	if idx >= 0 && idx < len(counts) {
		counts[idx]++
	}
}

func startOfWeek(t time.Time) time.Time {
	t = t.Local()
	year, month, day := t.Date()
	start := time.Date(year, month, day, 0, 0, 0, 0, t.Location())
	offset := (int(start.Weekday()) + 6) % 7
	return start.AddDate(0, 0, -offset)
}

func sameDay(a, b time.Time) bool {
	if a.IsZero() || b.IsZero() {
		return false
	}
	a = a.Local()
	b = b.Local()
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
