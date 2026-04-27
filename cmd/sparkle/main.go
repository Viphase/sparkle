package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/viphase/sparkle/internal/storage/markdown"
	"github.com/viphase/sparkle/internal/tui"
	"github.com/viphase/sparkle/internal/workspace"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	root, err := workspace.DefaultPath()
	if err != nil {
		return fmt.Errorf("resolve workspace: %w", err)
	}
	ws, err := workspace.Open(root)
	if err != nil {
		return fmt.Errorf("open workspace %s: %w", root, err)
	}
	store := markdown.NewStore(ws.Root)

	p := tea.NewProgram(tui.NewRoot(ws, store), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
