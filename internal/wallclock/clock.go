// This package provides a small abstraction over the system wall clock.
//
// Clock types in chrono that depend on wall time (hlc, truetime) accept
// a wallclock.Clock interface rather than calling time.Now() directly.
//
// This makes them deterministically testable: prod code passes Real(),
// while tests pass a Fake whose time can be set, advanced, or even moved
// backward to simulate an NTP correction.
package wallclock

import "time"

type Clock interface {
	// Returns the current wall time.
	Now() time.Time

	// Returns the current wall time in nanoseconds since the epoch.
	// This is equivalent to Now().UnixNano(), but offered separately
	// so hot paths (HLC.Now, HLC.Send, HLC.Receive) can avoid materializing
	// a time.Time value.
	NowNanos() int64
}

// This is the production implementation of Clock backed by time.Now.
//
// It has no state, so a single value is sufficient. We expose it via
// Real() rather than as an exported type to keep the surface area minimal.
type realClock struct{}

// Returns a clock backed by the system wall clock (time.Now).
func Real() Clock {
	return realClock{}
}

func (realClock) Now() time.Time  { return time.Now() }
func (realClock) NowNanos() int64 { return time.Now().UnixNano() }
