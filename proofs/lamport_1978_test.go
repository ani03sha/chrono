package proofs

import (
	"testing"

	"github.com/ani03sha/chrono/lamport"
	"github.com/stretchr/testify/require"
)

// TestLamport1978 reproduces the trace from Leslie Lamport's 1978 paper "Time, Clocks, and the Ordering of Events in a
// Distributed System," Figure 2.
//
// Three processes exchange messages in a ring (P1→P2→P3→P1). After two internal events on P1, every subsequent message-passing
// step must produce a strictly larger Lamport timestamp than the step that caused it.
//
// The test traces the full chain and asserts the +1 receive rule at every link.
func TestLamport1978(t *testing.T) {
	p1 := lamport.New()
	p2 := lamport.New()
	p3 := lamport.New()

	// P1 records two internal events before sending.
	require.Equal(t, uint64(1), p1.Tick())
	require.Equal(t, uint64(2), p1.Tick())

	// P1 sends m1 to P2. The +1 rule on receive guarantees
	// ts(recv) > ts(send) even though P2's local clock started at 0.
	m1 := p1.Send()
	require.Equal(t, uint64(3), m1)
	require.Equal(t, uint64(4), p2.Receive(m1),
		"recv(m1) must exceed send(m1) by at least 1")

	// P2 sends m2 to P3 — the second link in the causal chain.
	m2 := p2.Send()
	require.Equal(t, uint64(5), m2)
	require.Equal(t, uint64(6), p3.Receive(m2))

	// P3 sends m3 back to P1, closing the ring. P1's local counter
	// was at 3; max(3, 7) + 1 = 8.
	m3 := p3.Send()
	require.Equal(t, uint64(7), m3)
	require.Equal(t, uint64(8), p1.Receive(m3),
		"P1's final receive must take max(local, remote) + 1")

	// The causal chain p1.Tick(1) → p1.Tick(2) → send(3) → recv(4) →
	// send(5) → recv(6) → send(7) → recv(8) is strictly monotonic.
	// That is the exact property Lamport (1978) proved.
}
