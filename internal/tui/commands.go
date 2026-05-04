package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/storage/markdown"
	"github.com/viphase/sparkle/internal/tracker"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/workspace"
)

// LoadSkillsCmd reads .sparkle/skills/*.md and dispatches SkillDefsLoadedMsg.
func LoadSkillsCmd(store *markdown.Store, ws workspace.Workspace) tea.Cmd {
	if store == nil || ws.Root == "" {
		return nil
	}
	root := ws.Root
	return func() tea.Msg {
		skills, err := markdown.LoadSkills(root)
		if err != nil {
			return msgs.ErrorMsg{Source: "load-skills", Err: err}
		}
		return msgs.SkillDefsLoadedMsg{Skills: skills}
	}
}

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

// LoadTrackingCmd reads all project event logs, runs ScanProjectDir for every
// project directory to detect word-count changes since last launch, appends
// any new events, and dispatches a TrackingLoadedMsg.
// L1: integrates tracker.ScanProjectDir with scan-index persistence.
func LoadTrackingCmd(store *markdown.Store, ws workspace.Workspace) tea.Cmd {
	if store == nil {
		return nil
	}
	return func() tea.Msg {
		// 1. Load the persisted scan index (delta state from last run).
		rawIdx, _ := store.LoadScanIndex()
		idx := tracker.ScanIndex{
			FileMtime: rawIdx.FileMtime,
			FileWords: rawIdx.FileWords,
		}
		if idx.FileMtime == nil {
			idx.FileMtime = make(map[string]time.Time)
		}
		if idx.FileWords == nil {
			idx.FileWords = make(map[string]int)
		}

		// 2. Scan every project directory for file changes.
		projects, _ := store.ListProjects()
		now := time.Now()
		for _, p := range projects {
			projectDir := store.ProjectDir(p.ID)
			result := tracker.ScanProjectDir(p.ID, projectDir, idx, 10, 300, now)
			// Merge the updated index.
			for k, v := range result.Index.FileMtime {
				idx.FileMtime[k] = v
			}
			for k, v := range result.Index.FileWords {
				idx.FileWords[k] = v
			}
			// Append any new events to disk.
			for _, ev := range result.Events {
				_ = store.AppendEvent(p.ID, ev)
			}
		}

		// 3. Persist the updated index so the next run gets correct deltas.
		_ = store.SaveScanIndex(markdown.ScanIndexData{
			FileMtime: idx.FileMtime,
			FileWords: idx.FileWords,
		})

		// 4. Load all events (now including the newly appended ones).
		allEvents, err := store.LoadAllEvents()
		if err != nil {
			return msgs.ErrorMsg{Source: "load-tracking", Err: err}
		}
		return msgs.TrackingLoadedMsg{AllEvents: allEvents}
	}
}
