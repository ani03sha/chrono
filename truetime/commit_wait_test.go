package truetime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCommitWaitFastPathReturnsImmediately(t *testing.T) {
	// If the clock is already past commitAt, CommitWait returns
	// without waiting at all.
	src := NewFakeSource(Interval{Earliest: base.Add(time.Second), Latest: base.Add(2 * time.Second)})
	c := New(src)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	require.NoError(t, CommitWait(ctx, c, base))
	require.Less(t, time.Since(start), 5*time.Millisecond, "fast path must not poll")
}

func TestCommitWaitWaitsForUncertainty(t *testing.T) {
	// Set up an interval that does not yet pass commitAt. Advance the
	// source from another goroutine and verify CommitWait returns
	// once Earliest > commitAt.
	src := NewFakeSource(Interval{Earliest: base, Latest: base.Add(7 * time.Millisecond)})
	c := New(src)
	commitAt := base.Add(5 * time.Millisecond)

	go func() {
		time.Sleep(20 * time.Millisecond)
		// Move the interval so its Earliest is past commitAt.
		src.Set(Interval{Earliest: base.Add(10 * time.Millisecond), Latest: base.Add(17 * time.Millisecond)})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, CommitWait(ctx, c, commitAt))
	require.True(t, c.After(commitAt))
}

func TestCommitWaitContextCancellation(t *testing.T) {
	// Interval will never advance past commitAt; the ctx cancellation
	// is the only escape.
	src := NewFakeSource(Interval{Earliest: base, Latest: base.Add(time.Millisecond)})
	c := New(src)
	commitAt := base.Add(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err := CommitWait(ctx, c, commitAt)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestWithCommitWaitReturnsFnError(t *testing.T) {
	// If fn errors, WithCommitWait must surface that error without
	// waiting.
	src := NewFakeSource(Interval{Earliest: base, Latest: base.Add(time.Millisecond)})
	c := New(src)

	wantErr := errStub("commit failed")
	got := WithCommitWait(context.Background(), c, func() error { return wantErr })
	require.Equal(t, wantErr, got)
}

func TestWithCommitWaitWaitsAfterSuccessfulFn(t *testing.T) {
	src := NewFakeSource(Interval{Earliest: base, Latest: base.Add(5 * time.Millisecond)})
	c := New(src)

	// Background goroutine moves the source forward so the wait completes.
	go func() {
		time.Sleep(15 * time.Millisecond)
		src.Set(Interval{Earliest: base.Add(20 * time.Millisecond), Latest: base.Add(25 * time.Millisecond)})
	}()

	var ran bool
	err := WithCommitWait(context.Background(), c, func() error {
		ran = true
		return nil
	})
	require.NoError(t, err)
	require.True(t, ran)
}

func TestCommitWaitConcurrent(t *testing.T) {
	// Many goroutines waiting on the same clock — none should miss the
	// notification when the interval advances.
	src := NewFakeSource(Interval{Earliest: base, Latest: base.Add(time.Millisecond)})
	c := New(src)
	commitAt := base.Add(5 * time.Millisecond)

	const N = 50
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			require.NoError(t, CommitWait(ctx, c, commitAt))
		}()
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		src.Set(Interval{Earliest: base.Add(10 * time.Millisecond), Latest: base.Add(15 * time.Millisecond)})
	}()

	wg.Wait()
}

// errStub is a tiny error implementation to avoid an import for one
// trivial test.
type errStub string

func (e errStub) Error() string { return string(e) }
