package tracker

import (
	"testing"
	"time"

	"github.com/viphase/sparkle/internal/domain"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse("2006-01-02 15:04", s)
	if err != nil {
		t.Fatalf("bad time %q: %v", s, err)
	}
	return v
}

func TestPipelineStage_DraftNoEventsIsSpark(t *testing.T) {
	now := mustTime(t, "2026-05-04 12:00")
	p := domain.Project{Status: domain.ProjectStatusDraft}
	if got := PipelineStage(p, nil, now); got != StageSpark {
		t.Fatalf("want spark, got %s", got)
	}
}

func TestPipelineStage_DraftWithStageAdvancedIsShaping(t *testing.T) {
	now := mustTime(t, "2026-05-04 12:00")
	p := domain.Project{Status: domain.ProjectStatusDraft}
	evs := []domain.TrackingEvent{{Type: domain.EventStageAdvanced, Timestamp: now.Add(-2 * 24 * time.Hour)}}
	if got := PipelineStage(p, evs, now); got != StageShaping {
		t.Fatalf("want shaping, got %s", got)
	}
}

func TestPipelineStage_ActiveWithRecentEditIsBuilding(t *testing.T) {
	now := mustTime(t, "2026-05-04 12:00")
	p := domain.Project{Status: domain.ProjectStatusActive}
	evs := []domain.TrackingEvent{{Type: domain.EventEditApproved, Timestamp: now.Add(-1 * 24 * time.Hour)}}
	if got := PipelineStage(p, evs, now); got != StageBuilding {
		t.Fatalf("want building, got %s", got)
	}
}

func TestPipelineStage_RecentMilestoneIsShipping(t *testing.T) {
	now := mustTime(t, "2026-05-04 12:00")
	p := domain.Project{Status: domain.ProjectStatusActive}
	evs := []domain.TrackingEvent{{Type: domain.EventMilestoneComplete, Timestamp: now.Add(-3 * 24 * time.Hour)}}
	if got := PipelineStage(p, evs, now); got != StageShipping {
		t.Fatalf("want shipping, got %s", got)
	}
}

func TestPipelineStage_CompletedIsDone(t *testing.T) {
	now := mustTime(t, "2026-05-04 12:00")
	p := domain.Project{Status: domain.ProjectStatusCompleted}
	if got := PipelineStage(p, nil, now); got != StageDone {
		t.Fatalf("want done, got %s", got)
	}
}

func TestProjectVelocity_NoEventsIsZero(t *testing.T) {
	now := mustTime(t, "2026-05-04 12:00")
	if v := ProjectVelocity(nil, now, 7*24*time.Hour); v != 0 {
		t.Fatalf("want 0, got %v", v)
	}
}

func TestProjectVelocity_WordsPerActiveDay(t *testing.T) {
	now := mustTime(t, "2026-05-04 12:00")
	evs := []domain.TrackingEvent{
		{Type: domain.EventWordsAdded, Value: 100, Timestamp: now.Add(-1 * 24 * time.Hour)},
		{Type: domain.EventWordsAdded, Value: 50, Timestamp: now.Add(-1 * 24 * time.Hour)}, // same day
		{Type: domain.EventWordsAdded, Value: 200, Timestamp: now.Add(-3 * 24 * time.Hour)},
	}
	got := ProjectVelocity(evs, now, 7*24*time.Hour)
	want := float64(350) / 2.0
	if got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestProjectVelocity_OutsideWindowIgnored(t *testing.T) {
	now := mustTime(t, "2026-05-04 12:00")
	evs := []domain.TrackingEvent{
		{Type: domain.EventWordsAdded, Value: 999, Timestamp: now.Add(-30 * 24 * time.Hour)},
	}
	if v := ProjectVelocity(evs, now, 7*24*time.Hour); v != 0 {
		t.Fatalf("want 0, got %v", v)
	}
}

func TestDaysSinceActive_None(t *testing.T) {
	now := mustTime(t, "2026-05-04 12:00")
	if d := DaysSinceActive(nil, now); d != -1 {
		t.Fatalf("want -1, got %d", d)
	}
}

func TestDaysSinceActive_Recent(t *testing.T) {
	now := mustTime(t, "2026-05-04 12:00")
	evs := []domain.TrackingEvent{
		{Type: domain.EventEditApproved, Timestamp: now.Add(-3*24*time.Hour - 2*time.Hour)},
	}
	if d := DaysSinceActive(evs, now); d != 3 {
		t.Fatalf("want 3, got %d", d)
	}
}
