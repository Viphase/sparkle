package tracker_test

import (
	"testing"
	"time"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tracker"
)

func makeTime(daysAgo int) time.Time {
	return time.Now().AddDate(0, 0, -daysAgo).Truncate(24 * time.Hour)
}

func TestCompute_empty(t *testing.T) {
	stats := tracker.Compute(nil, time.Now())
	if stats.TodayWords != 0 || stats.CurrentStreak != 0 || stats.LongestStreak != 0 {
		t.Errorf("expected zero stats for empty events, got %+v", stats)
	}
}

func TestCompute_todayWords(t *testing.T) {
	events := []domain.TrackingEvent{
		{Timestamp: time.Now(), Type: domain.EventWordsAdded, Value: 300, Source: "auto"},
		{Timestamp: time.Now(), Type: domain.EventWordsAdded, Value: 150, Source: "auto"},
	}
	stats := tracker.Compute(events, time.Now())
	if stats.TodayWords != 450 {
		t.Errorf("expected 450 today words, got %d", stats.TodayWords)
	}
}

func TestCompute_streak(t *testing.T) {
	events := []domain.TrackingEvent{
		{Timestamp: makeTime(0), Type: domain.EventWordsAdded, Value: 100, Source: "auto"},
		{Timestamp: makeTime(1), Type: domain.EventWordsAdded, Value: 100, Source: "auto"},
		{Timestamp: makeTime(2), Type: domain.EventWordsAdded, Value: 100, Source: "auto"},
	}
	stats := tracker.Compute(events, time.Now())
	if stats.CurrentStreak != 3 {
		t.Errorf("expected streak=3, got %d", stats.CurrentStreak)
	}
	if stats.LongestStreak < 3 {
		t.Errorf("expected longest streak >= 3, got %d", stats.LongestStreak)
	}
}

func TestCompute_brokenStreak(t *testing.T) {
	events := []domain.TrackingEvent{
		{Timestamp: makeTime(0), Type: domain.EventWordsAdded, Value: 100, Source: "auto"},
		// gap on day 1
		{Timestamp: makeTime(2), Type: domain.EventWordsAdded, Value: 100, Source: "auto"},
		{Timestamp: makeTime(3), Type: domain.EventWordsAdded, Value: 100, Source: "auto"},
	}
	stats := tracker.Compute(events, time.Now())
	if stats.CurrentStreak != 1 {
		t.Errorf("expected streak=1 after break, got %d", stats.CurrentStreak)
	}
	if stats.LongestStreak < 2 {
		t.Errorf("expected longest >= 2, got %d", stats.LongestStreak)
	}
}

func TestWeeklyWordsByDay(t *testing.T) {
	now := time.Now()
	events := []domain.TrackingEvent{
		{Timestamp: now, Type: domain.EventWordsAdded, Value: 200, Source: "auto"},
		{Timestamp: now.AddDate(0, 0, -30), Type: domain.EventWordsAdded, Value: 999, Source: "auto"}, // outside week
	}
	days := tracker.WeeklyWordsByDay(events, now)
	total := 0
	for _, v := range days {
		total += v
	}
	if total != 200 {
		t.Errorf("expected 200 words in week, got %d", total)
	}
}

func TestLast30DaysActivity(t *testing.T) {
	now := time.Now()
	events := []domain.TrackingEvent{
		{Timestamp: now, Type: domain.EventWordsAdded, Value: 50, Source: "auto"},
		{Timestamp: now.AddDate(0, 0, -5), Type: domain.EventFileTouched, Value: 1, Source: "auto"},
	}
	activity := tracker.Last30DaysActivity(events, now)
	if !activity[29] {
		t.Error("expected today (index 29) to be active")
	}
	if !activity[24] {
		t.Error("expected 5 days ago (index 24) to be active")
	}
	if activity[0] {
		t.Error("expected 29 days ago (index 0) to be inactive")
	}
}
