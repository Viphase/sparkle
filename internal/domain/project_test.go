package domain

import "testing"

func TestProjectStatusValid(t *testing.T) {
	valid := []ProjectStatus{
		ProjectStatusDraft,
		ProjectStatusActive,
		ProjectStatusPaused,
		ProjectStatusCompleted,
		ProjectStatusArchived,
	}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if ProjectStatus("bogus").Valid() {
		t.Error("expected unknown status to be invalid")
	}
}
