package ordering

import (
	"testing"

	"github.com/ani03sha/chrono/hlc"
	"github.com/ani03sha/chrono/vector"
	"github.com/stretchr/testify/require"
)

func TestLamportStampHappensBefore(t *testing.T) {
	a, b := LamportStamp(3), LamportStamp(7)
	require.True(t, a.HappensBefore(b))
	require.False(t, b.HappensBefore(a))
	require.False(t, a.HappensBefore(a), "strict less, not ≤")
}

func TestLamportStampConcurrent(t *testing.T) {
	require.True(t, LamportStamp(5).Concurrent(LamportStamp(5)))
	require.False(t, LamportStamp(5).Concurrent(LamportStamp(6)))
}

func TestHLCStampHappensBefore(t *testing.T) {
	a := HLCStamp{WallTime: 100, Logical: 5}
	b := HLCStamp{WallTime: 100, Logical: 6}
	c := HLCStamp{WallTime: 200, Logical: 0}

	require.True(t, a.HappensBefore(b))
	require.True(t, b.HappensBefore(c))
	require.False(t, c.HappensBefore(a))
}

func TestHLCStampConcurrent(t *testing.T) {
	a := HLCStamp{WallTime: 100, Logical: 5}
	b := HLCStamp{WallTime: 100, Logical: 5}
	c := HLCStamp{WallTime: 100, Logical: 6}
	require.True(t, a.Concurrent(b))
	require.False(t, a.Concurrent(c))
}

func TestVectorStampHappensBeforeAndConcurrent(t *testing.T) {
	// Construct three vector snapshots from real Clocks so we exercise
	// the actual code path the rest of the library produces.
	c1 := vector.New("n1")
	c2 := vector.New("n2")

	s1 := c1.Tick()            // {n1:1}
	sent := c1.Send()          // {n1:2}
	s2 := c2.Tick()            // {n2:1}  (concurrent with s1 and sent)
	merged := c2.Receive(sent) // {n1:2, n2:2}

	a := NewVectorStamp(s1)
	b := NewVectorStamp(sent)
	concurrent := NewVectorStamp(s2)
	after := NewVectorStamp(merged)

	require.True(t, a.HappensBefore(b))
	require.True(t, b.HappensBefore(after))
	require.True(t, a.Concurrent(concurrent))
	require.False(t, a.HappensBefore(concurrent))
	require.False(t, concurrent.HappensBefore(a))
}

func TestCrossTypeNotComparable(t *testing.T) {
	// A Lamport stamp and an HLC stamp share no causal relation by
	// construction — the adapters must return false rather than panic.
	l := LamportStamp(5)
	h := HLCStamp{WallTime: 100, Logical: 0}
	require.False(t, l.HappensBefore(h))
	require.False(t, l.Concurrent(h))
	require.False(t, h.HappensBefore(l))
	require.False(t, h.Concurrent(l))
}

// _ assertions: each adapter must satisfy the Comparable interface at
// compile time. These lines fail to build if an adapter regresses.
var (
	_ Comparable = LamportStamp(0)
	_ Comparable = HLCStamp(hlc.Timestamp{})
	_ Comparable = VectorStamp{}
)
