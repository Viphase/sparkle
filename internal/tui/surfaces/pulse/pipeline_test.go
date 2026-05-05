package pulse

import (
	"strings"
	"testing"
	"time"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/theme"
)

// pipelinePanel must answer the fourth Pulse question:
// "where is each active project in its pipeline?"
func TestPulse_PipelinePanelShowsStageAndVelocity(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.Local)
	m := &Model{
		theme: theme.PastelDark(),
		now:   func() time.Time { return now },
	}

	projects := []domain.Project{
		{ID: "p_one", Title: "Sparkle", Status: domain.ProjectStatusActive},
		{ID: "p_two", Title: "Notes", Status: domain.ProjectStatusDraft},
	}
	events := map[string][]domain.TrackingEvent{
		"p_one": {
			{Type: domain.EventEditApproved, Timestamp: now.Add(-1 * 24 * time.Hour)},
			{Type: domain.EventWordsAdded, Value: 600, Timestamp: now.Add(-2 * 24 * time.Hour)},
		},
	}

	next, _ := m.Update(msgs.ProjectsLoadedMsg{Items: projects})
	m = next.(*Model)
	next, _ = m.Update(msgs.TrackingLoadedMsg{AllEvents: events})
	m = next.(*Model)

	got := m.pipelinePanel(now, 80)
	if !strings.Contains(got, "Sparkle") {
		t.Fatalf("pipeline missing project title 'Sparkle':\n%s", got)
	}
	if !strings.Contains(got, "building") {
		t.Fatalf("expected 'building' stage for active project with recent edit:\n%s", got)
	}
	if !strings.Contains(got, "spark") {
		t.Fatalf("expected 'spark' stage for fresh draft project:\n%s", got)
	}
	if !strings.Contains(got, "active projects") {
		t.Fatalf("missing panel caption:\n%s", got)
	}
}

func TestPulse_PipelinePanelEmptyState(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.Local)
	m := &Model{theme: theme.PastelDark(), now: func() time.Time { return now }}
	got := m.pipelinePanel(now, 60)
	if !strings.Contains(got, "no projects yet") {
		t.Fatalf("expected empty-state hint, got:\n%s", got)
	}
}
