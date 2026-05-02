package wallclock_test

import (
	"sync"
	"testing"
	"time"

	"github.com/ani03sha/chrono/internal/wallclock"
	"github.com/stretchr/testify/require"
)

func TestRealClockReturnsSystemTime(t *testing.T) {
	c := wallclock.Real()

	before := time.Now()
	got := c.Now()
	after := time.Now()

	require.False(t, got.Before(before), "Real().Now() returned a time before the call started")
	require.False(t, got.After(after), "Real().Now() returned a time after the call returned")
}

func TestRealClockNowNanosMatchesNow(t *testing.T) {
	c := wallclock.Real()

	nanos := c.NowNanos()
	got := c.Now()

	// The two reads happen sequentially; got should be at or after nanos so we can allow up to 10ms drift
	// for slow CI environments.
	delta := got.UnixNano() - nanos
	require.GreaterOrEqual(t, delta, int64(0), "Now() reported earlier than NowNanos()")
	require.Less(t, delta, int64(10*time.Millisecond), "NowNanos and Now drifted too far")
}

func TestFakeStartsAtInitial(t *testing.T) {
	initial := time.Unix(1000, 0)
	f := wallclock.NewFake(initial)

	require.Equal(t, initial, f.Now())
	require.Equal(t, initial.UnixNano(), f.NowNanos())
}

func TestFakeSet(t *testing.T) {
	f := wallclock.NewFake(time.Unix(1000, 0))
	target := time.Unix(2000, 0)

	f.Set(target)

	require.Equal(t, target, f.Now())
}

func TestFakeAdvance(t *testing.T) {
	initial := time.Unix(1000, 0)
	f := wallclock.NewFake(initial)

	f.Advance(500 * time.Millisecond)

	require.Equal(t, initial.Add(500*time.Millisecond), f.Now())
}

func TestFakeStepIsAliasForAdvance(t *testing.T) {
	initial := time.Unix(1000, 0)
	f := wallclock.NewFake(initial)

	f.Step(time.Second)

	require.Equal(t, initial.Add(time.Second), f.Now())
}

func TestFakeReverse(t *testing.T) {
	initial := time.Unix(1000, 0)
	f := wallclock.NewFake(initial)

	f.Reverse(200 * time.Millisecond)

	require.Equal(t, initial.Add(-200*time.Millisecond), f.Now())
}

func TestFakeAdvanceRejectsNegative(t *testing.T) {
	f := wallclock.NewFake(time.Unix(1000, 0))

	require.Panics(t, func() { f.Advance(-time.Second) })
}

func TestFakeReverseRejectsNegative(t *testing.T) {
	f := wallclock.NewFake(time.Unix(1000, 0))

	require.Panics(t, func() { f.Reverse(-time.Second) })
}

func TestFakeIsRaceFreeUnderConcurrentAccess(t *testing.T) {
	// This test must pass under `go test -race`. A missing or wrongly scoped mutex would surface here as a data race.
	const (
		writers    = 50
		readers    = 50
		iterations = 1000
	)

	initial := time.Unix(1000, 0)
	f := wallclock.NewFake(initial)

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				f.Advance(time.Nanosecond)
			}
		}()
	}
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = f.Now()
			}
		}()
	}

	wg.Wait()

	// Each writer advanced by 1ns exactly `iterations` times; total advance
	// is writers * iterations. Order doesn't matter because addition commutes.
	expected := initial.Add(time.Duration(writers * iterations))
	require.Equal(t, expected, f.Now(), "concurrent advances did not sum correctly")
}
