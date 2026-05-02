package wallclock

import (
	"sync"
	"time"
)

// This struct is used for tests that can control its behaviour. All methods are safe for concurrent use.
type Fake struct {
	mu  sync.Mutex
	now time.Time
}

// Initialize an instance of Fake whose initial time is given.
func NewFake(initial time.Time) *Fake {
	return &Fake{now: initial}
}

// Returns Fake's current time.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Returns Fake's current time in nanoseconds.
func (f *Fake) NowNanos() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now.UnixNano()
}

// Sets the current time with explicit value
func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	f.now = t
	f.mu.Unlock()
}

// Moves the clock forward by d duration.
func (f *Fake) Advance(d time.Duration) {
	if d < 0 {
		panic("wallclock.Fake.Advance: negative duration; use Reverse for backward steps")
	}
	f.mu.Lock()
	f.now = f.now.Add(d)
	f.mu.Unlock()
}

// Just an alias for Advance() for convenience.
func (f *Fake) Step(d time.Duration) {
	f.Advance(d)
}

// Moves the clock backwards by d. This simulates an NTP correction that steps the systems clock backward.
// This is the reason why logical clocks exist.
func (f *Fake) Reverse(d time.Duration) {
	if d < 0 {
		panic("wallclock.Fake.Reverse: negative duration; pass a positive value")
	}
	f.mu.Lock()
	f.now = f.now.Add(-d)
	f.mu.Unlock()
}
