package hlc

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ani03sha/chrono/internal/wallclock"
)

// This is returned by Receive when the incoming timestamp's WallTime is more than the Clock's configured maxDrift
// ahead of the local wall clock. The Clock state is not modified when this error is returned.
var ErrMaxDriftExceeded = errors.New("hlc: incoming timestamp exceeds maxDrift")

// This is an HLC timestamp: a wall-time approximation (nanoseconds since epoch) paired with a logical counter
// that breaks ties within the same WallTime.
type Timestamp struct {
	WallTime int64
	Logical  uint32
}

// Reports whether "a" sorts strictly before "b" in the HLC total order.
func (a Timestamp) Less(b Timestamp) bool {
	if a.WallTime != b.WallTime {
		return a.WallTime < b.WallTime
	}
	return a.Logical < b.Logical
}

// Reports whether "a" and "b" are equal timestamps.
func (a Timestamp) Equal(b Timestamp) bool {
	return a.WallTime == b.WallTime && a.Logical == b.Logical
}

// Compare returns -1, 0, or +1 according to whether a < b, a == b, or a > b in the HLC total order.
func (a Timestamp) Compare(b Timestamp) int {
	switch {
	case a.WallTime < b.WallTime:
		return -1
	case a.WallTime > b.WallTime:
		return 1
	case a.Logical < b.Logical:
		return -1
	case a.Logical > b.Logical:
		return 1
	default:
		return 0
	}
}

// Formats the timestamp as "<wall>.<logical>".
func (a Timestamp) String() string {
	return fmt.Sprintf("%d.%d", a.WallTime, a.Logical)
}

// This is an HLC owned by one process. All methods are safe for concurrent use.
type Clock struct {
	mu       sync.Mutex
	state    Timestamp
	wall     wallclock.Clock
	maxDrift time.Duration // 0 disables the drift check
}

// New returns a Clock backed by the given wall clock, with no drift check on incoming timestamps.
// Use NewWithMaxDrift to enable the check.
func New(wall wallclock.Clock) *Clock {
	return &Clock{wall: wall}
}

// This method returns a Clock that rejects incoming timestamps whose WallTime is more than maxDrift ahead
// of the local wall clock.
//
// A maxDrift of 0 disables the check (same as New). Typical production values are 250ms–1s — large enough to
// absorb expected inter-region clock skew, small enough to bound the damage from a misbehaving sender.
func NewWithMaxDrift(wall wallclock.Clock, maxDrift time.Duration) *Clock {
	return &Clock{wall: wall, maxDrift: maxDrift}
}

// Records a local event and returns the new HLC timestamp.
func (c *Clock) Now() Timestamp {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.advanceLocked()
}

// Send is an alias for Now provided so that call sites read intentionally when the returned timestamp is
// being attached to an outgoing message.
func (c *Clock) Send() Timestamp {
	return c.Now()
}

// Receive merges an incoming HLC timestamp into this Clock and returns the new local timestamp.
//
// If maxDrift is configured and remote.WallTime is more than maxDrift ahead of the local wall clock,
// the Clock is left unchanged and ErrMaxDriftExceeded is returned.
func (c *Clock) Receive(remote Timestamp) (Timestamp, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pt := c.wall.NowNanos()

	if c.maxDrift > 0 && remote.WallTime > pt+int64(c.maxDrift) {
		return Timestamp{}, fmt.Errorf("%w: remote=%d local=%d max=%s", ErrMaxDriftExceeded, remote.WallTime, pt, c.maxDrift)
	}

	newL := c.state.WallTime
	newL = max(remote.WallTime, newL)
	newL = max(pt, newL)

	var newC uint32
	switch {
	case newL == c.state.WallTime && newL == remote.WallTime:
		// Both local and remote were at the top: take the larger counter, then advance by 1.
		newC = c.state.Logical
		newC = max(newC, remote.Logical)
		newC++
	case newL == c.state.WallTime:
		// Local was already ahead of both remote and pt.
		newC = c.state.Logical + 1
	case newL == remote.WallTime:
		// Remote was ahead: adopt remote's l, and set our c so we are strictly after the remote's send event.
		newC = remote.Logical + 1
	default:
		// Physical time had moved past both. Fresh start.
		newC = 0
	}

	c.state = Timestamp{WallTime: newL, Logical: newC}
	return c.state, nil

}

// Current returns the Clock's most recent state without advancing it.
// Useful for inspection because observation is not itself an HLC event.
func (c *Clock) Current() Timestamp {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Applies the local-event update rule and returns the new state. Caller must hold c.mu.
func (c *Clock) advanceLocked() Timestamp {
	pt := c.wall.NowNanos()

	newL := c.state.WallTime
	newL = max(pt, newL)

	var newC uint32
	if newL == c.state.WallTime {
		newC = c.state.Logical + 1
	}

	c.state = Timestamp{WallTime: newL, Logical: newC}
	return c.state
}
