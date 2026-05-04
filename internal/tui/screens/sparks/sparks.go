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
	DeleteSpark(id string) error
}

// Promoter handles project creation when a spark is promoted. It is separate
// from Saver so tests don't need to implement project storage.
type Promoter interface {
	SaveProject(domain.Project) error
	ListProjects() ([]domain.Project, error)
}

type mode int

const (
	modeList mode = iota
	modeForm
	modeSearch
)

type Model struct {
	theme        theme.Theme
	saver        Saver
	promoter     Promoter
	items        []domain.Spark
	cursor       int
	offset       int
	listHeight   int
	showArchived bool
	loaded       bool

	mode      mode
	input     textinput.Model
	editingID string // empty when creating, set when editing

	searchInput   textinput.Model
	query         string
	deleteConfirm bool

	now func() time.Time // injectable for tests
}

// New wires a sparks screen. Pass an optional Promoter to enable spark→project
// promotion; nil disables the p key.
func New(t theme.Theme, saver Saver, promoters ...Promoter) screens.Screen {
	ti := textinput.New()
	ti.Placeholder = "Spark title"
	ti.CharLimit = 120
	ti.Prompt = "::: "
	ti.PromptStyle = theme.Fg(t, t.Primary).Bold(true)
	ti.TextStyle = theme.Fg(t, t.Foreground)
	ti.Cursor.Style = theme.Fg(t, t.Accent)
	ti.PlaceholderStyle = theme.Fg(t, t.Subtle)

	search := textinput.New()
	search.Placeholder = "Search title, description, or tags"
	search.CharLimit = 120
	search.Prompt = "/ "
	search.PromptStyle = theme.Fg(t, t.Accent).Bold(true)
	search.TextStyle = theme.Fg(t, t.Foreground)
	search.Cursor.Style = theme.Fg(t, t.Accent)
	search.PlaceholderStyle = theme.Fg(t, t.Subtle)

	var promoter Promoter
	if len(promoters) > 0 {
		promoter = promoters[0]
	}

	return &Model{
		theme:    t,
		saver:    saver,
		promoter: promoter,
		input:    ti,
		searchInput: search,
		now:      time.Now,
	}
}

func (m *Model) Init() tea.Cmd { return nil }
func (m *Model) Title() string { return "Sparks" }

// InForm reports whether the screen is currently accepting text input.
func (m *Model) InForm() bool { return m.mode == modeForm || m.mode == modeSearch }

func (m *Model) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.ThemeChangedMsg:
		m.theme = theme.ByName(msg.ThemeName)
		return m, nil
	case msgs.SparksLoadedMsg:
		m.items = displayOrdered(msg.Items)
		m.loaded = true
		m.clampCursor()
		m.ensureCursorVisible()
		return m, nil
	case tea.MouseMsg:
		return m.updateMouse(msg)
	case tea.KeyMsg:
		if m.mode == modeForm {
			return m.updateForm(msg)
		}
		if m.mode == modeSearch {
			return m.updateSearch(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

// updateMouse handles mouse wheel scrolling and click-to-select on list rows.
// The Y coordinate is content-relative (0 = top of this screen's content area),
// already adjusted by the root before forwarding.
func (m *Model) updateMouse(msg tea.MouseMsg) (screens.Screen, tea.Cmd) {
	if m.mode == modeForm {
		return m, nil // text input active — ignore mouse
	}
	visible := m.visibleItems()
	switch msg.Type {
	case tea.MouseWheelUp:
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
		}
	case tea.MouseWheelDown:
		if m.cursor+1 < len(visible) {
			m.cursor++
			m.ensureCursorVisible()
		}
	case tea.MouseLeft:
		// Layout inside the box (Padding(1,2)):
		//   row 0 → top padding
		//   row 1 → gradient header "Sparks"
		//   row 2 → blank
		//   row 3 → search bar (if active/has query) │ first item (no search)
		//   row 4 → blank after search bar           │ second item
		//   row 5 → first item (if search active)
		headerRows := 3 // padding + header + blank
		if m.mode == modeSearch || m.hasSearch() {
			headerRows += 2 // search bar + blank separator
		}
		itemRow := msg.Y - headerRows
		if itemRow >= 0 && m.offset+itemRow < len(visible) {
			m.cursor = m.offset + itemRow
			m.ensureCursorVisible()
		}
	}
	return m, nil
}

func (m *Model) updateList(key tea.KeyMsg) (screens.Screen, tea.Cmd) {
	if key.String() != "D" {
		m.deleteConfirm = false
	}
	switch key.String() {
	case "n":
		m.mode = modeForm
		m.editingID = ""
		m.input.Reset()
		m.input.Focus()
		return m, textinput.Blink
	case "e":
		sp, ok := m.selectedSpark()
		if !ok {
			return m, nil
		}
		m.mode = modeForm
		m.editingID = sp.ID
		m.input.SetValue(sp.Title)
		m.input.CursorEnd()
		m.input.Focus()
		return m, textinput.Blink
	case "p":
		sp, ok := m.selectedSpark()
		if !ok || sp.Status == domain.SparkStatusPromoted || sp.Status == domain.SparkStatusArchived {
			return m, nil
		}
		return m, m.promoteSparkCmd(sp)
	case "a":
		sp, ok := m.selectedSpark()
		if !ok {
			return m, nil
		}
		return m, m.toggleArchiveCmd(sp)
	case "h":
		m.toggleArchivedVisibility()
	case "/":
		// Disallow search when there are no sparks at all, or when a prior
		// search has already hidden everything (no point adding another filter).
		if len(m.items) == 0 {
			return m, nil
		}
		if len(m.visibleItems()) == 0 && !m.hasSearch() {
			return m, nil
		}
		m.mode = modeSearch
		m.searchInput.SetValue(m.query)
		m.searchInput.CursorEnd()
		m.searchInput.Focus()
		return m, textinput.Blink
	case "c":
		m.clearSearch()
	case "j", "down":
		if m.cursor+1 < len(m.visibleItems()) {
			m.cursor++
		}
		m.ensureCursorVisible()
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		m.ensureCursorVisible()
	case "g", "home":
		m.cursor = 0
		m.ensureCursorVisible()
	case "G", "end":
		if visible := m.visibleItems(); len(visible) > 0 {
			m.cursor = len(visible) - 1
		}
		m.ensureCursorVisible()
	case "D":
		sp, ok := m.selectedSpark()
		if !ok || sp.Status != domain.SparkStatusArchived {
			return m, nil
		}
		if m.deleteConfirm {
			m.deleteConfirm = false
			return m, m.deleteSparkCmd(sp)
		}
		m.deleteConfirm = true
		return m, nil
	}
	return m, nil
}

func (m *Model) updateSearch(key tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.mode = modeList
		m.searchInput.SetValue(m.query)
		m.searchInput.Blur()
		return m, nil
	case "enter":
		m.query = strings.TrimSpace(m.searchInput.Value())
		m.mode = modeList
		m.searchInput.Blur()
		m.clampCursor()
		m.ensureCursorVisible()
		return m, nil
	case "ctrl+u":
		m.searchInput.SetValue("")
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(key)
	return m, cmd
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

// deleteSparkCmd permanently removes an archived spark from disk.
func (m *Model) deleteSparkCmd(sp domain.Spark) tea.Cmd {
	saver := m.saver
	return func() tea.Msg {
		if saver == nil {
			return msgs.ErrorMsg{Source: "delete-spark", Err: fmt.Errorf("no storage configured")}
		}
		if err := saver.DeleteSpark(sp.ID); err != nil {
			return msgs.ErrorMsg{Source: "delete-spark", Err: err}
		}
		items, err := saver.ListSparks()
		if err != nil {
			return msgs.ErrorMsg{Source: "list-sparks", Err: err}
		}
		return msgs.SparksLoadedMsg{Items: items}
	}
}

// promoteSparkCmd creates a project from the spark, marks the spark as
// promoted, and returns a SparkPromotedMsg so the root can route to Projects.
func (m *Model) promoteSparkCmd(sp domain.Spark) tea.Cmd {
	saver := m.saver
	promoter := m.promoter
	now := m.now().UTC()
	return func() tea.Msg {
		if promoter == nil {
			return msgs.ErrorMsg{Source: "promote-spark", Err: fmt.Errorf("no project storage configured")}
		}
		projectID := domain.NewProjectID(sp.Title, now)
		project := domain.Project{
			ID:        projectID,
			Title:     sp.Title,
			Status:    domain.ProjectStatusDraft,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := promoter.SaveProject(project); err != nil {
			return msgs.ErrorMsg{Source: "promote-spark", Err: err}
		}
		sp.Status = domain.SparkStatusPromoted
		sp.PromotedProjectID = projectID
		sp.UpdatedAt = now
		if err := saver.SaveSpark(sp); err != nil {
			return msgs.ErrorMsg{Source: "promote-spark", Err: err}
		}
		sparks, err := saver.ListSparks()
		if err != nil {
			return msgs.ErrorMsg{Source: "list-sparks", Err: err}
		}
		projects, err := promoter.ListProjects()
		if err != nil {
			return msgs.ErrorMsg{Source: "list-projects", Err: err}
		}
		return msgs.SparkPromotedMsg{
			Project:  project,
			Sparks:   sparks,
			Projects: projects,
		}
	}
}

func (m *Model) View(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	box := theme.Base(m.theme).Padding(1, 2).Width(width).Height(height)
	header := theme.ApplyGradOn("Sparks", m.theme.GradientFrom, m.theme.GradientTo, m.theme.Background, true)

	if m.mode == modeForm {
		return box.Render(m.formView(width, height, header))
	}

	if !m.loaded {
		return box.Render(m.centerCard(width, height,
			header,
			theme.Fg(m.theme, m.theme.Muted).Render("Loading sparks…"),
		))
	}

	visible := m.visibleItems()
	if len(visible) == 0 {
		title := "No sparks yet."
		hint := "press n to capture your first one"
		if m.hasSearch() {
			title = "No sparks match."
			hint = "press c to clear search"
		} else if m.archivedCount() > 0 {
			title = "No active sparks."
			hint = "press h to show archived sparks"
		}
		return box.Render(m.centerCard(width, height,
			header,
			theme.Fg(m.theme, m.theme.Foreground).Render(title),
			theme.Fg(m.theme, m.theme.Subtle).Italic(true).Render(hint),
			"",
			m.archiveButton(),
		))
	}

	searchRows := 0
	if m.mode == modeSearch || m.hasSearch() {
		searchRows = 2
	}
	m.listHeight = max(1, height-6-searchRows)
	m.ensureCursorVisible()
	start := clamp(m.offset, 0, max(0, len(visible)-1))
	end := min(len(visible), start+m.listHeight)

	rows := []string{header}
	if m.mode == modeSearch || m.hasSearch() {
		rows = append(rows, "", m.searchBar())
	}
	rows = append(rows, "")
	for i, sp := range visible[start:end] {
		i += start
		rows = append(rows, m.renderRow(i, sp))
	}
	if start > 0 || end < len(visible) {
		rows = append(rows, theme.Fg(m.theme, m.theme.Subtle).
			Render(fmt.Sprintf("showing %d-%d of %d", start+1, end, len(visible))))
	}
	footerHint := fmt.Sprintf("%d active  ·  %d archived  ·  n new  ·  e edit  ·  p promote  ·  a archive  ·  D delete (archived)  ·  / search  ·  c clear  ·  ",
		m.activeCount(), m.archivedCount())
	footer := theme.Fg(m.theme, m.theme.Subtle).Render(footerHint) + m.archiveButton()
	if m.deleteConfirm {
		footer = theme.Fg(m.theme, m.theme.Danger).Render("D again to confirm delete — any other key cancels")
	}
	rows = append(rows, "", footer)

	return box.Render(strings.Join(rows, "\n"))
}

func (m *Model) renderRow(i int, sp domain.Spark) string {
	title := strings.TrimSpace(sp.Title)
	if title == "" {
		title = "(untitled)"
	}

	statusStyle := lipgloss.NewStyle().
		Background(m.theme.Background).
		Foreground(m.statusColor(sp.Status)).
		Width(13)
	titleStyle := theme.Fg(m.theme, m.theme.Foreground)
	cursor := "  "

	if i == m.cursor {
		cursor = theme.Fg(m.theme, m.theme.Accent).Bold(true).Render("▌ ")
		titleStyle = titleStyle.Foreground(m.theme.Primary).Bold(true)
	}
	if sp.Status == domain.SparkStatusArchived {
		titleStyle = titleStyle.Foreground(m.theme.Subtle).Strikethrough(true)
	}
	if sp.Status == domain.SparkStatusPromoted {
		titleStyle = titleStyle.Foreground(m.theme.Success)
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
	border := theme.SurfaceStyle(m.theme).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderFocus).
		Padding(1, 2).
		Width(min(width-8, 56))

	promptText := "Capture a spark — enter to save, esc to cancel"
	if m.editingID != "" {
		promptText = "Edit spark — enter to save, esc to cancel"
	}
	prompt := theme.Fg(m.theme, m.theme.Subtle).Render(promptText)

	card := border.Render(lipgloss.JoinVertical(lipgloss.Left,
		prompt,
		"",
		m.input.View(),
	))

	block := lipgloss.JoinVertical(lipgloss.Center, header, "", card)
	return theme.Place(m.theme, max(1, width-4), max(1, height-2), lipgloss.Center, lipgloss.Center, block)
}

func (m *Model) centerCard(width, height int, header string, lines ...string) string {
	body := append([]string{header, ""}, lines...)
	block := lipgloss.JoinVertical(lipgloss.Center, body...)
	return theme.Place(m.theme, max(1, width-4), max(1, height-2), lipgloss.Center, lipgloss.Center, block)
}

func (m *Model) visibleItems() []domain.Spark {
	out := make([]domain.Spark, 0, len(m.items))
	query := normalizeQuery(m.query)
	for _, sp := range m.items {
		if !m.showArchived && sp.Status == domain.SparkStatusArchived {
			continue
		}
		if query != "" && !sparkMatches(sp, query) {
			continue
		}
		out = append(out, sp)
	}
	return out
}

func (m *Model) selectedSpark() (domain.Spark, bool) {
	visible := m.visibleItems()
	if len(visible) == 0 || m.cursor < 0 || m.cursor >= len(visible) {
		return domain.Spark{}, false
	}
	return visible[m.cursor], true
}

func (m *Model) toggleArchivedVisibility() {
	m.showArchived = !m.showArchived
	m.clampCursor()
	if m.showArchived {
		visible := m.visibleItems()
		if len(visible) > 0 {
			m.cursor = len(visible) - 1
		}
	}
	m.ensureCursorVisible()
}

func (m *Model) clampCursor() {
	visible := m.visibleItems()
	if len(visible) == 0 {
		m.cursor = 0
		m.offset = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(visible) {
		m.cursor = len(visible) - 1
	}
}

func (m *Model) ensureCursorVisible() {
	m.clampCursor()
	visible := m.visibleItems()
	if m.listHeight <= 0 {
		m.listHeight = 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.listHeight {
		m.offset = m.cursor - m.listHeight + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
	maxOffset := max(0, len(visible)-m.listHeight)
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
}

func (m *Model) activeCount() int {
	total := 0
	for _, sp := range m.items {
		if sp.Status != domain.SparkStatusArchived {
			total++
		}
	}
	return total
}

func (m *Model) archivedCount() int {
	total := 0
	for _, sp := range m.items {
		if sp.Status == domain.SparkStatusArchived {
			total++
		}
	}
	return total
}

func (m *Model) archiveButton() string {
	if m.archivedCount() == 0 {
		return theme.Fg(m.theme, m.theme.Subtle).Render("h archived")
	}
	label := fmt.Sprintf("h show archived (%d)", m.archivedCount())
	if m.showArchived {
		label = "h hide archived"
	}
	return theme.Fg(m.theme, m.theme.Foreground).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

func (m *Model) searchBar() string {
	if m.mode == modeSearch {
		return m.searchInput.View()
	}
	query := m.query
	if query == "" {
		query = "(none)"
	}
	search := theme.Fg(m.theme, m.theme.Subtle).Render("search ") +
		theme.Fg(m.theme, m.theme.Foreground).Render(query)
	return search
}

func (m *Model) hasSearch() bool {
	return strings.TrimSpace(m.query) != ""
}

func (m *Model) clearSearch() {
	m.query = ""
	m.searchInput.Reset()
	m.clampCursor()
	m.ensureCursorVisible()
}

func sparkMatches(sp domain.Spark, query string) bool {
	fields := []string{
		sp.Title,
		sp.Description,
		string(sp.Status),
		sp.PromotedProjectID,
		strings.Join(sp.Tags, " "),
	}
	for _, field := range fields {
		if strings.Contains(normalizeQuery(field), query) {
			return true
		}
	}
	return false
}

func normalizeQuery(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func displayOrdered(items []domain.Spark) []domain.Spark {
	out := make([]domain.Spark, 0, len(items))
	archived := make([]domain.Spark, 0)
	for _, sp := range items {
		if sp.Status == domain.SparkStatusArchived {
			archived = append(archived, sp)
			continue
		}
		out = append(out, sp)
	}
	return append(out, archived...)
}

func clamp(n, low, high int) int {
	if n < low {
		return low
	}
	if n > high {
		return high
	}
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
