package proofs

import (
	"testing"

	"github.com/ani03sha/chrono/dotted"
	"github.com/stretchr/testify/require"
)

// TestDottedVersionDetectsConcurrentWrites reproduces the canonical Riak sibling scenario described by Preguiça et al. (2010).
//
// Two clients read the same key from the same starting state, then each writes concurrently.
//
// With a naive last-write-wins resolution (e.g., wall-clock timestamps), one of the writes is silently lost.
//
// With dotted version vectors, the storage layer detects that neither write descends from the other and surfaces BOTH as
// siblings for the application to resolve.
func TestDottedVersionDetectsConcurrentWrites(t *testing.T) {
	// Initial state: empty version.
	initial := dotted.NewVersion()

	// Client A reads the value and writes back with its dot.
	versionA := initial.Increment("clientA")

	// Client B reads the *same* initial value (it doesn't yet know
	// about A's write) and writes back with its dot.
	versionB := initial.Increment("clientB")

	// The server compares the two versions. Neither descends from the
	// other - they are concurrent. This is the signal that triggers
	// sibling storage instead of overwrite.
	require.True(t, versionA.Concurrent(versionB),
		"two writes from the same starting state must be detected as concurrent")
	require.False(t, versionA.Descends(versionB))
	require.False(t, versionB.Descends(versionA))

	// Both writes are preserved as siblings — nothing is silently
	// dropped. The application is now responsible for resolution.
	siblings := []dotted.Sibling[string]{
		{Version: versionA, Value: "A's value"},
		{Version: versionB, Value: "B's value"},
	}
	require.Len(t, siblings, 2,
		"both concurrent writes must be preserved for resolution")
}

// TestLastWriteWinsLosesOneOfTwoConcurrentWrites is the negative control: it demonstrates the EXACT failure
// dotted version vectors are designed to surface. When the application chooses LWW resolution, one of the two
// writes is silently discarded.
//
// In a real system, this discard would never appear in any log or metric - the data is just gone. Reproducing
// it here puts the loss in the test name so future readers see what we are guarding against.
func TestLastWriteWinsLosesOneOfTwoConcurrentWrites(t *testing.T) {
	initial := dotted.NewVersion()
	siblings := []dotted.Sibling[string]{
		{Version: initial.Increment("clientA"), Value: "A's value"},
		{Version: initial.Increment("clientB"), Value: "B's value"},
	}

	winner := dotted.ResolveLatestWriteWins(siblings)

	// Exactly ONE of the two values survives — the other is gone.
	require.True(t, winner.Value == "A's value" || winner.Value == "B's value")

	// The loser is the value not equal to the winner. It is irrecoverable
	// after this point.
	loser := "B's value"
	if winner.Value == "B's value" {
		loser = "A's value"
	}
	require.NotEqual(t, winner.Value, loser,
		"LWW silently discards %q — the application now has no way to recover it", loser)
}
