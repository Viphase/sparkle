package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viphase/sparkle/internal/config"
)

func TestCommandFromArgsAcceptsSampleDataBeforeOrAfterFlags(t *testing.T) {
	cmd, args := commandFromArgs([]string{"sample-data", "--workspace", "/tmp/ws"})
	if cmd != "sample-data" || strings.Join(args, " ") != "--workspace /tmp/ws" {
		t.Fatalf("cmd=%q args=%v", cmd, args)
	}

	cmd, args = commandFromArgs([]string{"--workspace", "/tmp/ws", "sample-data"})
	if cmd != "sample-data" || strings.Join(args, " ") != "--workspace /tmp/ws" {
		t.Fatalf("cmd=%q args=%v", cmd, args)
	}
}

func TestWorkspacePathFlagIsAbsolute(t *testing.T) {
	root := filepath.Join("testdata", "workspace")
	got, err := workspacePath([]string{"--workspace", root})
	if err != nil {
		t.Fatalf("workspacePath returned error: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("workspace path should be absolute: %q", got)
	}
	if !strings.HasSuffix(got, filepath.Join("testdata", "workspace")) {
		t.Fatalf("workspace path=%q, want suffix %q", got, root)
	}
}

func TestCreateSampleDataIsIdempotent(t *testing.T) {
	const wantSparks = 3
	const wantProjects = 2
	const wantTotal = wantSparks + wantProjects

	root := t.TempDir()
	result, err := createSampleData(root)
	if err != nil {
		t.Fatalf("createSampleData returned error: %v", err)
	}
	if result.Created != wantTotal || result.Skipped != 0 {
		t.Fatalf("result=%+v, want %d created", result, wantTotal)
	}
	if _, err := os.Stat(config.Path(root)); err != nil {
		t.Fatalf("config not created: %v", err)
	}
	sparkEntries, err := os.ReadDir(filepath.Join(root, "sparks"))
	if err != nil {
		t.Fatalf("read sparks dir: %v", err)
	}
	if len(sparkEntries) != wantSparks {
		t.Fatalf("sparks count=%d, want %d", len(sparkEntries), wantSparks)
	}
	projectEntries, err := os.ReadDir(filepath.Join(root, "projects"))
	if err != nil {
		t.Fatalf("read projects dir: %v", err)
	}
	if len(projectEntries) != wantProjects {
		t.Fatalf("projects count=%d, want %d", len(projectEntries), wantProjects)
	}

	result, err = createSampleData(root)
	if err != nil {
		t.Fatalf("second createSampleData returned error: %v", err)
	}
	if result.Created != 0 || result.Skipped != wantTotal {
		t.Fatalf("second result=%+v, want %d skipped", result, wantTotal)
	}
}

func TestRunSampleDataPrintsSummary(t *testing.T) {
	var out bytes.Buffer
	root := t.TempDir()
	if err := run([]string{"sample-data", "--workspace", root}, &out); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "sample-data: created 5, skipped 0") {
		t.Fatalf("unexpected output: %q", got)
	}
	if !strings.Contains(got, root) {
		t.Fatalf("output should include workspace: %q", got)
	}
}
