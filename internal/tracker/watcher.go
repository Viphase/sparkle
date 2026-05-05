// Watcher is a debounced rescanner that emits a tick after a quiet period
// following a file change. It is intentionally tea-agnostic: callers wrap
// the returned channel in a tea.Cmd loop so Update never blocks.
//
// We don't import fsnotify here to keep the domain/tracker boundary thin
// and the test surface trivial. Callers feed Notify() from whatever change
// signal they have (fsnotify, in-app saves, manual refresh).
package tracker

import (
	"sync"
	"time"
)

// Watcher coalesces rapid Notify calls into one Tick after `idle` quiet time.
// Concurrency-safe.
type Watcher struct {
	idle  time.Duration
	clock func() time.Time

	mu       sync.Mutex
	pending  bool
	lastBeep time.Time
	ticks    chan struct{}
	stop     chan struct{}
	stopped  bool
}

// NewWatcher returns a watcher that fires a tick on the returned channel
// when no Notify call has arrived for `idle` duration.
//
// idle ≤ 0 falls back to 2 * time.Second per M13 spec.
func NewWatcher(idle time.Duration) *Watcher {
	if idle <= 0 {
		idle = 2 * time.Second
	}
	w := &Watcher{
		idle:  idle,
		clock: time.Now,
		ticks: make(chan struct{}, 1),
		stop:  make(chan struct{}),
	}
	go w.loop()
	return w
}

// Ticks returns the channel that receives a struct{} after each idle period
// following a Notify burst.
func (w *Watcher) Ticks() <-chan struct{} { return w.ticks }

// Notify signals that a change happened. Multiple Notify calls within the
// idle window are coalesced into a single tick.
func (w *Watcher) Notify() {
	w.mu.Lock()
	w.pending = true
	w.lastBeep = w.clock()
	w.mu.Unlock()
}

// Stop terminates the watcher's goroutine.
func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	close(w.stop)
	w.mu.Unlock()
}

func (w *Watcher) loop() {
	// Poll at idle/4 so we react promptly without busy-waiting.
	poll := w.idle / 4
	if poll < 100*time.Millisecond {
		poll = 100 * time.Millisecond
	}
	tick := time.NewTicker(poll)
	defer tick.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-tick.C:
			w.mu.Lock()
			if !w.pending {
				w.mu.Unlock()
				continue
			}
			if w.clock().Sub(w.lastBeep) < w.idle {
				w.mu.Unlock()
				continue
			}
			w.pending = false
			w.mu.Unlock()
			// Non-blocking send: if a tick is already waiting, drop the new one.
			select {
			case w.ticks <- struct{}{}:
			default:
			}
		}
	}
}
