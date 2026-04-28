package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/storage/markdown"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/workspace"
)

// LoadSparksCmd reads every spark in the workspace and dispatches either a
// SparksLoadedMsg on success or an ErrorMsg routed to the status bar.
func LoadSparksCmd(store *markdown.Store) tea.Cmd {
	if store == nil {
		return nil
	}
	return func() tea.Msg {
		items, err := store.ListSparks()
		if err != nil {
			return msgs.ErrorMsg{Source: "list-sparks", Err: err}
		}
		return msgs.SparksLoadedMsg{Items: items}
	}
}

// LoadProjectsCmd reads every project in the workspace and dispatches either a
// ProjectsLoadedMsg on success or an ErrorMsg routed to the status bar.
func LoadProjectsCmd(store *markdown.Store) tea.Cmd {
	if store == nil {
		return nil
	}
	return func() tea.Msg {
		items, err := store.ListProjects()
		if err != nil {
			return msgs.ErrorMsg{Source: "list-projects", Err: err}
		}
		return msgs.ProjectsLoadedMsg{Items: items}
	}
}

// LoadTrackingCmd reads all project event logs and dispatches a
// TrackingLoadedMsg. The scan also runs a startup file-touch scan but defers
// writing new events to a later iteration to keep startup non-blocking.
func LoadTrackingCmd(store *markdown.Store, ws workspace.Workspace) tea.Cmd {
	if store == nil {
		return nil
	}
	return func() tea.Msg {
		allEvents, err := store.LoadAllEvents()
		if err != nil {
			return msgs.ErrorMsg{Source: "load-tracking", Err: err}
		}
		_ = ws // workspace root available for future scan integration
		return msgs.TrackingLoadedMsg{AllEvents: allEvents}
	}
}
