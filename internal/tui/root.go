package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	guideai "github.com/viphase/sparkle/internal/ai"
	"github.com/viphase/sparkle/internal/config"
	"github.com/viphase/sparkle/internal/storage/markdown"
	"github.com/viphase/sparkle/internal/tui/components/logo"
	"github.com/viphase/sparkle/internal/tui/components/statusbar"
	"github.com/viphase/sparkle/internal/tui/components/tabs"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/screens"
	screenai "github.com/viphase/sparkle/internal/tui/screens/ai"
	"github.com/viphase/sparkle/internal/tui/screens/dashboard"
	"github.com/viphase/sparkle/internal/tui/screens/projects"
	"github.com/viphase/sparkle/internal/tui/screens/settings"
	"github.com/viphase/sparkle/internal/tui/screens/sparks"
	"github.com/viphase/sparkle/internal/tui/theme"
	"github.com/viphase/sparkle/internal/workspace"
)

const (
	minAppWidth  = 72
	minAppHeight = 20
	maxAppWidth  = 118
	maxAppHeight = 36
)

var thinkingFrames = []string{"ꕤ"}

type animationTickMsg struct{}

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
	mouseEnabled bool
}

// NewRoot wires the root model. ws and store may be zero/nil — handy for tests
// — in which case background loads simply don't fire.
func NewRoot(ws workspace.Workspace, store *markdown.Store) Root {
	return NewRootWithConfig(ws, store, config.Defaults())
}

// NewRootWithConfig wires the root model with workspace preferences loaded
// before Bubble Tea starts.
func NewRootWithConfig(ws workspace.Workspace, store *markdown.Store, cfg config.Config) Root {
	t := theme.ByName(cfg.Theme)

	// Saver and Promoter for the sparks screen — both satisfied by *markdown.Store.
	var saver sparks.Saver
	var promoter sparks.Promoter
	if store != nil {
		saver = store
		promoter = store
	}

	// Loader for the projects screen — also satisfied by *markdown.Store.
	var projectLoader projects.Loader
	if store != nil {
		projectLoader = store
	}

	// Select AI provider: use real Anthropic provider when an API key is set,
	// otherwise fall back to the local mock.
	var aiScreen screens.Screen
	if key := cfg.ResolvedAPIKey(); key != "" {
		realProvider := guideai.NewAnthropicProvider(key, cfg.AIModel)
		aiScreen = screenai.NewWithWorkDir(t, ws.Root, realProvider)
	} else {
		aiScreen = screenai.NewWithWorkDir(t, ws.Root)
	}

	return Root{
		theme:        t,
		route:        RouteDashboard,
		status:       statusbar.New(t),
		ws:           ws,
		store:        store,
		mouseEnabled: cfg.MouseEnabled,
		screens: map[Route]screens.Screen{
			RouteDashboard: dashboard.New(t),
			RouteSparks:    sparks.New(t, saver, promoter),
			RouteProjects:  projects.New(t, projectLoader),
			RouteAI:        aiScreen,
			RouteSettings:  settings.New(t, ws, cfg),
		},
	}
}

func (r Root) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(r.screens)+5)
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
		var cmds []tea.Cmd
		for rt, s := range r.screens {
			next, cmd := s.Update(msg)
			r.screens[rt] = next
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return r, tea.Batch(cmds...)
	case msgs.SparksLoadedMsg:
		// Broadcast to every screen — sparks screen needs the list, dashboard
		// needs the count.
		var cmds []tea.Cmd
		for rt, s := range r.screens {
			next, cmd := s.Update(msg)
			r.screens[rt] = next
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return r, tea.Batch(cmds...)
	case msgs.ProjectsLoadedMsg:
		var cmds []tea.Cmd
		for rt, s := range r.screens {
			next, cmd := s.Update(msg)
			r.screens[rt] = next
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return r, tea.Batch(cmds...)
	case msgs.TrackingLoadedMsg:
		var cmds []tea.Cmd
		for rt, s := range r.screens {
			next, cmd := s.Update(msg)
			r.screens[rt] = next
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return r, tea.Batch(cmds...)
	case msgs.MouseToggledMsg:
		r.mouseEnabled = m.Enabled
		if m.Enabled {
			return r, tea.EnableMouseCellMotion
		}
		return r, tea.DisableMouse
	case tea.MouseMsg:
		return r.handleMouse(m)
	case msgs.SparkPromotedMsg:
		// Route to projects, broadcast updated sparks + projects, show status.
		r.route = RouteProjects
		r.status = r.status.SetInfo(fmt.Sprintf("✦ %q promoted to project", m.Project.Title))
		sparksMsg := msgs.SparksLoadedMsg{Items: m.Sparks}
		projectsMsg := msgs.ProjectsLoadedMsg{Items: m.Projects}
		var cmds []tea.Cmd
		for rt, s := range r.screens {
			next, cmd := s.Update(sparksMsg)
			r.screens[rt] = next
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		for rt, s := range r.screens {
			next, cmd := s.Update(projectsMsg)
			r.screens[rt] = next
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return r, tea.Batch(cmds...)
	}

	next, cmd := r.screens[r.route].Update(msg)
	r.screens[r.route] = next
	return r, cmd
}

// handleGlobalKey processes keys the root owns. Returns (cmd, true) if the
// root consumed the key; otherwise the active screen sees it.
func (r *Root) handleGlobalKey(m tea.KeyMsg) (tea.Cmd, bool) {
	// Let the sparks screen handle all keys when in text-input mode.
	if r.route == RouteSparks {
		if sm, ok := r.screens[RouteSparks].(*sparks.Model); ok && sm.InForm() {
			if m.String() == "ctrl+c" {
				return tea.Quit, true
			}
			return nil, false
		}
	}
	// Let the projects screen handle all keys when in text-input mode.
	if r.route == RouteProjects {
		if pm, ok := r.screens[RouteProjects].(*projects.Model); ok && pm.InForm() {
			if m.String() == "ctrl+c" {
				return tea.Quit, true
			}
			return nil, false
		}
	}
	// Let the AI screen receive text keys while its chat input is focused.
	if r.route == RouteAI {
		if am, ok := r.screens[RouteAI].(*screenai.Model); ok && am.InForm() {
			if m.String() == "ctrl+c" {
				return tea.Quit, true
			}
			return nil, false
		}
	}
	switch m.String() {
	case "q", "ctrl+c":
		return tea.Quit, true
	case "?":
		r.status = r.status.ToggleHelp()
		return nil, true
	case "tab":
		r.route = r.route.Next()
		return nil, true
	case "shift+tab":
		r.route = r.route.Prev()
		return nil, true
	}
	if jump, ok := numberRoute(m.String()); ok {
		r.route = jump
		return nil, true
	}
	return nil, false
}

// handleMouse dispatches mouse events.
//   - Clicks in the tab bar (top 2 rows of the app) switch tabs via hit-testing.
//   - Clicks and wheel events in the content area are forwarded to the active
//     screen with Y adjusted to be content-relative (0 = first content row).
//   - Clicks outside the app widget are ignored.
func (r Root) handleMouse(m tea.MouseMsg) (tea.Model, tea.Cmd) {
	termW, termH := r.viewport()
	if termW < minAppWidth || termH < minAppHeight {
		return r, nil
	}
	appW := clamp(termW, minAppWidth, maxAppWidth)
	appH := clamp(termH, minAppHeight, maxAppHeight)
	appX := (termW - appW) / 2
	appY := (termH - appH) / 2

	// tabs = 2 rows (content line + bottom border).
	// statusbar = 2 rows (top border + content line).
	const tabsH = 2

	relX := m.X - appX
	relY := m.Y - appY

	// Ignore events outside the app widget.
	if relX < 0 || relX >= appW || relY < 0 || relY >= appH {
		return r, nil
	}

	if m.Type == tea.MouseLeft && relY < tabsH {
		// ── Tab bar click ──────────────────────────────────────────
		labels := r.tabLabels()
		current := r.currentTabIndex()
		zones := tabs.Zones(appW, current, labels)
		for i, z := range zones {
			if relX >= z.Start && relX < z.End {
				if i < len(orderedRoutes) {
					r.route = orderedRoutes[i]
				}
				return r, nil
			}
		}
		return r, nil
	}

	// ── Content area event ─────────────────────────────────────────
	// Forward to the active screen with Y made content-relative.
	adjusted := tea.MouseMsg{
		Type:  m.Type,
		X:     relX,
		Y:     relY - tabsH,
		Alt:   m.Alt,
		Ctrl:  m.Ctrl,
		Shift: m.Shift,
	}
	next, cmd := r.screens[r.route].Update(adjusted)
	r.screens[r.route] = next
	return r, cmd
}

// tabLabels builds the ordered label slice used for tab rendering and zone
// computation, matching exactly what View produces.
func (r Root) tabLabels() []string {
	labels := make([]string, 0, len(orderedRoutes))
	for _, rt := range orderedRoutes {
		label := r.screens[rt].Title()
		if rt == RouteAI {
			label = r.thinkingGlyph() + " " + label
		}
		labels = append(labels, label)
	}
	return labels
}

// currentTabIndex returns the 0-based index of the active route in orderedRoutes.
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

	appW := clamp(termW, minAppWidth, maxAppWidth)
	appH := clamp(termH, minAppHeight, maxAppHeight)

	labels := make([]string, 0, len(orderedRoutes))
	current := 0
	for i, rt := range orderedRoutes {
		label := r.screens[rt].Title()
		if rt == RouteAI {
			label = r.thinkingGlyph() + " " + label
		}
		labels = append(labels, label)
		if rt == r.route {
			current = i
		}
	}

	tabsView := tabs.Render(r.theme, appW, current, labels)
	statusView := r.status.View(appW)

	contentH := appH - lipgloss.Height(tabsView) - lipgloss.Height(statusView)
	if contentH < 1 {
		contentH = 1
	}
	body := r.screens[r.route].View(appW, contentH)
	bodyStyled := theme.Base(r.theme).
		Width(appW).
		Height(contentH).
		MaxHeight(contentH).
		Render(body)

	app := lipgloss.JoinVertical(lipgloss.Left, tabsView, bodyStyled, statusView)
	app = theme.Base(r.theme).Width(appW).Height(appH).MaxHeight(appH).Render(app)

	placed := lipgloss.Place(termW, termH, lipgloss.Center, lipgloss.Center, app,
		lipgloss.WithWhitespaceBackground(r.theme.Background))
	return theme.PaintBackground(r.theme, termW, termH, placed)
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

func clamp(n, low, high int) int {
	if n < low {
		return low
	}
	if n > high {
		return high
	}
	return n
}
