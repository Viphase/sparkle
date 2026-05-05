package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	guideai "github.com/viphase/sparkle/internal/ai"
	"github.com/viphase/sparkle/internal/config"
	"github.com/viphase/sparkle/internal/storage/markdown"
	"github.com/viphase/sparkle/internal/tui/components/logo"
	"github.com/viphase/sparkle/internal/tui/components/statusbar"
	"github.com/viphase/sparkle/internal/tui/components/tabs"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/screens/settings"
	"github.com/viphase/sparkle/internal/tui/surfaces/pulse"
	"github.com/viphase/sparkle/internal/tui/theme"
	workspacesurf "github.com/viphase/sparkle/internal/tui/surfaces/workspace"
	"github.com/viphase/sparkle/internal/workspace"
)

// M12: minimum terminal size for graceful degraded mode. No maximums.
const (
	minAppWidth  = 50
	minAppHeight = 16
)

var thinkingFrames = []string{"ꕤ"}

type animationTickMsg struct{}

// workspaceScreen adapts *workspacesurf.Model to the screens.Screen interface
// so the root can treat it like any other screen.
type workspaceScreen struct {
	m *workspacesurf.Model
}

func (ws *workspaceScreen) Init() tea.Cmd { return ws.m.Init() }
func (ws *workspaceScreen) Title() string { return ws.m.Title() }
func (ws *workspaceScreen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	next, cmd := ws.m.Update(msg)
	return &workspaceScreen{m: next}, cmd
}
func (ws *workspaceScreen) View(w, h int) string { return ws.m.View(w, h) }
func (ws *workspaceScreen) inForm() bool         { return ws.m.InForm() }
func (ws *workspaceScreen) isEditing() bool      { return ws.m.IsEditing() }

type Root struct {
	theme        theme.Theme
	width        int
	height       int
	route        Route
	frame        int
	screens      map[Route]screens.Screen
	status       statusbar.Model
	ws           workspace.Workspace
	store        *markdown.Store
	cfg          config.Config
	mouseEnabled bool
	showHelp     bool // L7: context-aware help overlay
	showSettings bool // M11: settings modal overlay (`,` toggles)
	showError    bool // M11: full error text overlay (`?` when an error is shown)
}

// NewRoot wires the root model with defaults. ws and store may be zero/nil for tests.
func NewRoot(ws workspace.Workspace, store *markdown.Store) Root {
	return NewRootWithConfig(ws, store, config.Defaults())
}

// NewRootWithConfig wires the root model with workspace preferences loaded
// before Bubble Tea starts.
func NewRootWithConfig(ws workspace.Workspace, store *markdown.Store, cfg config.Config) Root {
	t := theme.ByName(cfg.Theme)

	provider := buildProvider(cfg)
	wsModel := workspacesurf.New(t, ws.Root, provider)
	if cfg.ActiveSkill != "" {
		wsModel.SetSkill(cfg.ActiveSkill)
	}

	r := Root{
		theme:        t,
		route:        RoutePulse,
		status:       statusbar.New(t),
		ws:           ws,
		store:        store,
		cfg:          cfg,
		mouseEnabled: cfg.MouseEnabled,
		screens: map[Route]screens.Screen{
			RouteWorkspace: &workspaceScreen{m: wsModel},
			RoutePulse:     pulse.New(t),
			RouteSettings:  settings.New(t, ws, cfg),
		},
	}
	r.status = r.status.SetHint(routeHint(r.route, false))
	return r
}

// routeHint returns the surface-specific keybinding strip shown in the status
// bar. editing=true overrides the workspace hint to highlight ctrl+s.
func routeHint(rt Route, editing bool) string {
	if editing {
		return "ctrl+s  save  ·  esc  cancel  ·  ctrl+c  quit"
	}
	switch rt {
	case RouteWorkspace:
		return "j/k  nav  ·  n  new  ·  d  delete  ·  e  edit  ·  a  ask  ·  ?  help"
	case RoutePulse:
		return "j/k  scroll  ·  g/G  top/bottom  ·  tab  switch  ·  ?  help"
	case RouteSettings:
		return "j/k  nav  ·  enter  edit  ·  esc  cancel  ·  ?  help"
	}
	return "tab  switch  ·  ?  help  ·  q  quit"
}

func buildProvider(cfg config.Config) guideai.Provider {
	if key := cfg.ResolvedAPIKey(); key != "" {
		return guideai.NewAnthropicProvider(key, cfg.AIModel)
	}
	return nil // workspace uses MockProvider as fallback when nil
}

func (r Root) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(r.screens)+6)
	cmds = append(cmds, animationTickCmd())
	if r.mouseEnabled {
		cmds = append(cmds, tea.EnableMouseCellMotion)
	}
	if c := LoadSparksCmd(r.store); c != nil {
		cmds = append(cmds, c)
	}
	if c := LoadProjectsCmd(r.store); c != nil {
		cmds = append(cmds, c)
	}
	if c := LoadTrackingCmd(r.store, r.ws); c != nil {
		cmds = append(cmds, c)
	}
	if c := LoadSkillsCmd(r.store, r.ws); c != nil {
		cmds = append(cmds, c)
	}
	for _, s := range r.screens {
		if c := s.Init(); c != nil {
			cmds = append(cmds, c)
		}
	}
	return tea.Batch(cmds...)
}

func (r Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		r.width, r.height = m.Width, m.Height
	case animationTickMsg:
		r.frame++
		return r, animationTickCmd()
	case tea.KeyMsg:
		if cmd, handled := r.handleGlobalKey(m); handled {
			return r, cmd
		}
	case msgs.ErrorMsg:
		r.status = r.status.SetError(m.Source, m.Err)
		return r, nil
	case msgs.StatusMsg:
		r.status = r.status.SetInfo(m.Text)
		return r, nil
	case msgs.ThemeChangedMsg:
		newT := theme.ByName(m.ThemeName)
		r.theme = newT
		r.status = r.status.WithTheme(newT)
		return r, r.broadcastToAll(msg)
	case msgs.APIKeyChangedMsg:
		r.cfg.AnthropicAPIKey = m.Key
		r.cfg.AIModel = m.Model
		provider := "mock provider · local only"
		if m.Key != "" {
			provider = "claude · " + m.Model
		}
		r.status = r.status.SetInfo("AI provider: " + provider)
		return r, r.broadcastToAll(msg)
	case msgs.SparksLoadedMsg:
		return r, r.broadcastToAll(msg)
	case msgs.ProjectsLoadedMsg:
		return r, r.broadcastToAll(msg)
	case msgs.TrackingLoadedMsg:
		return r, r.broadcastToAll(msg)
	case msgs.SkillDefsLoadedMsg:
		return r, r.broadcastToAll(msg)
	case msgs.MouseToggledMsg:
		r.mouseEnabled = m.Enabled
		if m.Enabled {
			return r, tea.EnableMouseCellMotion
		}
		return r, tea.DisableMouse
	case msgs.SkillChangedMsg:
		skillName := m.Skill
		if skillName == "" {
			skillName = "none"
		}
		r.status = r.status.SetInfo("AI skill: " + skillName)
		return r, r.broadcastToAll(msg)
	case tea.MouseMsg:
		return r.handleMouse(m)
	case msgs.SparkPromotedMsg:
		// Stay on Workspace; route the promoted project context there.
		r.route = RouteWorkspace
		r.status = r.status.SetInfo(fmt.Sprintf("✦ %q promoted — AI guide ready", m.Project.Title))
		sparksMsg := msgs.SparksLoadedMsg{Items: m.Sparks}
		projectsMsg := msgs.ProjectsLoadedMsg{Items: m.Projects}
		ctxMsg := msgs.ProjectContextMsg{Project: m.Project}
		var cmds []tea.Cmd
		for _, toSend := range []tea.Msg{sparksMsg, projectsMsg, ctxMsg} {
			if cmd := r.broadcastToAll(toSend); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return r, tea.Batch(cmds...)
	}

	next, cmd := r.screens[r.route].Update(msg)
	r.screens[r.route] = next
	r.status = r.status.SetHint(r.currentHint())
	return r, cmd
}

// currentHint resolves the route-specific status-bar hint, accounting for the
// embedded editor when on Workspace.
func (r Root) currentHint() string {
	editing := false
	if ws, ok := r.screens[RouteWorkspace].(*workspaceScreen); ok {
		editing = ws.isEditing() && r.route == RouteWorkspace
	}
	return routeHint(r.route, editing)
}

// broadcastToAll sends msg to every screen and returns a batched cmd.
func (r *Root) broadcastToAll(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for rt, s := range r.screens {
		next, cmd := s.Update(msg)
		r.screens[rt] = next
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// handleGlobalKey processes keys the root owns.
func (r *Root) handleGlobalKey(m tea.KeyMsg) (tea.Cmd, bool) {
	if r.route == RouteWorkspace {
		if ws, ok := r.screens[RouteWorkspace].(*workspaceScreen); ok && ws.inForm() {
			if m.String() == "ctrl+c" {
				return tea.Quit, true
			}
			return nil, false
		}
	}
	// When the settings modal is open, give it the keystroke (except global
	// quit / dismiss). esc and `,` dismiss the modal.
	if r.showSettings {
		switch m.String() {
		case "ctrl+c":
			return tea.Quit, true
		case "esc", ",":
			r.showSettings = false
			return nil, true
		}
		next, cmd := r.screens[RouteSettings].Update(m)
		r.screens[RouteSettings] = next
		return cmd, true
	}
	switch m.String() {
	case "q", "ctrl+c":
		return tea.Quit, true
	case "esc":
		if r.showError {
			r.showError = false
			return nil, true
		}
		if r.showHelp {
			r.showHelp = false
			return nil, true
		}
		if r.status.HasError() {
			r.status = r.status.ClearError()
			return nil, true
		}
	case "?":
		// If a status-bar error is showing, `?` expands it to a full overlay.
		// Otherwise toggle context-aware help.
		if r.status.HasError() && !r.showError {
			r.showError = true
			r.showHelp = false
			return nil, true
		}
		if r.showError {
			r.showError = false
			return nil, true
		}
		r.showHelp = !r.showHelp
		return nil, true
	case ",":
		// Settings modal — toggleable from any surface.
		r.showSettings = true
		r.showHelp = false
		return nil, true
	case "tab":
		r.route = r.route.Next()
		r.status = r.status.SetHint(r.currentHint())
		return nil, true
	case "shift+tab":
		r.route = r.route.Prev()
		r.status = r.status.SetHint(r.currentHint())
		return nil, true
	}
	if jump, ok := numberRoute(m.String()); ok {
		r.route = jump
		r.status = r.status.SetHint(r.currentHint())
		return nil, true
	}
	return nil, false
}

// handleMouse dispatches mouse events. App fills the full terminal — no offset.
func (r Root) handleMouse(m tea.MouseMsg) (tea.Model, tea.Cmd) {
	termW, termH := r.viewport()
	if termW < minAppWidth || termH < minAppHeight {
		return r, nil
	}

	// tabs = 2 rows (content line + bottom border).
	const tabsH = 2

	if m.X < 0 || m.X >= termW || m.Y < 0 || m.Y >= termH {
		return r, nil
	}

	if m.Type == tea.MouseLeft && m.Y < tabsH {
		labels := r.tabLabels()
		current := r.currentTabIndex()
		zones := tabs.Zones(termW, current, labels)
		for i, z := range zones {
			if m.X >= z.Start && m.X < z.End {
				if i < len(orderedRoutes) {
					r.route = orderedRoutes[i]
				}
				return r, nil
			}
		}
		return r, nil
	}

	adjusted := tea.MouseMsg{
		Type:  m.Type,
		X:     m.X,
		Y:     m.Y - tabsH,
		Alt:   m.Alt,
		Ctrl:  m.Ctrl,
		Shift: m.Shift,
	}
	next, cmd := r.screens[r.route].Update(adjusted)
	r.screens[r.route] = next
	return r, cmd
}

func (r Root) tabLabels() []string {
	labels := make([]string, 0, len(orderedRoutes))
	for _, rt := range orderedRoutes {
		labels = append(labels, r.screens[rt].Title())
	}
	return labels
}

func (r Root) currentTabIndex() int {
	for i, rt := range orderedRoutes {
		if rt == r.route {
			return i
		}
	}
	return 0
}

func (r Root) View() string {
	termW, termH := r.viewport()
	if termW < minAppWidth || termH < minAppHeight {
		return r.minimumSizeView(termW, termH)
	}

	labels := make([]string, 0, len(orderedRoutes))
	current := 0
	for i, rt := range orderedRoutes {
		labels = append(labels, r.screens[rt].Title())
		if rt == r.route {
			current = i
		}
	}

	// M12: use full terminal width — no letterboxing.
	tabsView := tabs.Render(r.theme, termW, current, labels)
	statusView := r.status.View(termW)

	contentH := termH - lipgloss.Height(tabsView) - lipgloss.Height(statusView)
	if contentH < 1 {
		contentH = 1
	}
	body := r.screens[r.route].View(termW, contentH)
	bodyStyled := theme.Base(r.theme).
		Width(termW).
		Height(contentH).
		MaxHeight(contentH).
		Render(body)

	assembled := lipgloss.JoinVertical(lipgloss.Left, tabsView, bodyStyled, statusView)
	// PaintBackground re-applies the app background after ANSI resets so every
	// cell is painted — fixes "text looks like shit" background transparency.
	base := theme.PaintBackground(r.theme, termW, termH, assembled)

	// L7: help modal — when active, paint the help view on top of base.
	if r.showSettings {
		return r.renderSettingsModal(base, termW, termH)
	}
	if r.showError {
		return r.renderErrorModal(base, termW, termH)
	}
	if r.showHelp {
		return r.renderHelpView(base, termW, termH)
	}
	return base
}

// renderSettingsModal paints the settings screen as a centered modal overlay
// (~80%×70% of the terminal) so the user reaches Settings from any surface
// via `,` without losing context.
func (r Root) renderSettingsModal(base string, termW, termH int) string {
	t := r.theme
	modalW := termW * 80 / 100
	modalH := termH * 70 / 100
	if modalW < 40 {
		modalW = termW
	}
	if modalH < 12 {
		modalH = termH
	}
	// Inner area = modal minus border (1) and padding (2 horizontal cells).
	innerW := modalW - 4
	innerH := modalH - 2
	if innerW < 10 {
		innerW = 10
	}
	if innerH < 6 {
		innerH = 6
	}
	body := r.screens[RouteSettings].View(innerW, innerH)
	box := theme.Base(t).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus).
		Padding(0, 1).
		Render(body)
	return overlayBox(base, box, termW, termH)
}

// renderErrorModal renders the full status-bar error text as a centered
// overlay so the user can read errors that exceeded the one-line bar.
func (r Root) renderErrorModal(base string, termW, termH int) string {
	t := r.theme
	header := theme.Fg(t, t.Danger).Bold(true).Render("✗ error")
	body := theme.Fg(t, t.Foreground).Render(r.status.ErrorText())
	hint := theme.Fg(t, t.Subtle).Italic(true).Render("? close   ·   esc dismiss")
	maxW := termW * 80 / 100
	if maxW < 30 {
		maxW = termW - 4
	}
	wrapped := lipgloss.NewStyle().Width(maxW).Render(body)
	box := theme.Base(t).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Danger).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, header, "", wrapped, "", hint))
	return overlayBox(base, box, termW, termH)
}

// overlayBox centers `box` over `base` using the existing ANSI-aware splice
// helper so background colors in `base` show through where `box` doesn't paint.
func overlayBox(base, box string, termW, termH int) string {
	boxW := lipgloss.Width(box)
	boxH := lipgloss.Height(box)
	startX := (termW - boxW) / 2
	startY := (termH - boxH) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}
	baseLines := strings.Split(base, "\n")
	boxLines := strings.Split(box, "\n")
	for i, bl := range boxLines {
		row := startY + i
		if row >= len(baseLines) {
			break
		}
		baseLines[row] = spliceAtColumn(baseLines[row], bl, startX)
	}
	return strings.Join(baseLines, "\n")
}

// renderHelpView renders the context-aware help modal centered over base.
// It uses lipgloss.Place to position a bordered box at the screen center,
// writing the result over the (already painted) base frame.
func (r Root) renderHelpView(base string, termW, termH int) string {
	t := r.theme

	type keyRow struct{ keys, desc string }
	global := []keyRow{
		{"tab / shift+tab", "next / prev surface"},
		{"1 / 2 / 3", "jump to surface directly"},
		{"?", "toggle this help"},
		{"esc", "dismiss error"},
		{"q / ctrl+c", "quit"},
	}
	surfaceKeys := map[Route][]keyRow{
		RouteWorkspace: {
			{"j / k", "navigate rail"},
			{"enter", "focus AI input"},
			{"n", "new spark"},
			{"d", "delete item (y confirm)"},
			{"a", "ask AI"},
			{"e", "edit item body"},
			{"ctrl+s", "save edit (in editor)"},
			{"esc", "cancel / close panel"},
			{"J / K", "scroll detail"},
		},
		RoutePulse: {
			{"j / k", "scroll down / up"},
			{"g / G", "top / bottom"},
		},
		RouteSettings: {
			{"j / k", "navigate rows"},
			{"← → / h l", "cycle values"},
			{"enter", "edit / test"},
			{"esc", "cancel edit"},
		},
	}

	renderKV := func(rows []keyRow) []string {
		lines := make([]string, 0, len(rows))
		for _, r := range rows {
			k := theme.Fg(t, t.Primary).Width(18).Render(r.keys)
			d := theme.Fg(t, t.Foreground).Render(r.desc)
			lines = append(lines, " "+k+d)
		}
		return lines
	}

	header := theme.ApplyGradOn("ꕤ  Key Reference", t.GradientFrom, t.GradientTo, t.Background, true)
	sectionHdr := func(s string) string {
		return theme.Fg(t, t.Accent).Bold(true).Render("── " + s)
	}

	routeLabel := map[Route]string{
		RouteWorkspace: "WORKSPACE",
		RoutePulse:     "PULSE",
		RouteSettings:  "SETTINGS",
	}

	// No blank line between header and first section — keeps height compact.
	parts := []string{header, sectionHdr("GLOBAL")}
	parts = append(parts, renderKV(global)...)
	if label, ok := routeLabel[r.route]; ok {
		parts = append(parts, sectionHdr(label))
		parts = append(parts, renderKV(surfaceKeys[r.route])...)
	}
	parts = append(parts, theme.Fg(t, t.Subtle).Italic(true).Render(" ? to close"))

	box := theme.Base(t).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocus).
		Padding(0, 1).
		Render(strings.Join(parts, "\n"))

	// Center the box. Use lipgloss.Place over a blank canvas to get clean
	// positioning, then splice onto base using ANSI-aware line overlay.
	boxW := lipgloss.Width(box)
	boxH := lipgloss.Height(box)
	startX := (termW - boxW) / 2
	startY := (termH - boxH) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	baseLines := strings.Split(base, "\n")
	boxLines := strings.Split(box, "\n")
	for i, bl := range boxLines {
		row := startY + i
		if row >= len(baseLines) {
			break
		}
		baseLines[row] = spliceAtColumn(baseLines[row], bl, startX)
	}
	return strings.Join(baseLines, "\n")
}

// spliceAtColumn replaces visual columns [x, x+width(overlay)] in base with
// overlay. Uses ANSI-aware truncation so escape codes in base are respected.
func spliceAtColumn(base, overlay string, x int) string {
	left := ansi.Truncate(base, x, "")
	// Pad left to exactly x columns in case base is shorter.
	leftW := lipgloss.Width(left)
	if leftW < x {
		left += strings.Repeat(" ", x-leftW)
	}
	return left + overlay
}

func numberRoute(s string) (Route, bool) {
	if len(s) != 1 {
		return 0, false
	}
	n := int(s[0] - '0')
	if n < 1 || n > len(orderedRoutes) {
		return 0, false
	}
	return orderedRoutes[n-1], true
}

func animationTickCmd() tea.Cmd {
	return tea.Tick(350*time.Millisecond, func(time.Time) tea.Msg {
		return animationTickMsg{}
	})
}

func (r Root) thinkingGlyph() string {
	if len(thinkingFrames) == 0 {
		return ""
	}
	return thinkingFrames[r.frame%len(thinkingFrames)]
}

func (r Root) viewport() (int, int) {
	w, h := r.width, r.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return w, h
}

func (r Root) minimumSizeView(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	title := logo.Compact(r.theme)
	copy := theme.Fg(r.theme, r.theme.Muted).Render(
		fmt.Sprintf("minimum %dx%d · current %dx%d", minAppWidth, minAppHeight, width, height),
	)
	block := lipgloss.JoinVertical(lipgloss.Center, title, "", copy)
	placed := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, block,
		lipgloss.WithWhitespaceBackground(r.theme.Background))
	return theme.PaintBackground(r.theme, width, height, placed)
}
