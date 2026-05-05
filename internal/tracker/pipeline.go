// Pipeline & velocity stat functions for the Pulse surface (M13).
//
// PipelineStage classifies a project by where it sits on the
// spark→shaping→building→shipping arc, derived from the project's
// status and recent EventStageAdvanced events.
//
// ProjectVelocity returns words-per-active-day over the trailing window —
// a simple "is this project moving?" signal.
package tracker

import (
	"time"

	"github.com/viphase/sparkle/internal/domain"
)

// Stage is a pipeline classifier independent of ProjectStatus so the dashboard
// can group projects by where they actually are in the create→ship arc.
type Stage string

const (
	StageSpark    Stage = "spark"
	StageShaping  Stage = "shaping"
	StageBuilding Stage = "building"
	StageShipping Stage = "shipping"
	StageDone     Stage = "done"
)

// PipelineStage maps a project + its events to a Stage.
//
// Rules:
//   - status=completed/archived → done
//   - status=draft AND no EventStageAdvanced events → spark
//   - status=draft WITH stage_advanced events → shaping
//   - status=active AND no edit_approved/task_completed in last 14d → shaping
//   - status=active WITH recent activity → building
//   - status=active WITH a milestone_completed in last 30d → shipping
//   - status=paused → preserve last known stage but cap at building
//
// The function is pure: it inspects only the project's own events.
func PipelineStage(p domain.Project, events []domain.TrackingEvent, now time.Time) Stage {
	switch p.Status {
	case domain.ProjectStatusCompleted, domain.ProjectStatusArchived:
		return StageDone
	}

	// Look at recency of activity signals.
	cutoff14 := now.AddDate(0, 0, -14)
	cutoff30 := now.AddDate(0, 0, -30)
	var hasStageAdv, hasRecentEdit, hasRecentMilestone bool
	for _, ev := range events {
		switch ev.Type {
		case domain.EventStageAdvanced:
			hasStageAdv = true
		case domain.EventEditApproved, domain.EventTaskCompleted:
			if ev.Timestamp.After(cutoff14) {
				hasRecentEdit = true
			}
		case domain.EventMilestoneComplete:
			if ev.Timestamp.After(cutoff30) {
				hasRecentMilestone = true
			}
		}
	}

	if hasRecentMilestone {
		return StageShipping
	}
	if p.Status == domain.ProjectStatusDraft {
		if hasStageAdv {
			return StageShaping
		}
		return StageSpark
	}
	if p.Status == domain.ProjectStatusActive {
		if hasRecentEdit {
			return StageBuilding
		}
		return StageShaping
	}
	// Paused or unknown — show as shaping (the safe "needs attention" bucket).
	return StageShaping
}

// ProjectVelocity returns words-added-per-active-day over the trailing window.
// Returns 0 if no activity in window. window must be > 0.
//
// Active days are days with any EventWordsAdded ≥ 1. This avoids over-rewarding
// long sessions on a single day and under-rewarding short consistent ones.
func ProjectVelocity(events []domain.TrackingEvent, now time.Time, window time.Duration) float64 {
	if window <= 0 {
		return 0
	}
	cutoff := now.Add(-window)
	totalWords := 0
	activeDays := map[string]struct{}{}
	for _, ev := range events {
		if ev.Timestamp.Before(cutoff) {
			continue
		}
		if ev.Type != domain.EventWordsAdded {
			continue
		}
		totalWords += ev.Value
		activeDays[dateOf(ev.Timestamp)] = struct{}{}
	}
	if len(activeDays) == 0 {
		return 0
	}
	return float64(totalWords) / float64(len(activeDays))
}

// DaysSinceActive returns whole days between the most recent activity and now.
// Returns -1 if there is no activity at all. M16 uses this for the
// stale-project banner threshold (>7 days).
func DaysSinceActive(events []domain.TrackingEvent, now time.Time) int {
	var last time.Time
	for _, ev := range events {
		switch ev.Type {
		case domain.EventWordsAdded, domain.EventFileTouched,
			domain.EventTaskCompleted, domain.EventMilestoneComplete,
			domain.EventEditApproved, domain.EventStageAdvanced:
			if ev.Timestamp.After(last) {
				last = ev.Timestamp
			}
		}
	}
	if last.IsZero() {
		return -1
	}
	return int(now.Sub(last).Hours() / 24)
}
