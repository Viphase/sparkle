package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

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

// AllProjectStatuses returns statuses in cycle order.
func AllProjectStatuses() []ProjectStatus {
	return []ProjectStatus{
		ProjectStatusDraft,
		ProjectStatusActive,
		ProjectStatusPaused,
		ProjectStatusCompleted,
		ProjectStatusArchived,
	}
}

type Project struct {
	ID             string
	Title          string
	Status         ProjectStatus
	GitHubURL      string
	TargetAudience string
	Tags           []string
	Body           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewProjectID returns a slug-based, collision-resistant project ID.
// Format: project_<slug>_<date>_<4 hex chars>.
func NewProjectID(title string, now time.Time) string {
	slug := slugifyProject(title)
	date := now.UTC().Format("20060102")
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("project_%s_%s_%06d", slug, date, now.UTC().Nanosecond()/1000)
	}
	suffix := hex.EncodeToString(b[:])
	if slug == "" {
		return fmt.Sprintf("project_%s_%s", date, suffix)
	}
	return fmt.Sprintf("project_%s_%s_%s", slug, date, suffix)
}

func slugifyProject(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			sb.WriteRune(r)
		case r == ' ' || r == '-':
			sb.WriteByte('_')
		}
	}
	result := sb.String()
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}
	return strings.Trim(result, "_")
}
