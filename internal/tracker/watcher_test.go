package tracker

import (
	"testing"
	"time"
)

func TestWatcher_DebouncesNotify(t *testing.T) {
	w := NewWatcher(150 * time.Millisecond)
	defer w.Stop()

	// Burst of notifies — should coalesce to one tick.
	for i := 0; i < 5; i++ {
		w.Notify()
		time.Sleep(20 * time.Millisecond)
	}

	select {
	case <-w.Ticks():
		t.Fatalf("tick fired before idle period elapsed")
	case <-time.After(80 * time.Millisecond):
	}

	select {
	case <-w.Ticks():
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected one tick after idle, got none")
	}

	// No further ticks without new Notify.
	select {
	case <-w.Ticks():
		t.Fatalf("unexpected second tick without new Notify")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestWatcher_StopIsIdempotent(t *testing.T) {
	w := NewWatcher(50 * time.Millisecond)
	w.Stop()
	w.Stop() // must not panic
}

func TestWatcher_DefaultsTo2sIdle(t *testing.T) {
	w := NewWatcher(0)
	defer w.Stop()
	if w.idle != 2*time.Second {
		t.Fatalf("default idle = %v, want 2s", w.idle)
	}
}
