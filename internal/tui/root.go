package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/viphase/sparkle/internal/storage/markdown"
	"github.com/viphase/sparkle/internal/tui/components/statusbar"
	"github.com/viphase/sparkle/internal/tui/components/tabs"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/screens"
	"github.com/viphase/sparkle/internal/tui/screens/ai"
	"github.com/viphase/sparkle/internal/tui/screens/dashboard"
	"github.com/viphase/sparkle/internal/tui/screens/projects"
	"github.com/viphase/sparkle/internal/tui/screens/settings"
	"github.com/viphase/sparkle/internal/tui/screens/sparks"
	"github.com/viphase/sparkle/internal/tui/screens/tracker"
	"github.com/viphase/sparkle/internal/tui/theme"
	"github.com/viphase/sparkle/internal/workspace"
)

type Root struct {
	theme   theme.Theme
	width   int
	height  int
	route   Route
	screens map[Route]screens.Screen
	status  statusbar.Model
	ws      workspace.Workspace
	store   *markdown.Store
}

// NewRoot wires the root model. ws and store may be zero/nil — handy for tests
// — in which case background loads simply don't fire.
func NewRoot(ws workspace.Workspace, store *markdown.Store) Root {
	t := theme.PastelDark()
	// store is the Saver for the sparks screen too; nil store yields a
	// read-only screen that never persists.
	var saver sparks.Saver
	if store != nil {
		saver = store
	}
	return Root{
		theme:  t,
		route:  RouteDashboard,
		status: statusbar.New(t),
		ws:     ws,
		store:  store,
		screens: map[Route]screens.Screen{
			RouteDashboard: dashboard.New(t),
			RouteSparks:    sparks.New(t, saver),
			RouteProjects:  projects.New(t),
			RouteTracker:   tracker.New(t),
			RouteAI:        ai.New(t),
			RouteSettings:  settings.New(t, ws),
		},
	}
}

func (r Root) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(r.screens)+1)
	if c := LoadSparksCmd(r.store); c != nil {
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
	}

	next, cmd := r.screens[r.route].Update(msg)
	r.screens[r.route] = next
	return r, cmd
}

// handleGlobalKey processes keys the root owns. Returns (cmd, true) if the
// root consumed the key; otherwise the active screen sees it. Inside the
// sparks form-mode the user is typing a title, so most global keys must yield
// to the screen — only ctrl+c stays.
func (r *Root) handleGlobalKey(m tea.KeyMsg) (tea.Cmd, bool) {
	if r.route == RouteSparks && r.screens[RouteSparks].(*sparks.Model).InForm() {
		if m.String() == "ctrl+c" {
			return tea.Quit, true
		}
		return nil, false
	}
	switch m.String() {
	case "q", "ctrl+c":
		return tea.Quit, true
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

func (r Root) View() string {
	labels := make([]string, 0, len(orderedRoutes))
	current := 0
	for i, rt := range orderedRoutes {
		labels = append(labels, r.screens[rt].Title())
		if rt == r.route {
			current = i
		}
	}

	tabsView := tabs.Render(r.theme, r.width, current, labels)
	statusView := r.status.View(r.width)

	contentH := r.height - lipgloss.Height(tabsView) - lipgloss.Height(statusView)
	if contentH < 1 {
		contentH = 1
	}
	body := r.screens[r.route].View(r.width, contentH)
	bodyStyled := lipgloss.NewStyle().
		Background(r.theme.Background).
		Foreground(r.theme.Foreground).
		Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, tabsView, bodyStyled, statusView)
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
