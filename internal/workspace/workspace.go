// Package workspace resolves and bootstraps a Sparkle workspace directory.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnvHome overrides the workspace root when set. Useful for tests and for users
// who want their workspace somewhere other than $HOME/sparkle.
const EnvHome = "SPARKLE_HOME"

// DefaultDirName is the directory inside $HOME used when SPARKLE_HOME is unset.
const DefaultDirName = "sparkle"

// MetaDirName is the workspace-internal directory holding derived data
// (config, index, event log).
const MetaDirName = ".sparkle"

// Workspace is a resolved, bootstrapped workspace root.
type Workspace struct {
	Root string
}

// DefaultPath returns the path Sparkle will use for the workspace, honoring
// $SPARKLE_HOME first and falling back to $HOME/sparkle.
func DefaultPath() (string, error) {
	if v := os.Getenv(EnvHome); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home: %w", err)
	}
	return filepath.Join(home, DefaultDirName), nil
}

// Open resolves and bootstraps the workspace at root, creating the standard
// layout (sparks/, projects/, .sparkle/, .sparkle/events/) if missing.
func Open(root string) (Workspace, error) {
	if root == "" {
		return Workspace{}, fmt.Errorf("workspace root must not be empty")
	}
	for _, d := range layoutDirs(root) {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return Workspace{}, fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return Workspace{Root: root}, nil
}

func layoutDirs(root string) []string {
	return []string{
		root,
		filepath.Join(root, "sparks"),
		filepath.Join(root, "projects"),
		filepath.Join(root, MetaDirName),
		filepath.Join(root, MetaDirName, "events"),
	}
}
