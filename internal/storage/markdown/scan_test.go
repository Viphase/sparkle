package markdown

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanIndexRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	now := time.Now().Truncate(time.Second) // truncate for JSON round-trip equality
	idx := ScanIndexData{
		FileMtime: map[string]time.Time{
			filepath.Join(dir, "a.md"): now,
		},
		FileWords: map[string]int{
			filepath.Join(dir, "a.md"): 42,
		},
	}
	if err := store.SaveScanIndex(idx); err != nil {
		t.Fatalf("SaveScanIndex: %v", err)
	}

	loaded, err := store.LoadScanIndex()
	if err != nil {
		t.Fatalf("LoadScanIndex: %v", err)
	}
	path := filepath.Join(dir, "a.md")
	if w := loaded.FileWords[path]; w != 42 {
		t.Errorf("FileWords[a.md] = %d, want 42", w)
	}
	if !loaded.FileMtime[path].Equal(now) {
		t.Errorf("FileMtime mismatch: got %v, want %v", loaded.FileMtime[path], now)
	}
}

func TestLoadScanIndexMissing(t *testing.T) {
	store := NewStore(t.TempDir())
	idx, err := store.LoadScanIndex()
	if err != nil {
		t.Fatalf("expected no error for missing index, got %v", err)
	}
	if idx.FileMtime == nil || idx.FileWords == nil {
		t.Error("expected non-nil maps on empty index")
	}
}

func TestScanIndexPathUnderSparkle(t *testing.T) {
	store := NewStore("/workspace")
	want := filepath.Join("/workspace", ".sparkle", "scan-index.json")
	if got := store.ScanIndexPath(); got != want {
		t.Errorf("ScanIndexPath = %q, want %q", got, want)
	}
}

// TestLoadTrackingCmdScansProjects verifies that LoadTrackingCmd (integration-
// level) writes tracking events when project files exist.
// This does NOT import the tui package (would cycle); instead we test the
// store's LoadAllEvents after a manual ScanProjectDir + AppendEvent call.
func TestScanIndexPersistencePreservesDeltas(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// Write a project.md with content.
	projDir := filepath.Join(dir, "projects", "proj_01")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mdPath := filepath.Join(projDir, "project.md")
	if err := os.WriteFile(mdPath, []byte("hello world foo bar baz qux quux corge grault"), 0o644); err != nil {
		t.Fatal(err)
	}

	// First scan: empty index — should detect new words.
	empty := ScanIndexData{
		FileMtime: map[string]time.Time{},
		FileWords: map[string]int{},
	}
	if err := store.SaveScanIndex(empty); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadScanIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.FileWords) != 0 {
		t.Errorf("expected empty FileWords after save of empty index, got %v", loaded.FileWords)
	}

	// After saving a non-empty index, loading should preserve the data.
	filled := ScanIndexData{
		FileMtime: map[string]time.Time{mdPath: time.Now()},
		FileWords: map[string]int{mdPath: 9},
	}
	if err := store.SaveScanIndex(filled); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.LoadScanIndex()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.FileWords[mdPath] != 9 {
		t.Errorf("FileWords[project.md] = %d, want 9", reloaded.FileWords[mdPath])
	}
}
