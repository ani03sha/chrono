package dotted

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewVersionIsEmpty(t *testing.T) {
	v := NewVersion()
	require.Empty(t, v.Dots())
}

func TestIncrementCreatesDot(t *testing.T) {
	v := NewVersion().Increment("clientA")
	require.Equal(t, uint64(1), v.Dots()["clientA"])

	v = v.Increment("clientA")
	require.Equal(t, uint64(2), v.Dots()["clientA"])
}

func TestIncrementDoesNotMutateReceiver(t *testing.T) {
	// The whole point of returning *Version instead of mutating is so
	// that prior versions attached to stored values stay stable.
	v1 := NewVersion().Increment("A")
	v2 := v1.Increment("A")

	require.Equal(t, uint64(1), v1.Dots()["A"], "v1 must be untouched after Increment")
	require.Equal(t, uint64(2), v2.Dots()["A"])
}

func TestMergePointwiseMax(t *testing.T) {
	a := NewVersion().Increment("A").Increment("A").Increment("B")
	b := NewVersion().Increment("A").Increment("B").Increment("B").Increment("C")
	m := a.Merge(b)

	require.Equal(t, uint64(2), m.Dots()["A"], "max(2, 1)")
	require.Equal(t, uint64(2), m.Dots()["B"], "max(1, 2)")
	require.Equal(t, uint64(1), m.Dots()["C"], "max(0, 1)")
}

func TestMergeIsCommutative(t *testing.T) {
	// Merge must produce identical output regardless of operand order.
	// Without this property, replication would diverge based on the
	// order in which nodes saw updates.
	a := NewVersion().Increment("A").Increment("A")
	b := NewVersion().Increment("A").Increment("B")
	require.True(t, a.Merge(b).Equals(b.Merge(a)))
}

func TestMergeIsAssociative(t *testing.T) {
	a := NewVersion().Increment("A")
	b := NewVersion().Increment("B")
	c := NewVersion().Increment("C")
	left := a.Merge(b).Merge(c)
	right := a.Merge(b.Merge(c))
	require.True(t, left.Equals(right))
}

func TestDescendsBasic(t *testing.T) {
	a := NewVersion().Increment("A")
	b := a.Increment("A") // b descends from a
	require.True(t, b.Descends(a))
	require.False(t, a.Descends(b))
}

func TestDescendsReflexive(t *testing.T) {
	v := NewVersion().Increment("A").Increment("B")
	require.True(t, v.Descends(v), "every version descends from itself")
}

func TestDescendsFromEmpty(t *testing.T) {
	empty := NewVersion()
	v := NewVersion().Increment("A")
	require.True(t, v.Descends(empty), "non-empty descends from empty")
	require.False(t, empty.Descends(v), "empty does not descend from non-empty")
}

func TestDominatesStrict(t *testing.T) {
	a := NewVersion().Increment("A")
	b := a.Increment("A")
	require.True(t, b.Dominates(a))
	require.False(t, a.Dominates(b))
	require.False(t, a.Dominates(a), "a version never strictly dominates itself")
}

func TestDominatesByNewActor(t *testing.T) {
	// b descends from a and adds a brand-new actor — still strict
	// domination because b knows about an actor a is implicitly at 0 on.
	a := NewVersion().Increment("A")
	b := a.Increment("B")
	require.True(t, b.Dominates(a))
}

func TestConcurrentDetection(t *testing.T) {
	// The Riak scenario: two clients write from the same base concurrently.
	base := NewVersion()
	a := base.Increment("clientA")
	b := base.Increment("clientB")

	require.True(t, a.Concurrent(b), "concurrent writes must be detected")
	require.True(t, b.Concurrent(a), "concurrency is symmetric")
	require.False(t, a.Descends(b))
	require.False(t, b.Descends(a))
}

func TestEqualsIgnoresInsertionOrder(t *testing.T) {
	a := NewVersion().Increment("A").Increment("B")
	b := NewVersion().Increment("B").Increment("A")
	require.True(t, a.Equals(b))
}

func TestDotsReturnsCopy(t *testing.T) {
	// Dots must not expose the internal map — that would defeat
	// immutability.
	v := NewVersion().Increment("A")
	d := v.Dots()
	d["A"] = 999
	require.Equal(t, uint64(1), v.Dots()["A"], "internal map must not be mutable through Dots")
}
