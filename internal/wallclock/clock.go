// This package provides an injectable wall-clock interface used by every chrono clock that depends on physical time.
//
// Production code calls Real() to obtain the system clock. Tests use the Fake type to control time deterministically:
// so that we can freeze it, advance it, reverse it to simulate NTP corrections, etc.
//
// This package is internal: it cannot be imported from outside the chrono module.
// The interface is an implementation detail of chrono's clocks, not part of chrono's public API.
package wallclock

import "time"

// Clock is a minimal interface chrono's clocks need from a wall-time source.
// Implementation must be safe for concurrent use.
type Clock interface {
	// Returns the current wall clock time.
	Now() time.Time

	// Returns the current wall clock time as Unix nanoseconds. It is a hot-path shortcut: callers that only need
	// an int64(HLC, in particular) avoid constructing or copying a time.Time.
	NowNanos() int64
}

// This is the production implementation of Clock. It is a zero-size struct so that the calls through the Clock
// interface don't require any heap allocation for the receiver.
type realClock struct{}

// Returns a clock backed by the system wall clock.
// The returned value is stateless and safe for concurrent use.
// Callers may discard or share it freely.
func Real() Clock {
	return realClock{}
}

// --- Why both Now() and NowNanos()? ---
// Because chrono's HLC stored WallTime int64. If the interface only had Now(), every HLC tick would do something like:
// c.Now().UnixNano(), still constructing time.Time, reading its fields, then discarding it. NowNanos() allows implementors
// to skip that construction in their own code.

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) NowNanos() int64 {
	return time.Now().UnixNano()
}
