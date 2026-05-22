package hlc

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ani03sha/chrono/internal/wallclock"
	"github.com/stretchr/testify/require"
)

// newFake creates a fake-clock-backed HLC at a known wall time. Most
// tests use this so they can drive the wall clock deterministically.
func newFake(t *testing.T, wallSec int64) (*Clock, *wallclock.Fake) {
	t.Helper()
	fake := wallclock.NewFake(time.Unix(wallSec, 0))
	return New(fake), fake
}

func TestNowAdoptsWallTimeOnFirstCall(t *testing.T) {
	c, fake := newFake(t, 1000)
	ts := c.Now()
	require.Equal(t, fake.NowNanos(), ts.WallTime)
	require.Equal(t, uint32(0), ts.Logical)
}

func TestNowIsMonotonicAcrossManyCalls(t *testing.T) {
	// Consecutive Now() calls must strictly increase in HLC order, no
	// matter what the wall clock does between them.
	c, fake := newFake(t, 1000)
	prev := c.Now()

	for i := 0; i < 100; i++ {
		// Mix of forward, no-op, and backward wall-clock movements.
		switch i % 3 {
		case 0:
			fake.Advance(time.Millisecond)
		case 1:
			// no-op — wall clock stays still
		case 2:
			fake.Reverse(100 * time.Microsecond)
		}
		cur := c.Now()
		require.True(t, prev.Less(cur),
			"iter %d: prev=%s must be less than cur=%s", i, prev, cur)
		prev = cur
	}
}

func TestNowIncrementsLogicalWhenWallUnchanged(t *testing.T) {
	c, _ := newFake(t, 1000)
	a := c.Now()  // (wall, 0)
	b := c.Now()  // wall didn't move → (wall, 1)
	c2 := c.Now() // → (wall, 2)

	require.Equal(t, a.WallTime, b.WallTime)
	require.Equal(t, uint32(1), b.Logical)
	require.Equal(t, uint32(2), c2.Logical)
}

func TestNowResetsLogicalWhenWallAdvances(t *testing.T) {
	c, fake := newFake(t, 1000)
	c.Now()
	c.Now()
	c.Now() // logical = 2

	fake.Advance(time.Second)
	ts := c.Now()
	require.Equal(t, uint32(0), ts.Logical, "logical resets when wall advances")
}

func TestNowSurvivesWallClockBackwardStep(t *testing.T) {
	// THE key test for HLC: when the wall clock steps backward (NTP
	// correction), the HLC must NOT go backward. It absorbs the
	// regression into the logical counter.
	c, fake := newFake(t, 1000)
	first := c.Now() // (1000s, 0)

	fake.Reverse(500 * time.Millisecond)
	second := c.Now()
	require.Equal(t, first.WallTime, second.WallTime, "wall stays put across backward step")
	require.Equal(t, uint32(1), second.Logical)

	fake.Reverse(time.Second)
	third := c.Now()
	require.Equal(t, first.WallTime, third.WallTime)
	require.Equal(t, uint32(2), third.Logical)
	require.True(t, second.Less(third))
}

func TestReceiveAdoptsRemoteWhenAhead(t *testing.T) {
	// Remote is in the future relative to our wall. We must adopt
	// remote's wall and set our logical to remote.Logical + 1.
	c, _ := newFake(t, 1000)
	remote := Timestamp{WallTime: time.Unix(1100, 0).UnixNano(), Logical: 5}

	got, err := c.Receive(remote)
	require.NoError(t, err)
	require.Equal(t, remote.WallTime, got.WallTime)
	require.Equal(t, uint32(6), got.Logical)
}

func TestReceiveIgnoresRemoteWhenBehind(t *testing.T) {
	// Our wall is ahead of the remote. After Receive, our wall stays;
	// our logical bumps by 1.
	c, fake := newFake(t, 2000)
	c.Now() // state = (2000s, 0)

	remote := Timestamp{WallTime: time.Unix(1000, 0).UnixNano(), Logical: 5}
	got, err := c.Receive(remote)
	require.NoError(t, err)
	require.Equal(t, fake.NowNanos(), got.WallTime)
	require.Equal(t, uint32(1), got.Logical)
}

func TestReceiveTakesMaxOfBothLogicalsWhenWallTimesMatch(t *testing.T) {
	// Local (l, 3) and remote (l, 7) with pt < l → take max(3, 7) + 1.
	c, fake := newFake(t, 1000)
	c.Now()
	c.Now()
	c.Now()
	c.Now() // logical = 3, wall = 1000s

	// Step wall backward so pt < l for the receive.
	fake.Reverse(500 * time.Millisecond)

	remote := Timestamp{WallTime: c.Current().WallTime, Logical: 7}
	got, err := c.Receive(remote)
	require.NoError(t, err)
	require.Equal(t, remote.WallTime, got.WallTime)
	require.Equal(t, uint32(8), got.Logical, "max(3, 7) + 1")
}

func TestReceiveResetsLogicalWhenPhysicalTimeIsHighest(t *testing.T) {
	c, fake := newFake(t, 1000)
	c.Now() // state (1000s, 0)

	// Advance wall well past everyone.
	fake.Advance(time.Hour)
	remote := Timestamp{WallTime: time.Unix(1500, 0).UnixNano(), Logical: 9}

	got, err := c.Receive(remote)
	require.NoError(t, err)
	require.Equal(t, fake.NowNanos(), got.WallTime)
	require.Equal(t, uint32(0), got.Logical, "pt wins → logical resets")
}

func TestReceiveRejectsExcessiveDrift(t *testing.T) {
	fake := wallclock.NewFake(time.Unix(1000, 0))
	c := NewWithMaxDrift(fake, 250*time.Millisecond)
	c.Now()
	before := c.Current()

	// Remote claims to be 10 seconds in the future — well past 250ms.
	remote := Timestamp{WallTime: fake.NowNanos() + int64(10*time.Second), Logical: 0}

	_, err := c.Receive(remote)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrMaxDriftExceeded))

	// State must be unchanged after an error.
	require.True(t, before.Equal(c.Current()),
		"Clock state must not change on ErrMaxDriftExceeded")
}

func TestReceiveAcceptsRemoteWithinDrift(t *testing.T) {
	fake := wallclock.NewFake(time.Unix(1000, 0))
	c := NewWithMaxDrift(fake, 250*time.Millisecond)

	remote := Timestamp{WallTime: fake.NowNanos() + int64(100*time.Millisecond), Logical: 3}
	got, err := c.Receive(remote)
	require.NoError(t, err)
	require.Equal(t, remote.WallTime, got.WallTime)
	require.Equal(t, uint32(4), got.Logical)
}

func TestMaxDriftZeroDisablesCheck(t *testing.T) {
	// New (no maxDrift) accepts arbitrarily-future timestamps.
	c, fake := newFake(t, 1000)
	remote := Timestamp{WallTime: fake.NowNanos() + int64(time.Hour), Logical: 0}
	_, err := c.Receive(remote)
	require.NoError(t, err)
}

func TestTimestampOrdering(t *testing.T) {
	a := Timestamp{WallTime: 100, Logical: 5}
	b := Timestamp{WallTime: 100, Logical: 6}
	c := Timestamp{WallTime: 200, Logical: 0}

	require.True(t, a.Less(b))
	require.True(t, b.Less(c))
	require.True(t, a.Less(c))

	require.False(t, b.Less(a))
	require.True(t, a.Equal(Timestamp{WallTime: 100, Logical: 5}))

	require.Equal(t, -1, a.Compare(b))
	require.Equal(t, 1, b.Compare(a))
	require.Equal(t, 0, a.Compare(a))
}

func TestCurrentDoesNotAdvance(t *testing.T) {
	c, _ := newFake(t, 1000)
	c.Now()
	cur := c.Current()
	require.True(t, cur.Equal(c.Current()))
	require.True(t, cur.Equal(c.Current()))
}

func TestConcurrentClockOperations(t *testing.T) {
	// 1000 Now() + 1000 Receive() in parallel. The contract under
	// `go test -race` is "no torn state"; we additionally check that
	// every emitted timestamp is monotonically the largest yet —
	// i.e., the final Current() dominates the lot.
	const N = 1000
	c, _ := newFake(t, 1000)

	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		max Timestamp
	)
	bump := func(ts Timestamp) {
		mu.Lock()
		if max.Less(ts) {
			max = ts
		}
		mu.Unlock()
	}

	for i := 0; i < N; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); bump(c.Now()) }()
		go func(i int) {
			defer wg.Done()
			ts, _ := c.Receive(Timestamp{WallTime: int64(1_000_000_000 + i), Logical: 0})
			bump(ts)
		}(i)
	}
	wg.Wait()

	require.True(t, max.Equal(c.Current()) || max.Less(c.Current()),
		"the maximum observed timestamp must equal the final clock state")
}
