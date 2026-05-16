package wallclock

import (
	"sync"
	"time"
)

// This is a clock whose time is controlled by the test. It doesn't advance on its own:
// callers must call Set, Advance, or Reverse to move it.
//
// All methods are safe for concurrent use, so a test goroutine can advance the clock while
// the code under test is reading it.
type Fake struct {
	mu  sync.Mutex
	now time.Time
}

// Returns a clock whose time is controlled by the test.
func NewFake(initial time.Time) *Fake {
	return &Fake{
		now: initial,
	}
}

// Returns the Fake's current time.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Returns the Fake's current time in nanoseconds.
func (f *Fake) NowNanos() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now.UnixNano()
}

// Jumps the Fake to the given absolute time. Useful when a test needs to position the clock
// at a precise instant rather than advance from now.
func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = t
}

// Moves the Fake's time forward by the given duration d.
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
}

// This is an alias for Advance, provided because scenario tests often read more naturally as
// "step forward by 100ms" than "advance by 100ms".
func (f *Fake) Step(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
}

// Moves the Fake's time backward by the given duration d.
func (f *Fake) Reverse(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(-d)
}
