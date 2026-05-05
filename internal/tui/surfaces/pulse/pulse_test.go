package pulse

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
	m := &Model{
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

	got := next.(*Model)
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

func TestDashboardScrollClampsToContent(t *testing.T) {
	m := &Model{
		theme: theme.PastelDark(),
		now:   time.Now,
	}
	// Scroll way past content.
	m.scroll = 9999
	// View should clamp scroll so the detail section still shows content.
	view := m.View(90, 30)
	if view == "" {
		t.Fatal("view should not be empty after excess scroll")
	}
	// After View, scroll should have been clamped.
	if m.scroll > 100 {
		t.Fatalf("scroll not clamped; got %d", m.scroll)
	}
}

func TestDashboardLogoCache(t *testing.T) {
	m := &Model{
		theme: theme.PastelDark(),
		now:   time.Now,
	}
	// First render at width 90 — should populate cache keyed to logoW=86.
	m.View(90, 30)
	if m.cachedLogoW == 0 {
		t.Fatal("logo cache should be populated after View")
	}
	firstLogoW := m.cachedLogoW

	// Second render at same width — cachedLogoW must be unchanged.
	m.View(90, 30)
	if m.cachedLogoW != firstLogoW {
		t.Fatalf("cachedLogoW changed on same-width render: %d → %d", firstLogoW, m.cachedLogoW)
	}

	// Narrow render switches to compact logo (logoW=0 sentinel).
	m.View(40, 30)
	if m.cachedLogoW != 0 {
		t.Fatalf("expected cachedLogoW=0 for narrow render, got %d", m.cachedLogoW)
	}

	// Back to wide — cache key changes back.
	m.View(90, 30)
	if m.cachedLogoW != firstLogoW {
		t.Fatalf("expected cachedLogoW=%d after returning to wide, got %d", firstLogoW, m.cachedLogoW)
	}
}
