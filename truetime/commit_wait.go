package truetime

import (
	"context"
	"time"
)

// CommitWait blocks until the clock is certain that time has moved past commitAt — that is, until clock.After(commitAt)
// returns true. Returns ctx.Err() if ctx is cancelled before that happens.
//
// This is the building block of external consistency: a transaction that picks commit timestamp s should call
// CommitWait(ctx, clock, s) before exposing its commit. After CommitWait returns, any future observer sampling
// clock.Now() will see an interval whose Earliest is strictly greater than s.
func CommitWait(ctx context.Context, clock *Clock, commitAt time.Time) error {
	if clock.After(commitAt) {
		return nil
	}

	for {
		interval := clock.Now()
		if interval.Earliest.After(commitAt) {
			return nil
		}

		// Compute a target wait: the gap between commitAt and the current Earliest.
		// Cap at 100ms so the loop stays responsive to interval-changing events
		// (e.g., a background NTP refresh that shrinks uncertainty). Floor at 100µs so we don't spin.
		wait := commitAt.Sub(interval.Earliest)
		if wait > 100*time.Millisecond {
			wait = 100 * time.Millisecond
		}
		if wait < 100*time.Microsecond {
			wait = 100 * time.Microsecond
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// Runs fn, records the commit timestamp as the upper bound of the post-fn interval, and waits until that timestamp is in
// the past everywhere. The common pattern in a database commit path.
//
// Returns fn's error (without waiting) if fn fails, or any ctx error encountered during the wait.
func WithCommitWait(ctx context.Context, clock *Clock, fn func() error) error {
	if err := fn(); err != nil {
		return err
	}
	commitAt := clock.Now().Latest
	return CommitWait(ctx, clock, commitAt)
}
