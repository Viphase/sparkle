package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/storage/markdown"
	"github.com/viphase/sparkle/internal/tui/msgs"
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
