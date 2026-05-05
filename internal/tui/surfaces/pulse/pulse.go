package pulse

import (
	"fmt"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/NimbleMarkets/ntcharts/heatmap"
	"github.com/NimbleMarkets/ntcharts/sparkline"
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
	activeCount  int
	projectCount int
	projects     []domain.Project // M13: needed for pipeline & velocity panel
	stats        domain.TrackingStats
	allEvents    map[string][]domain.TrackingEvent
	now          func() time.Time
	scroll       int

	// Logo cache: re-render only when available width changes.
	cachedLogoW int
	cachedLogo  string
	cachedLogoH int

	// Merged event cache: recomputed only on TrackingLoadedMsg.
	cachedMerged []domain.TrackingEvent
}

func New(t theme.Theme) screens.Screen { return &Model{theme: t, now: time.Now} }

func (m *Model) Init() tea.Cmd { return nil }
func (m *Model) Title() string { return "Pulse" }

func (m *Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.ThemeChangedMsg:
		m.theme = theme.ByName(msg.ThemeName)
		m.cachedLogoW = 0
	case msgs.SparksLoadedMsg:
		m.activeCount = 0
		for _, s := range msg.Items {
			if s.Status != domain.SparkStatusArchived {
				m.activeCount++
			}
		}
	case msgs.ProjectsLoadedMsg:
		m.projectCount = len(msg.Items)
		m.projects = msg.Items
	case msgs.TrackingLoadedMsg:
		m.allEvents = msg.AllEvents
		m.cachedMerged = mergeAllEvents(msg.AllEvents)
		m.stats = tracker.Compute(m.cachedMerged, m.clock())
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.scroll += 2
		case "k", "up":
			m.scroll -= 2
			if m.scroll < 0 {
				m.scroll = 0
			}
		case "g":
			m.scroll = 0
		case "G":
			m.scroll = 9999
		}
	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelDown:
			m.scroll += 3
		case tea.MouseWheelUp:
			m.scroll -= 3
			if m.scroll < 0 {
				m.scroll = 0
			}
		}
	}
	return m, nil
}

func (m *Model) clock() time.Time {
	if m.now != nil {
		return m.now()
	}
	return time.Now()
}

func mergeAllEvents(all map[string][]domain.TrackingEvent) []domain.TrackingEvent {
	var out []domain.TrackingEvent
	for _, evs := range all {
		out = append(out, evs...)
	}
	return out
}

func (m *Model) View(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	// Narrow: compact single-column view.
	if width < 50 {
		m.cachedLogoW = 0
		streakColor := m.theme.Accent
		if m.stats.CurrentStreak >= 7 {
			streakColor = m.theme.Success
		}
		lines := []string{
			logo.Compact(m.theme),
			"",
			theme.Fg(m.theme, m.theme.Primary).Bold(true).Render(fmt.Sprintf("%d active sparks", m.activeCount)),
			theme.Fg(m.theme, m.theme.Success).Bold(true).Render(fmt.Sprintf("%d projects", m.projectCount)),
			theme.Fg(m.theme, streakColor).Bold(true).Render(fmt.Sprintf("%d streak days", m.stats.CurrentStreak)),
		}
		block := lipgloss.JoinVertical(lipgloss.Center, lines...)
		return theme.Place(m.theme, width, height, lipgloss.Center, lipgloss.Center, block)
	}

	// ── Logo (cached) ──────────────────────────────────────────────────────
	logoW := width - 4
	useFullLogo := height >= 12 && width >= 80 // L8: compact logo below 80 cols
	if !useFullLogo {
		logoW = 0
	}
	if m.cachedLogoW != logoW {
		if useFullLogo {
			m.cachedLogo = logo.Render(m.theme, logoW)
		} else {
			m.cachedLogo = logo.Compact(m.theme)
		}
		m.cachedLogoH = lipgloss.Height(m.cachedLogo)
		m.cachedLogoW = logoW
	}

	// ── Hero stat blocks — responsive width ───────────────────────────────
	// Scale each box between 10 and 18 chars based on available width.
	innerWidth := clamp((width-24)/3, 10, 18)
	heroRow := m.buildHeroRow(innerWidth)

	heroRowCentered := lipgloss.PlaceHorizontal(width, lipgloss.Center, heroRow,
		lipgloss.WithWhitespaceBackground(m.theme.Background))

	const heroH = 4
	fixedH := m.cachedLogoH + 1 + heroH
	fixedBlock := lipgloss.JoinVertical(lipgloss.Center, m.cachedLogo, "", heroRowCentered)

	// ── Scrollable charts panel ────────────────────────────────────────────
	// Use most of the width; do NOT hard-cap at 100 so wide terminals can
	// show wider charts.
	detailH := height - fixedH - 2
	if detailH < 1 {
		detailH = 1
	}
	detailW := width - 8
	if detailW < 24 {
		detailW = 24
	}

	// Pass an unrestricted maxH so the panel always builds every chart that fits.
	// We then slice the result against detailH for the scroll viewport — this
	// avoids the previous magic +30 that broke at small heights.
	detailContent := m.chartsPanel(detailW, 9999)

	detailLines := strings.Split(detailContent, "\n")
	totalLines := len(detailLines)

	maxScroll := maxInt(0, totalLines-detailH)
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
	visible := detailLines[m.scroll:]
	if len(visible) > detailH {
		visible = visible[:detailH]
	}

	scrollHint := ""
	if m.scroll+detailH < totalLines || m.scroll > 0 {
		pct := ""
		if maxScroll > 0 {
			pct = fmt.Sprintf(" %d%%", m.scroll*100/maxScroll)
		}
		scrollHint = theme.Fg(m.theme, m.theme.Subtle).Render("↑↓ / jk  scroll" + pct)
	}

	detailBlock := theme.Base(m.theme).Width(detailW).Render(strings.Join(visible, "\n"))
	detailCentered := lipgloss.PlaceHorizontal(width, lipgloss.Center, detailBlock,
		lipgloss.WithWhitespaceBackground(m.theme.Background))

	parts := []string{fixedBlock, "", detailCentered}
	if scrollHint != "" {
		scrollHintCentered := lipgloss.PlaceHorizontal(width, lipgloss.Center, scrollHint,
			lipgloss.WithWhitespaceBackground(m.theme.Background))
		parts = append(parts, scrollHintCentered)
	}
	block := lipgloss.JoinVertical(lipgloss.Center, parts...)
	return theme.Place(m.theme, width, height, lipgloss.Center, lipgloss.Top, block)
}

// buildHeroRow assembles the three stat boxes with a responsive innerWidth.
func (m *Model) buildHeroRow(innerWidth int) string {
	streakColor := m.theme.Accent
	if m.stats.CurrentStreak >= 7 {
		streakColor = m.theme.Success
	}
	stat := func(label string, n int, color lipgloss.Color) string {
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
	sp := spacer(m.theme, 2)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		stat("active sparks", m.activeCount, m.theme.Primary),
		sp,
		stat("active projects", m.projectCount, m.theme.Success),
		sp,
		stat("streak days", m.stats.CurrentStreak, streakColor),
	)
}

// chartsPanel renders the scrollable charts section using real ntcharts canvases.
// Each section is centered horizontally within the available width.
func (m *Model) chartsPanel(width, maxH int) string {
	now := m.clock()
	merged := m.cachedMerged

	center := func(s string) string {
		return lipgloss.PlaceHorizontal(width, lipgloss.Center, s,
			lipgloss.WithWhitespaceBackground(m.theme.Background))
	}

	rows := []string{center(m.statsRow())}

	if maxH >= 8 {
		rows = append(rows, "", center(m.weeklyBarChart(merged, now, width)))
	}
	if maxH >= 14 {
		rows = append(rows, "", center(m.activityHeatmapChart(merged, now, width)))
	}
	if maxH >= 24 {
		rows = append(rows, "", center(m.trendSparkline(merged, now, width)))
	}
	if maxH >= 30 || len(m.projects) > 0 {
		rows = append(rows, "", center(m.pipelinePanel(now, width)))
	}

	return theme.Base(m.theme).Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

// pipelinePanel renders the fifth Pulse panel: a row per active project showing
// pipeline stage and 14-day velocity. Answers the fourth Pulse question
// ("where is each project in its pipeline?").
//
// Pure rendering; data comes from m.projects + per-project events.
func (m *Model) pipelinePanel(now time.Time, width int) string {
	caption := theme.Fg(m.theme, m.theme.Subtle).Render("active projects · pipeline & velocity")
	if len(m.projects) == 0 {
		empty := theme.Fg(m.theme, m.theme.Muted).Render(
			"no projects yet — promote a spark to start the pipeline")
		return lipgloss.JoinVertical(lipgloss.Left, caption, empty)
	}

	stageGlyph := map[tracker.Stage]string{
		tracker.StageSpark:    "✦",
		tracker.StageShaping:  "◐",
		tracker.StageBuilding: "▲",
		tracker.StageShipping: "◆",
		tracker.StageDone:     "✓",
	}
	stageColor := map[tracker.Stage]lipgloss.Color{
		tracker.StageSpark:    m.theme.Accent,
		tracker.StageShaping:  m.theme.Primary,
		tracker.StageBuilding: m.theme.Warning,
		tracker.StageShipping: m.theme.Success,
		tracker.StageDone:     m.theme.Muted,
	}

	rows := []string{caption}
	const titleW = 22
	const stageW = 10
	for _, p := range m.projects {
		if p.Status == domain.ProjectStatusArchived {
			continue
		}
		evs := m.allEvents[p.ID]
		stage := tracker.PipelineStage(p, evs, now)
		velocity := tracker.ProjectVelocity(evs, now, 14*24*time.Hour)

		title := p.Title
		if len(title) > titleW {
			title = title[:titleW-1] + "…"
		}
		titleCell := theme.Fg(m.theme, m.theme.Foreground).
			Render(fmt.Sprintf("%-*s", titleW, title))
		stageCell := theme.Fg(m.theme, stageColor[stage]).Bold(true).
			Render(fmt.Sprintf("%s %-*s", stageGlyph[stage], stageW-2, stage))
		velCell := theme.Fg(m.theme, m.theme.Subtle).
			Render(fmt.Sprintf("%6.1f w/d", velocity))
		row := lipgloss.JoinHorizontal(lipgloss.Top, titleCell, stageCell, velCell)
		rows = append(rows, row)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *Model) statsRow() string {
	s := m.stats
	// cellW is the content area width passed to lipgloss Width().
	// In lipgloss, Width() is the inner content width; border adds 2 more.
	const cellW = 12
	cell := func(label, val string, color lipgloss.Color) string {
		// PlaceHorizontal centers the text within the cell content area.
		vLine := lipgloss.PlaceHorizontal(cellW, lipgloss.Center,
			theme.Fg(m.theme, color).Bold(true).Render(val),
			lipgloss.WithWhitespaceBackground(m.theme.Background))
		lLine := lipgloss.PlaceHorizontal(cellW, lipgloss.Center,
			theme.Fg(m.theme, m.theme.Muted).Render(label),
			lipgloss.WithWhitespaceBackground(m.theme.Background))
		inner := lipgloss.JoinVertical(lipgloss.Center, vLine, lLine)
		return theme.Base(m.theme).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color).
			Width(cellW).
			Render(inner)
	}
	sp := theme.Base(m.theme).Width(2).Render("")
	return lipgloss.JoinHorizontal(lipgloss.Top,
		cell("today words", fmt.Sprintf("%d", s.TodayWords), m.theme.Primary),
		sp,
		cell("week words", fmt.Sprintf("%d", s.WeekWords), m.theme.Accent),
		sp,
		cell("days active", fmt.Sprintf("%d/7", s.ActiveDaysWeek), m.theme.Success),
	)
}

// weeklyBarChart renders a real ntcharts BarChart of words per day this week.
func (m *Model) weeklyBarChart(events []domain.TrackingEvent, now time.Time, width int) string {
	weekDays := tracker.WeeklyWordsByDay(events, now)
	labels := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

	caption := theme.Fg(m.theme, m.theme.Subtle).Render("words added · this week")

	chartW := minInt(width, 56)
	chartH := 5

	barStyle := lipgloss.NewStyle().Foreground(m.theme.Accent).Background(m.theme.Background)
	axisStyle := lipgloss.NewStyle().Foreground(m.theme.Muted).Background(m.theme.Background)
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.Subtle).Background(m.theme.Background)

	bc := barchart.New(chartW, chartH,
		barchart.WithStyles(axisStyle, labelStyle),
	)

	var data []barchart.BarData
	for i, v := range weekDays {
		data = append(data, barchart.BarData{
			Label: labels[i],
			Values: []barchart.BarValue{
				{Name: "words", Value: float64(v), Style: barStyle},
			},
		})
	}
	bc.PushAll(data)
	bc.Draw()

	// Append raw numbers alongside chart.
	chartView := bc.View()
	valLine := "     " // align with chart labels
	for _, v := range weekDays {
		valLine += fmt.Sprintf("%-5d", v)
	}
	valLine = theme.Fg(m.theme, m.theme.Muted).Render(valLine)

	return lipgloss.JoinVertical(lipgloss.Left, caption, chartView, valLine)
}

// activityHeatmapChart renders a calendar-grid heatmap of last 30 days.
func (m *Model) activityHeatmapChart(events []domain.TrackingEvent, now time.Time, width int) string {
	// 7-column (days of week) × 5-row (weeks) grid.
	activity := tracker.Last30DaysActivity(events, now)

	caption := theme.Fg(m.theme, m.theme.Subtle).Render("activity · last 30 days")

	chartW := minInt(width, 42)
	chartH := 7

	colorScale := []lipgloss.Color{
		m.theme.Surface, // 0 = inactive
		m.theme.Success, // 1 = active
	}

	hm := heatmap.New(chartW, chartH,
		heatmap.WithColorScale(colorScale),
		heatmap.WithValueRange(0, 1),
	)

	for i, active := range activity {
		dayOffset := 29 - i
		d := now.AddDate(0, 0, -dayOffset)
		col := int(d.Weekday()+6) % 7
		row := 4 - (dayOffset / 7)
		if row < 0 {
			row = 0
		}
		val := 0.0
		if active {
			val = 1.0
		}
		hm.Push(heatmap.NewHeatPointInt(col, row, val))
	}
	hm.Draw()

	dayLabels := theme.Fg(m.theme, m.theme.Subtle).Render("M  T  W  T  F  S  S")

	return lipgloss.JoinVertical(lipgloss.Left, caption, hm.View(), dayLabels)
}

// trendSparkline renders a 12-week word trend using a braille sparkline.
func (m *Model) trendSparkline(events []domain.TrackingEvent, now time.Time, width int) string {
	weeks := tracker.Last12WeeksWords(events, now)

	caption := theme.Fg(m.theme, m.theme.Subtle).Render("words · 12-week trend")

	chartW := minInt(width-4, 60)
	chartH := 4

	sl := sparkline.New(chartW, chartH,
		sparkline.WithStyle(lipgloss.NewStyle().Foreground(m.theme.Primary).Background(m.theme.Background)),
	)

	vals := make([]float64, 12)
	for i, v := range weeks {
		vals[i] = float64(v)
	}
	sl.PushAll(vals)
	sl.DrawBraille()

	oldest := now.AddDate(0, 0, -77).Format("Jan 2")
	newest := now.Format("Jan 2")
	rangeLabel := theme.Fg(m.theme, m.theme.Muted).Render(
		fmt.Sprintf("%-*s%s", chartW-len(newest), oldest, newest),
	)

	return lipgloss.JoinVertical(lipgloss.Left, caption, sl.View(), rangeLabel)
}

func spacer(t theme.Theme, n int) string {
	return theme.Base(t).Width(n).Render("")
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
