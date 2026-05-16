package wallclock

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRealClockMovesForward(t *testing.T) {
	c := Real()
	t1 := c.Now()
	// Sleep a touch so the second read is guaranteed to be later. The actual
	// duration doesn't matter, we are only asserting monotonic forward motion,
	// not any particular precision.
	time.Sleep(time.Millisecond)
	t2 := c.Now()
	require.True(t, t2.After(t1), "real clock must move forward over time")
}

func TestRealClockNowNanosMatchesNow(t *testing.T) {
	// NowNanos must agree with Now().UnixNano() to within a small window
	// (the two calls happen at slightly different instants).
	c := Real()
	a := c.Now().UnixNano()
	b := c.NowNanos()
	delta := b - a
	if delta < 0 {
		delta = -delta
	}
	require.Less(t, delta, int64(10*time.Millisecond),
		"NowNanos must report essentially the same instant as Now")
}

func TestFakeStartsAtInitial(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	f := NewFake(start)
	require.Equal(t, start, f.Now())
	require.Equal(t, start.UnixNano(), f.NowNanos())
}

func TestFakeSetJumpsToAbsoluteTime(t *testing.T) {
	f := NewFake(time.Unix(1000, 0))
	target := time.Unix(2000, 500)
	f.Set(target)
	require.Equal(t, target, f.Now())
}

func TestFakeAdvanceMovesForward(t *testing.T) {
	f := NewFake(time.Unix(1000, 0))
	f.Advance(250 * time.Millisecond)
	require.Equal(t, time.Unix(1000, 0).Add(250*time.Millisecond), f.Now())
}

func TestFakeStepIsAdvance(t *testing.T) {
	f := NewFake(time.Unix(1000, 0))
	f.Step(time.Second)
	require.Equal(t, time.Unix(1001, 0), f.Now())
}

func TestFakeReverseSimulatesNTPBackwardStep(t *testing.T) {
	// This is the scenario the whole abstraction exists to enable: we need
	// to be able to simulate an NTP backward correction so the proofs/
	// package can demonstrate that wall-clock-based ordering breaks.
	f := NewFake(time.Unix(1000, 0))
	f.Advance(500 * time.Millisecond)
	require.Equal(t, time.Unix(1000, 0).Add(500*time.Millisecond), f.Now())

	f.Reverse(200 * time.Millisecond)
	require.Equal(t, time.Unix(1000, 0).Add(300*time.Millisecond), f.Now())
}

func TestFakeIsSafeForConcurrentUse(t *testing.T) {
	// One goroutine advances the clock 1000 times; many others read it.
	// With the race detector enabled (`go test -race`) this catches any
	// missing locking around the now field.
	f := NewFake(time.Unix(1000, 0))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			f.Advance(time.Microsecond)
		}
	}()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_ = f.Now()
				_ = f.NowNanos()
			}
		}()
	}
	wg.Wait()

	// After 1000 microsecond advances we expect exactly 1ms of forward motion.
	require.Equal(t, time.Unix(1000, 0).Add(time.Millisecond), f.Now())
}
