package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/viphase/sparkle/internal/domain"
)

func TestSaveAndLoadSparkRoundtrip(t *testing.T) {
	store := NewStore(t.TempDir())
	want := domain.Spark{
		ID:          "spark_test_1",
		Title:       "Test spark",
		Description: "A description here.\nMultiple lines.",
		Status:      domain.SparkStatusNew,
		Tags:        []string{"a", "b"},
		CreatedAt:   time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC),
	}
	if err := store.SaveSpark(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadSpark(want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != want.Title {
		t.Errorf("title: %q vs %q", got.Title, want.Title)
	}
	if got.Description != want.Description {
		t.Errorf("description:\nwant %q\ngot  %q", want.Description, got.Description)
	}
	if got.Status != want.Status {
		t.Errorf("status: %q vs %q", got.Status, want.Status)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("created_at: %v vs %v", got.CreatedAt, want.CreatedAt)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "a" || got.Tags[1] != "b" {
		t.Errorf("tags: %v", got.Tags)
	}
}

func TestSaveSparkPreservesUnknownFrontmatter(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := os.MkdirAll(store.SparksDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `---
schema_version: 1
id: spark_x
title: Initial
status: new
custom_field: keep-me
external_id: 42
created_at: "2026-04-27T10:00:00Z"
updated_at: "2026-04-27T10:00:00Z"
---
Initial body.
`
	path := store.SparkPath("spark_x")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	sp, err := store.LoadSpark("spark_x")
	if err != nil {
		t.Fatal(err)
	}
	sp.Title = "Updated"
	if err := store.SaveSpark(sp); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)
	if !strings.Contains(out, "custom_field: keep-me") {
		t.Errorf("custom_field lost; got:\n%s", out)
	}
	if !strings.Contains(out, "external_id: 42") {
		t.Errorf("external_id lost; got:\n%s", out)
	}
	if !strings.Contains(out, "title: Updated") {
		t.Errorf("title not updated; got:\n%s", out)
	}
}

func TestListSparksIgnoresNonMarkdownAndDirs(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := os.MkdirAll(store.SparksDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSpark(domain.Spark{
		ID:        "a",
		Title:     "A",
		Status:    domain.SparkStatusNew,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.SparksDir(), "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(store.SparksDir(), "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := store.ListSparks()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d sparks, want 1: %+v", len(got), got)
	}
	if got[0].ID != "a" {
		t.Errorf("wrong spark: %+v", got[0])
	}
}

func TestListSparksMissingDir(t *testing.T) {
	store := NewStore(t.TempDir())
	got, err := store.ListSparks()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestListSparksSortedByCreatedAtDesc(t *testing.T) {
	store := NewStore(t.TempDir())
	older := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	if err := store.SaveSpark(domain.Spark{ID: "old", Title: "older", Status: domain.SparkStatusNew, CreatedAt: older, UpdatedAt: older}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSpark(domain.Spark{ID: "new", Title: "newer", Status: domain.SparkStatusNew, CreatedAt: newer, UpdatedAt: newer}); err != nil {
		t.Fatal(err)
	}
	got, err := store.ListSparks()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "new" || got[1].ID != "old" {
		t.Errorf("expected [new, old], got %+v", got)
	}
}

func TestLoadSparkMissing(t *testing.T) {
	store := NewStore(t.TempDir())
	_, err := store.LoadSpark("nope")
	if err == nil {
		t.Error("expected error loading missing spark")
	}
}

func TestSaveSparkRejectsEmptyID(t *testing.T) {
	store := NewStore(t.TempDir())
	err := store.SaveSpark(domain.Spark{Title: "x", Status: domain.SparkStatusNew})
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestSaveSparkRejectsInvalidStatus(t *testing.T) {
	store := NewStore(t.TempDir())
	err := store.SaveSpark(domain.Spark{ID: "a", Title: "x", Status: "bogus"})
	if err == nil {
		t.Error("expected error for invalid status")
	}
}
