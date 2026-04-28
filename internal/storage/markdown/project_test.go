package markdown

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/viphase/sparkle/internal/domain"
)

func TestSaveAndLoadProjectRoundtrip(t *testing.T) {
	store := NewStore(t.TempDir())
	want := domain.Project{
		ID:             "project_sparkle_20260428_test",
		Title:          "Sparkle",
		Status:         domain.ProjectStatusActive,
		GitHubURL:      "https://github.com/viphase/sparkle",
		TargetAudience: "developers",
		Tags:           []string{"tui", "go"},
		CreatedAt:      time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
	}
	if err := store.SaveProject(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadProject(want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != want.Title {
		t.Errorf("title: %q vs %q", got.Title, want.Title)
	}
	if got.Status != want.Status {
		t.Errorf("status: %q vs %q", got.Status, want.Status)
	}
	if got.GitHubURL != want.GitHubURL {
		t.Errorf("github_url: %q vs %q", got.GitHubURL, want.GitHubURL)
	}
	if got.TargetAudience != want.TargetAudience {
		t.Errorf("target_audience: %q vs %q", got.TargetAudience, want.TargetAudience)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("created_at: %v vs %v", got.CreatedAt, want.CreatedAt)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "tui" || got.Tags[1] != "go" {
		t.Errorf("tags: %v", got.Tags)
	}
	if got.Body == "" {
		t.Error("body should be loaded from project.md")
	}
}

func TestSaveProjectCreatesNotesMd(t *testing.T) {
	store := NewStore(t.TempDir())
	p := domain.Project{
		ID:        "project_notes_test_20260428",
		Title:     "My Project",
		Status:    domain.ProjectStatusDraft,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	notesPath := filepath.Join(store.ProjectDir(p.ID), "notes.md")
	data, err := os.ReadFile(notesPath)
	if err != nil {
		t.Fatalf("notes.md not created: %v", err)
	}
	if len(data) == 0 {
		t.Error("notes.md should not be empty")
	}
}

func TestSaveProjectPreservesNotesOnUpdate(t *testing.T) {
	store := NewStore(t.TempDir())
	p := domain.Project{
		ID:        "project_preserve_notes_test",
		Title:     "Initial",
		Status:    domain.ProjectStatusDraft,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	// Write user content to notes.md
	notesPath := filepath.Join(store.ProjectDir(p.ID), "notes.md")
	userContent := "# My notes\n\nVery important notes.\n"
	if err := os.WriteFile(notesPath, []byte(userContent), 0o644); err != nil {
		t.Fatal(err)
	}
	// Update the project title
	p.Title = "Updated"
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	// User notes must survive
	got, err := os.ReadFile(notesPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != userContent {
		t.Errorf("notes.md was overwritten: %q", string(got))
	}
}

func TestSaveProjectCreatesDefaultBody(t *testing.T) {
	store := NewStore(t.TempDir())
	p := domain.Project{
		ID:        "project_body_test",
		Title:     "Body Test",
		Status:    domain.ProjectStatusDraft,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.SaveProject(p); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(store.ProjectPath(p.ID))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, section := range []string{"# Description", "# Architecture", "# Roadmap"} {
		if !contains(content, section) {
			t.Errorf("project.md missing section %q", section)
		}
	}
}

func TestListProjectsMissingDir(t *testing.T) {
	store := NewStore(t.TempDir())
	got, err := store.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestListProjectsSortedByCreatedAtDesc(t *testing.T) {
	store := NewStore(t.TempDir())
	older := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)
	if err := store.SaveProject(domain.Project{
		ID: "project_old_test", Title: "Older", Status: domain.ProjectStatusDraft,
		CreatedAt: older, UpdatedAt: older,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProject(domain.Project{
		ID: "project_new_test", Title: "Newer", Status: domain.ProjectStatusActive,
		CreatedAt: newer, UpdatedAt: newer,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := store.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "project_new_test" || got[1].ID != "project_old_test" {
		t.Errorf("expected [new, old], got %+v", got)
	}
}

func TestSaveProjectRejectsEmptyID(t *testing.T) {
	store := NewStore(t.TempDir())
	err := store.SaveProject(domain.Project{Title: "x", Status: domain.ProjectStatusDraft})
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestSaveProjectRejectsInvalidStatus(t *testing.T) {
	store := NewStore(t.TempDir())
	err := store.SaveProject(domain.Project{ID: "project_invalid", Title: "x", Status: "bogus"})
	if err == nil {
		t.Error("expected error for invalid status")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
