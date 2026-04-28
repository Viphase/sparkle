// Package msgs holds tea.Msg types shared across the root model and screens.
// Putting them in a leaf package avoids the tui ↔ screens import cycle that
// would otherwise force every screen to import the parent tui package.
package msgs

import "github.com/viphase/sparkle/internal/domain"

// ErrorMsg is the unified error envelope. Any tea.Cmd that fails should
// return one so the root model can route it to the status bar.
type ErrorMsg struct {
	Source string
	Err    error
}

// StatusMsg surfaces an informational line in the status bar.
type StatusMsg struct {
	Text string
}

// SparksLoadedMsg is broadcast by the root after a list/save/archive completes.
// Both the sparks screen and the dashboard listen for it.
type SparksLoadedMsg struct {
	Items []domain.Spark
}

// ProjectsLoadedMsg is broadcast after the project list is refreshed.
// The projects screen and dashboard listen for it.
type ProjectsLoadedMsg struct {
	Items []domain.Project
}

// SparkPromotedMsg is emitted when a spark is successfully promoted to a
// project. The root routes to the Projects tab and broadcasts both the updated
// spark list and the new project list.
type SparkPromotedMsg struct {
	Project  domain.Project
	Sparks   []domain.Spark
	Projects []domain.Project
}

// ThemeChangedMsg is emitted by the settings screen when the user switches
// theme. The root and every screen listen for it to re-style themselves.
type ThemeChangedMsg struct {
	ThemeName string
}
