package domain

import "time"

// EventType classifies a tracking event.
type EventType string

const (
	EventWordsAdded        EventType = "words_added"
	EventWordsRemoved      EventType = "words_removed"
	EventSessionMinutes    EventType = "session_minutes"
	EventFileTouched       EventType = "file_touched"
	EventTaskCompleted     EventType = "task_completed"
	EventMilestoneComplete EventType = "milestone_completed"
	EventDeadlineAdded     EventType = "deadline_added"
	EventMoodLogged        EventType = "mood_logged"
	EventNoteAdded         EventType = "note_added"
	// EventStageAdvanced fires when a project moves to the next pipeline stage
	// (e.g. spark→shaping, shaping→building). Emitted by the AI mentor on
	// confirm-advance and by manual stage changes. Used by Pulse pipeline panel.
	EventStageAdvanced EventType = "stage_advanced"
	// EventEditApproved fires when the user accepts an AI-proposed <edit> block.
	// Counted by Pulse velocity to distinguish "AI-accelerated" from cold edits.
	EventEditApproved EventType = "edit_approved"
)

// TrackingEvent is one append-only entry in a project's event log.
type TrackingEvent struct {
	Timestamp time.Time `json:"ts"`
	Type      EventType `json:"type"`
	Value     int       `json:"value"`
	Source    string    `json:"source"`
	Note      string    `json:"note,omitempty"`
}

// TrackingStats is a pre-computed view derived from a slice of TrackingEvents.
type TrackingStats struct {
	TodayWords     int
	WeekWords      int
	CurrentStreak  int
	LongestStreak  int
	ActiveDaysWeek int
	LastActive     time.Time
}
