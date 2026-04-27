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
