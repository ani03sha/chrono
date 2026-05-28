package proofs

import (
	"testing"
	"time"

	"github.com/ani03sha/chrono/hlc"
	"github.com/ani03sha/chrono/internal/wallclock"
	"github.com/stretchr/testify/require"
)

// TestHLCPreservesCausalityAcrossSkewedNodes reproduces the property used by CockroachDB, YugabyteDB, and MongoDB
// to order events across nodes whose wall clocks disagree. Even when Node 2's wall clock is 50ms ahead of Node 1's,
// an HLC Receive on Node 2 of a message from Node 1 produces a timestamp strictly greater than the send.
//
// The 50ms skew is realistic - NTP-synced data center clocks routinely diverge by tens of milliseconds even when working correctly.
func TestHLCPreservesCausalityAcrossSkewedNodes(t *testing.T) {
	// Node 1 at wall=1000s. Node 2 at wall=1000s + 50ms — AHEAD.
	fake1 := wallclock.NewFake(time.Unix(1000, 0))
	fake2 := wallclock.NewFake(time.Unix(1000, 0).Add(50 * time.Millisecond))

	clock1 := hlc.New(fake1)
	clock2 := hlc.New(fake2)

	// Node 1 sends a message. Its HLC stamp uses its own wall time.
	msg := clock1.Send()

	// Node 2 receives the message. The HLC Receive algorithm sees:
	//   pt    = node2 wall (= node1 wall + 50ms) - ahead of the message
	//   remote.l = node1's WallTime
	//   l     = 0 (node2 has never ticked)
	// l' = max(0, remote.l, pt) = pt, so the receive falls into the
	// "pt wins" case and the new logical counter resets to 0.
	got, err := clock2.Receive(msg)
	require.NoError(t, err)

	// Causality assertion: node 2's HLC must be at least as great as
	// the received message. This is the etcd/CockroachDB invariant.
	require.False(t, got.Less(msg),
		"after Receive, node 2's HLC must not be less than the message's stamp")

	// Verify the same property in the OTHER direction — node 2 sends,
	// node 1 (BEHIND) receives. Node 1 must absorb the future stamp.
	msg2 := clock2.Send()
	got1, err := clock1.Receive(msg2)
	require.NoError(t, err)
	require.False(t, got1.Less(msg2),
		"node 1 (behind) must adopt node 2's future stamp on Receive")

	// The logical counter on node 1 must be at least msg2.Logical + 1
	// because the receive fell into the "remote wins" case.
	require.Equal(t, msg2.WallTime, got1.WallTime,
		"node 1 adopts node 2's WallTime since it is ahead of node 1's pt")
	require.Equal(t, msg2.Logical+1, got1.Logical,
		"node 1's logical = remote.Logical + 1 to ensure strict causality")
}

// TestHLCRingPreservesCausality runs a three-node ring of messages analogous to Lamport 1978 Figure 2,
// but with HLC and skewed wall clocks. Every link in the chain must produce a strictly larger HLC timestamp despite the skew.
func TestHLCRingPreservesCausality(t *testing.T) {
	// Three nodes with progressively-advanced wall clocks.
	base := time.Unix(1000, 0)
	f1 := wallclock.NewFake(base)
	f2 := wallclock.NewFake(base.Add(30 * time.Millisecond))
	f3 := wallclock.NewFake(base.Add(60 * time.Millisecond))

	c1 := hlc.New(f1)
	c2 := hlc.New(f2)
	c3 := hlc.New(f3)

	m1 := c1.Send()
	t2, err := c2.Receive(m1)
	require.NoError(t, err)

	m2 := c2.Send()
	t3, err := c3.Receive(m2)
	require.NoError(t, err)

	m3 := c3.Send()
	t1, err := c1.Receive(m3)
	require.NoError(t, err)

	// Walk the chain: every step must be strictly greater than the previous.
	require.True(t, m1.Less(t2), "send→recv: t2 > m1")
	require.True(t, t2.Less(m2), "local progression on node 2")
	require.True(t, m2.Less(t3), "send→recv: t3 > m2")
	require.True(t, t3.Less(m3), "local progression on node 3")
	require.True(t, m3.Less(t1), "send→recv: t1 > m3")

	// Transitive closure: t1 is causally after every event in the ring.
	require.True(t, m1.Less(t1))
	require.True(t, m2.Less(t1))
	require.True(t, m3.Less(t1))
}
