package dashboard

import (
	"strings"
	"testing"
	"time"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/tui/msgs"
	"github.com/viphase/sparkle/internal/tui/theme"
)

// weeklySparkActivity was removed in favour of event-based tracking.
// The equivalent logic now lives in internal/tracker.WeeklyWordsByDay and
// is tested there; no dashboard-level test needed.

func TestDashboardRendersTrackingLoadedStats(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.Local)
	m := Model{
		theme: theme.PastelDark(),
		now: func() time.Time {
			return now
		},
	}

	next, _ := m.Update(msgs.TrackingLoadedMsg{AllEvents: map[string][]domain.TrackingEvent{
		"project_one": {
			{
				Timestamp: now.Add(-2 * time.Hour),
				Type:      domain.EventWordsAdded,
				Value:     120,
				Source:    "auto",
			},
			{
				Timestamp: now.AddDate(0, 0, -1),
				Type:      domain.EventWordsAdded,
				Value:     80,
				Source:    "auto",
			},
		},
	}})

	got := next.(Model)
	if got.stats.TodayWords != 120 {
		t.Fatalf("TodayWords=%d, want 120", got.stats.TodayWords)
	}
	// "Tracking" heading was removed; stats are shown inline on the dashboard.
	view := got.View(90, 30)
	for _, want := range []string{"today words", "words added", "activity"} {
		if !strings.Contains(view, want) {
			t.Fatalf("dashboard view missing %q: %q", want, view)
		}
	}
}
