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

// TrackingLoadedMsg carries pre-computed stats and per-project event maps
// broadcast after the startup scan completes.
type TrackingLoadedMsg struct {
	// AllEvents maps project ID → its events.
	AllEvents map[string][]domain.TrackingEvent
}

// MouseToggledMsg is emitted by the settings screen when the user toggles
// mouse support. The root listens to enable or disable cell-motion tracking.
type MouseToggledMsg struct {
	Enabled bool
}

// ProjectContextMsg is emitted when a specific project should become the
// active AI guide context (e.g. just after spark promotion).
type ProjectContextMsg struct {
	Project domain.Project
}

// SkillChangedMsg is emitted by the settings screen when the user changes the
// active AI skill. The root and AI screen listen for it.
type SkillChangedMsg struct {
	Skill string // matches a domain.Skill constant; "" = none
}

// APIKeyChangedMsg is emitted by the settings screen when the user saves a new
// Anthropic API key. Root swaps the AI provider.
type APIKeyChangedMsg struct {
	Key   string
	Model string
}

// PingResultMsg is the result of a "Test connection" ping from settings.
type PingResultMsg struct {
	Err error // nil = success
}

// SkillDefsLoadedMsg carries filesystem-loaded skill definitions to the
// settings screen so the skill picker shows user-authored skills.
type SkillDefsLoadedMsg struct {
	Skills []domain.SkillDef
}
