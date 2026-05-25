package truetime

import (
	"sync"
	"time"
)

// This produces intervals containing the current global time. Implementation must be safe for concurrent use;
// Now may be called from many goroutines at once.
type UncertaintySource interface {
	Now() Interval
}

// This is a TrueTime clock parameterized by an UncertaintySource. Almost all the logic lives in the source:
// Clock is a thin shell that exposes the convenience predicates After and Before on top of the interval the
// source returns.
type Clock struct {
	source UncertaintySource
}

// Returns a Clock backed by source.
func New(source UncertaintySource) *Clock {
	return &Clock{source: source}
}

// Returns the current interval from the underlying source.
func (c *Clock) Now() Interval {
	return c.source.Now()
}

// Reports whether the clock is certain that time has moved past t. Equivalent to clock.Now().
// Definitely(t) but expressed as a Clock method so callers can use it without storing an Interval.
func (c *Clock) After(t time.Time) bool {
	return c.source.Now().Earliest.After(t)
}

// Reports whether the clock is certain that time has not yet reached t. Equivalent to clock.Now().Latest.Before(t).
func (c *Clock) Before(t time.Time) bool {
	return c.source.Now().Latest.Before(t)
}

// This is an UncertaintySource whose interval is set by the test. It is the source the proofs/ package uses to reproduce
// Spanner-class scenarios deterministically: set an interval, run a commit, advance the interval, observe the result.
type FakeSource struct {
	mu       sync.RWMutex
	interval Interval
}

// This returns a FakeSource initialized to the given interval.
func NewFakeSource(initial Interval) *FakeSource {
	return &FakeSource{interval: initial}
}

// This replaces the source's interval.
func (f *FakeSource) Set(interval Interval) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.interval = interval
}

// Shifts both endpoints of the interval forward by d, keeping the width unchanged.
// Convenience helper for tests that simulate the passage of time without altering uncertainty.
func (f *FakeSource) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.interval = Interval{
		Earliest: f.interval.Earliest.Add(d),
		Latest:   f.interval.Latest.Add(d),
	}
}

// Returns the source's current interval
func (f *FakeSource) Now() Interval {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.interval
}
