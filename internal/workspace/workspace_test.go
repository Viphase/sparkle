package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPathHonorsEnv(t *testing.T) {
	t.Setenv(EnvHome, "/custom/path")
	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if got != "/custom/path" {
		t.Errorf("got %q", got)
	}
}

func TestDefaultPathFallsBackToHome(t *testing.T) {
	t.Setenv(EnvHome, "")
	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, DefaultDirName)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOpenCreatesLayout(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "ws")
	ws, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Root != root {
		t.Errorf("root: %q vs %q", ws.Root, root)
	}
	for _, sub := range []string{"sparks", "projects", MetaDirName, filepath.Join(MetaDirName, "events")} {
		path := filepath.Join(root, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("missing %s: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", sub)
		}
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "ws")
	if _, err := Open(root); err != nil {
		t.Fatal(err)
	}
	// Drop a file inside sparks/ and re-open — Open should not wipe it.
	canary := filepath.Join(root, "sparks", "canary.md")
	if err := os.WriteFile(canary, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(canary); err != nil {
		t.Errorf("canary file was lost after re-Open: %v", err)
	}
}

func TestOpenRejectsEmptyRoot(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Error("expected error for empty root")
	}
}
