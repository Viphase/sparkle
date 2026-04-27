package domain

import "time"

type ProjectStatus string

const (
	ProjectStatusDraft     ProjectStatus = "draft"
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusPaused    ProjectStatus = "paused"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusArchived  ProjectStatus = "archived"
)

func (s ProjectStatus) Valid() bool {
	switch s {
	case ProjectStatusDraft, ProjectStatusActive, ProjectStatusPaused, ProjectStatusCompleted, ProjectStatusArchived:
		return true
	}
	return false
}

type Project struct {
	ID             string
	Title          string
	Status         ProjectStatus
	GitHubURL      string
	TargetAudience string
	Tags           []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
