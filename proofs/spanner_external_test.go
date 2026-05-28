package proofs

import (
	"context"
	"testing"
	"time"

	"github.com/ani03sha/chrono/truetime"
	"github.com/stretchr/testify/require"
)

// TestSpannerExternalConsistency reproduces the external-consistency property from Corbett et al. (2012),
// "Spanner: Google's Globally - Distributed Database."
//
// External consistency: if transaction T1 commits before T2 starts in real wall-clock time, then T1's commit
// timestamp must be strictly less than T2's commit timestamp.
//
// Spanner achieves this with TrueTime + commit-wait:
//
//  1. T1 picks commit timestamp s = TT.Now().Latest.
//  2. T1 performs commit work.
//  3. T1 waits until TT.Now().Earliest > s ("commit-wait").
//  4. Only then does T1 become visible.
//
// After step 3, any T2 sampling TT.Now() will get an interval whose Earliest > s, so T2's commit timestamp will be > s.
// External consistency holds.
func TestSpannerExternalConsistency(t *testing.T) {
	base := time.Unix(1000, 0)
	// 7ms uncertainty interval — same magnitude Spanner reports in
	// production with GPS+atomic clocks.
	src := truetime.NewFakeSource(truetime.Interval{
		Earliest: base,
		Latest:   base.Add(7 * time.Millisecond),
	})
	clock := truetime.New(src)

	// T1 picks its commit timestamp as the latest possible "now."
	t1Commit := clock.Now().Latest

	// Simulate time passing by advancing the source's interval after a
	// brief delay. This stands in for the wall clock actually moving
	// forward in a real system.
	go func() {
		time.Sleep(15 * time.Millisecond)
		src.Set(truetime.Interval{
			Earliest: base.Add(20 * time.Millisecond),
			Latest:   base.Add(27 * time.Millisecond),
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Commit-wait. After this returns, clock.After(t1Commit) is true
	// — we are CERTAIN time has moved past t1Commit, everywhere.
	require.NoError(t, truetime.CommitWait(ctx, clock, t1Commit))
	require.True(t, clock.After(t1Commit),
		"after commit-wait, the clock must be provably past t1Commit")

	// T2 starts. Its sampled "earliest possible now" must be strictly
	// greater than T1's commit timestamp. This is external consistency.
	t2Earliest := clock.Now().Earliest
	require.True(t, t2Earliest.After(t1Commit),
		"external consistency: T2's start time must be strictly after T1's commit")
}

// TestSpannerWithoutCommitWaitCanViolateExternalConsistency is the negative control. It shows that if we skip commit-wait,
// T2 can pick a timestamp inside T1's uncertainty interval — which means an observer who saw T1 commit could then watch T2
// start and find T2 was assigned an EARLIER timestamp. External consistency violated.
//
// This test exists not to demonstrate a chrono failure but to make the "why commit-wait" lesson concrete.
func TestSpannerWithoutCommitWaitCanViolateExternalConsistency(t *testing.T) {
	base := time.Unix(1000, 0)
	src := truetime.NewFakeSource(truetime.Interval{
		Earliest: base,
		Latest:   base.Add(7 * time.Millisecond),
	})
	clock := truetime.New(src)

	// T1 picks commit timestamp = Latest = base + 7ms.
	t1Commit := clock.Now().Latest

	// WITHOUT commit-wait, T2 starts immediately. It samples Now() and
	// picks any value in its interval — say, Earliest = base.
	t2Pick := clock.Now().Earliest

	// t2Pick < t1Commit despite T2 starting AFTER T1 committed.
	// This is exactly the violation commit-wait is designed to prevent.
	require.True(t, t2Pick.Before(t1Commit),
		"without commit-wait, T2 can pick a timestamp earlier than T1's commit")
}
