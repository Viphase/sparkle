package dashboard

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tracker"
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
	stats        domain.TrackingStats
	allEvents    map[string][]domain.TrackingEvent
	now          func() time.Time
}

func New(t theme.Theme) screens.Screen { return Model{theme: t, now: time.Now} }

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
	case msgs.TrackingLoadedMsg:
		m.allEvents = msg.AllEvents
		merged := mergeAllEvents(msg.AllEvents)
		m.stats = tracker.Compute(merged, m.clock())
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
	// Pass the full content width so the block-pixel logo can render.
	// The logo itself is 8 rows (7 letter rows + byline); fall back to the
	// compact single-line form only when the content area is very small.
	logoBlock := logo.Render(m.theme, width-4)
	if height < 12 {
		logoBlock = logo.Compact(m.theme)
	}

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

	trackingHeight := height - lipgloss.Height(logoBlock) - lipgloss.Height(stats) - 2
	if trackingHeight < 1 {
		trackingHeight = 1
	}
	tracking := m.trackingPanel(min(width-8, 64), trackingHeight)

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

func (m Model) trackingPanel(width, height int) string {
	if width < 24 {
		width = 24
	}

	merged := mergeAllEvents(m.allEvents)
	now := m.clock()

	statsRow := m.trackingStatsRow()
	rows := []string{statsRow}
	if height >= 14 {
		rows = append(rows,
			m.weeklyWordsChart(merged, now, width),
			m.activityHeatmap(merged, now, width),
		)
	} else if height >= 12 {
		rows = append(rows, m.weeklyWordsChart(merged, now, width))
	} else if height >= 6 {
		rows = append(rows, m.activityHeatmap(merged, now, width))
	}

	return theme.Base(m.theme).Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Center, rows...),
	)
}

func (m Model) trackingStatsRow() string {
	s := m.stats
	streakColor := m.theme.Accent
	if s.CurrentStreak >= 7 {
		streakColor = m.theme.Success
	}

	cell := func(label string, val string, color lipgloss.Color) string {
		v := theme.Fg(m.theme, color).Bold(true).Render(val)
		l := theme.Fg(m.theme, m.theme.Muted).Render(label)
		inner := lipgloss.JoinVertical(lipgloss.Center, v, l)
		return theme.Base(m.theme).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color).
			Width(12).
			Render(inner)
	}

	sp := theme.Base(m.theme).Width(1).Render("")
	return lipgloss.JoinHorizontal(lipgloss.Top,
		cell("today words", fmt.Sprintf("%d", s.TodayWords), m.theme.Primary),
		sp,
		cell("week words", fmt.Sprintf("%d", s.WeekWords), m.theme.Accent),
		sp,
		cell("streak", fmt.Sprintf("%dd", s.CurrentStreak), streakColor),
		sp,
		cell("active days", fmt.Sprintf("%d", s.ActiveDaysWeek), m.theme.Success),
	)
}

func (m Model) weeklyWordsChart(events []domain.TrackingEvent, now time.Time, width int) string {
	if width < 24 {
		width = 24
	}
	weekDays := tracker.WeeklyWordsByDay(events, now)
	labels := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	maxValue := 1
	for _, v := range weekDays {
		if v > maxValue {
			maxValue = v
		}
	}

	barWidth := (width - 20) / 7
	if barWidth < 2 {
		barWidth = 2
	}

	caption := theme.Fg(m.theme, m.theme.Subtle).Render("words added · this week")
	rows := make([]string, 0, 7)
	for i, v := range weekDays {
		filled := 0
		if v > 0 {
			filled = max(1, v*barWidth/maxValue)
		}
		bar := theme.Fg(m.theme, m.theme.Accent).Render(strings.Repeat("█", filled))
		empty := theme.Fg(m.theme, m.theme.Muted).Render(strings.Repeat("░", barWidth-filled))
		label := theme.Fg(m.theme, m.theme.Subtle).Render(fmt.Sprintf("%-3s", labels[i]))
		value := theme.Fg(m.theme, m.theme.Primary).Render(fmt.Sprintf("%4d", v))
		rows = append(rows, label+"  "+bar+empty+"  "+value)
	}

	chart := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return theme.Base(m.theme).Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, caption, chart),
	)
}

func (m Model) activityHeatmap(events []domain.TrackingEvent, now time.Time, width int) string {
	activity := tracker.Last30DaysActivity(events, now)
	title := theme.Fg(m.theme, m.theme.Subtle).Render("activity · last 30 days")

	cells := make([]string, 0, len(activity))
	for i, active := range activity {
		if i > 0 && i%5 == 0 {
			cells = append(cells, " ")
		}
		if active {
			cells = append(cells, theme.Fg(m.theme, m.theme.Success).Render("■"))
		} else {
			cells = append(cells, theme.Fg(m.theme, m.theme.Muted).Render("·"))
		}
	}

	row := strings.Join(cells, "")
	return theme.Base(m.theme).Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, row),
	)
}

func (m Model) clock() time.Time {
	if m.now == nil {
		return time.Now()
	}
	return m.now()
}

func mergeAllEvents(all map[string][]domain.TrackingEvent) []domain.TrackingEvent {
	total := 0
	for _, evs := range all {
		total += len(evs)
	}
	merged := make([]domain.TrackingEvent, 0, total)
	for _, evs := range all {
		merged = append(merged, evs...)
	}
	return merged
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

