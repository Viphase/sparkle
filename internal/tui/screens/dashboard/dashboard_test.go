package dashboard

import (
	"testing"
	"time"

	"github.com/viphase/sparkle/internal/domain"
)

func TestWeeklySparkActivityCountsCreatesAndUpdates(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.Local) // Thursday
	items := []domain.Spark{
		{
			CreatedAt: time.Date(2026, 4, 27, 9, 0, 0, 0, time.Local),
			UpdatedAt: time.Date(2026, 4, 29, 10, 0, 0, 0, time.Local),
		},
		{
			CreatedAt: time.Date(2026, 4, 30, 11, 0, 0, 0, time.Local),
			UpdatedAt: time.Date(2026, 4, 30, 11, 30, 0, 0, time.Local),
		},
		{
			CreatedAt: time.Date(2026, 4, 20, 9, 0, 0, 0, time.Local),
			UpdatedAt: time.Date(2026, 5, 4, 9, 0, 0, 0, time.Local),
		},
	}

	got := weeklySparkActivity(items, now)
	want := []int{1, 0, 1, 1, 0, 0, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("bucket %d=%d, want %d (all buckets: %v)", i, got[i], want[i], got)
		}
	}
}
