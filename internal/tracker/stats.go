// Package tracker provides pure-function statistics derived from TrackingEvent slices.
package tracker

import (
	"time"

	"github.com/viphase/sparkle/internal/domain"
)

// Compute derives a TrackingStats snapshot from a slice of events.
// now is the reference time (typically time.Now()).
func Compute(events []domain.TrackingEvent, now time.Time) domain.TrackingStats {
	today := dateOf(now)
	weekStart := startOfWeek(now)

	var stats domain.TrackingStats
	activeDays := map[string]bool{}

	for _, ev := range events {
		d := dateOf(ev.Timestamp)
		switch ev.Type {
		case domain.EventWordsAdded:
			activeDays[d] = true
			if d == today {
				stats.TodayWords += ev.Value
			}
			if !ev.Timestamp.Before(weekStart) {
				stats.WeekWords += ev.Value
			}
		case domain.EventWordsRemoved:
			// word removals don't count as additions, but do mark the day active
			activeDays[d] = true
		case domain.EventFileTouched, domain.EventSessionMinutes,
			domain.EventTaskCompleted, domain.EventMilestoneComplete:
			activeDays[d] = true
		}
		if ev.Timestamp.After(stats.LastActive) {
			stats.LastActive = ev.Timestamp
		}
	}

	stats.ActiveDaysWeek = countActiveDaysInRange(activeDays, weekStart, now)
	stats.CurrentStreak, stats.LongestStreak = computeStreaks(activeDays, now)
	return stats
}

// WeeklyWordsByDay returns 7 ints (Mon–Sun) of words added in the current week.
func WeeklyWordsByDay(events []domain.TrackingEvent, now time.Time) [7]int {
	weekStart := startOfWeek(now)
	var days [7]int
	for _, ev := range events {
		if ev.Type != domain.EventWordsAdded {
			continue
		}
		if ev.Timestamp.Before(weekStart) {
			continue
		}
		offset := int(ev.Timestamp.In(now.Location()).Sub(weekStart).Hours() / 24)
		if offset >= 0 && offset < 7 {
			days[offset] += ev.Value
		}
	}
	return days
}

// Last30DaysActivity returns 30 booleans (oldest first) for whether each day had activity.
func Last30DaysActivity(events []domain.TrackingEvent, now time.Time) [30]bool {
	activeDays := map[string]bool{}
	for _, ev := range events {
		switch ev.Type {
		case domain.EventWordsAdded, domain.EventFileTouched,
			domain.EventSessionMinutes, domain.EventTaskCompleted:
			activeDays[dateOf(ev.Timestamp)] = true
		}
	}
	var result [30]bool
	for i := range result {
		d := now.AddDate(0, 0, -(29 - i))
		result[i] = activeDays[dateOf(d)]
	}
	return result
}

func computeStreaks(activeDays map[string]bool, now time.Time) (current, longest int) {
	if len(activeDays) == 0 {
		return 0, 0
	}
	// Walk backwards from today.
	cur := 0
	for i := 0; ; i++ {
		d := dateOf(now.AddDate(0, 0, -i))
		if !activeDays[d] {
			break
		}
		cur++
	}
	current = cur

	// Find longest by iterating unique days.
	longest = 0
	run := 0
	d := now
	// Walk back 365 days max.
	for i := 0; i < 365; i++ {
		key := dateOf(d.AddDate(0, 0, -i))
		if activeDays[key] {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	return current, longest
}

func countActiveDaysInRange(activeDays map[string]bool, from, to time.Time) int {
	count := 0
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if activeDays[dateOf(d)] {
			count++
		}
	}
	return count
}

func dateOf(t time.Time) string {
	y, m, d := t.Local().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

func startOfWeek(t time.Time) time.Time {
	t = t.Local()
	y, mo, d := t.Date()
	start := time.Date(y, mo, d, 0, 0, 0, 0, t.Location())
	offset := (int(start.Weekday()) + 6) % 7 // Mon=0
	return start.AddDate(0, 0, -offset)
}
