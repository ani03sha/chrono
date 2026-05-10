package vector_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/ani03sha/chrono/vector"
	"github.com/stretchr/testify/require"
)

// Builds a Snapshot with the given nodeID and vector for use in test inputs. Reaches through New/Now to avoid exposing
// Snapshot's internals to test code.
func snapshotForTest(nodeID string, vec map[string]uint64) vector.Snapshot {
	return vector.NewFromMap(nodeID, vec).Now()
}

func TestNewStartsEmpty(t *testing.T) {
	c := vector.New("n1")
	s := c.Now()

	require.Equal(t, "n1", s.NodeID())
	require.Equal(t, uint64(0), s.Get("n1"), "no events yet, even self should be 0")
	require.Empty(t, s.Nodes())
}

func TestNewRejectsEmptyNodeID(t *testing.T) {
	require.Panics(t, func() { vector.New("") })
	require.Panics(t, func() { vector.NewFromMap("", nil) })
}

func TestTickIncrementsOnlySelf(t *testing.T) {
	c := vector.New("n1")

	s1 := c.Tick()
	require.Equal(t, uint64(1), s1.Get("n1"))
	require.ElementsMatch(t, []string{"n1"}, s1.Nodes())

	s2 := c.Tick()
	require.Equal(t, uint64(2), s2.Get("n1"))
	require.Equal(t, uint64(0), s2.Get("n2"), "Tick must not introduce new nodes")
}

func TestSendIncrementsSelf(t *testing.T) {
	c := vector.New("n1")
	s := c.Send()
	require.Equal(t, uint64(1), s.Get("n1"))
}

func TestReceiveMergesAndIncrementsSelf(t *testing.T) {
	c := vector.New("n1")
	c.Tick() // n1 = 1

	remote := snapshotForTest("n2", map[string]uint64{
		"n1": 0, // sender's view of us; lower than ours
		"n2": 5,
		"n3": 3,
	})

	s := c.Receive(remote)

	// Merge picks max(local, remote) per component, then self increments.
	require.Equal(t, uint64(2), s.Get("n1"), "self = max(1, 0) + 1 = 2")
	require.Equal(t, uint64(5), s.Get("n2"))
	require.Equal(t, uint64(3), s.Get("n3"))
}

func TestSnapshotImmutability(t *testing.T) {
	c := vector.New("n1")

	s1 := c.Tick() // s1 captures n1 = 1

	c.Tick()
	c.Tick() // clock now at n1 = 3

	require.Equal(t, uint64(1), s1.Get("n1"),
		"snapshot taken at n1=1 must not change when the clock advances")
}

func TestNodesReturnsIndependentSlice(t *testing.T) {
	c := vector.New("n1")
	c.Tick()
	s := c.Now()

	nodes := s.Nodes()
	nodes[0] = "MUTATED"

	require.Equal(t, []string{"n1"}, s.Nodes(),
		"Nodes() must return a fresh slice; caller mutations cannot affect the snapshot")
}

func TestConcurrentTicksAndSends(t *testing.T) {
	const (
		goroutines      = 50
		opsPerGoroutine = 200
	)

	c := vector.New("n1")

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				if j%2 == 0 {
					c.Tick()
				} else {
					c.Send()
				}
			}
		}()
	}
	wg.Wait()

	require.Equal(t, uint64(goroutines*opsPerGoroutine), c.Now().Get("n1"),
		"every concurrent Tick/Send must increment self exactly once")
}

func TestRelationAllCases(t *testing.T) {
	cases := []struct {
		name string
		a, b map[string]uint64
		want vector.Relation
	}{
		{"a less than b on single node",
			map[string]uint64{"A": 1}, map[string]uint64{"A": 2}, vector.Before},
		{"a greater than b on single node",
			map[string]uint64{"A": 2}, map[string]uint64{"A": 1}, vector.After},
		{"equal across two nodes",
			map[string]uint64{"A": 1, "B": 1}, map[string]uint64{"A": 1, "B": 1}, vector.Equal},
		{"concurrent — diverging in opposite directions",
			map[string]uint64{"A": 2, "B": 1}, map[string]uint64{"A": 1, "B": 2}, vector.Concurrent},
		{"concurrent — disjoint node sets",
			map[string]uint64{"A": 1}, map[string]uint64{"B": 1}, vector.Concurrent},
		{"empty before non-empty",
			map[string]uint64{}, map[string]uint64{"A": 1}, vector.Before},
		{"non-empty after empty",
			map[string]uint64{"A": 1}, map[string]uint64{}, vector.After},
		{"empty equals empty",
			map[string]uint64{}, map[string]uint64{}, vector.Equal},
		{"a is prefix of b across many nodes",
			map[string]uint64{"A": 1, "B": 2}, map[string]uint64{"A": 1, "B": 2, "C": 3}, vector.Before},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := snapshotForTest("a", tc.a)
			b := snapshotForTest("b", tc.b)
			require.Equal(t, tc.want, vector.Compare(a, b),
				"Compare(%v, %v)", tc.a, tc.b)
		})
	}
}

func TestRelationStringer(t *testing.T) {
	require.Equal(t, "before", vector.Before.String())
	require.Equal(t, "after", vector.After.String())
	require.Equal(t, "equal", vector.Equal.String())
	require.Equal(t, "concurrent", vector.Concurrent.String())
}

// Demonstrates the canonical send/receive pattern and the Compare relation that vector clocks make available.
func ExampleClock() {
	n1 := vector.New("n1")
	n2 := vector.New("n2")

	n1.Tick()
	n1.Tick()

	msg := n1.Send()
	n2.Receive(msg)

	s1 := n1.Now()
	s2 := n2.Now()

	fmt.Println("n1:", s1.Get("n1"), s1.Get("n2"))
	fmt.Println("n2:", s2.Get("n1"), s2.Get("n2"))
	fmt.Println("relation:", vector.Compare(s1, s2))

	// Output:
	// n1: 3 0
	// n2: 3 1
	// relation: before
}
