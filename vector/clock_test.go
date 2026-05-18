package vector

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewStartsEmpty(t *testing.T) {
	c := New("n1")
	require.Equal(t, "n1", c.NodeID())
	require.Empty(t, c.Now().Nodes())
}

func TestNewFromMapCopiesInput(t *testing.T) {
	initial := map[string]uint64{"n1": 5, "n2": 3}
	c := NewFromMap("n1", initial)

	// Mutating the caller's map must not affect the Clock — that's the
	// invariant a defensive copy is supposed to provide.
	initial["n1"] = 999
	require.Equal(t, uint64(5), c.Now().Get("n1"))
}

func TestTickIncrementsOnlySelf(t *testing.T) {
	c := NewFromMap("n1", map[string]uint64{"n1": 0, "n2": 7})
	snap := c.Tick()
	require.Equal(t, uint64(1), snap.Get("n1"))
	require.Equal(t, uint64(7), snap.Get("n2"), "Tick must not touch other entries")
}

func TestSendIncrementsOnlySelf(t *testing.T) {
	c := New("n1")
	c.Send()
	c.Send()
	require.Equal(t, uint64(2), c.Now().Get("n1"))
}

func TestReceiveMergesAndIncrementsSelf(t *testing.T) {
	c := New("n1")
	c.Tick() // V = {n1:1}

	// Incoming snapshot from n2 carrying its own history plus knowledge of n1.
	remote := Snapshot{vec: map[string]uint64{"n1": 0, "n2": 5, "n3": 2}}

	out := c.Receive(remote)
	require.Equal(t, uint64(2), out.Get("n1"), "self bumped after merge")
	require.Equal(t, uint64(5), out.Get("n2"), "n2 absorbed from remote")
	require.Equal(t, uint64(2), out.Get("n3"), "n3 absorbed from remote")
}

func TestReceiveTakesMax(t *testing.T) {
	// Local has higher n2; remote has higher n3. Result must take max
	// of each, not blindly overwrite.
	c := NewFromMap("n1", map[string]uint64{"n1": 1, "n2": 10, "n3": 2})
	remote := Snapshot{vec: map[string]uint64{"n2": 4, "n3": 9}}
	out := c.Receive(remote)
	require.Equal(t, uint64(2), out.Get("n1"))  // bumped from 1
	require.Equal(t, uint64(10), out.Get("n2")) // local wins
	require.Equal(t, uint64(9), out.Get("n3"))  // remote wins
}

func TestSnapshotImmutability(t *testing.T) {
	// A Snapshot captured before further ticks must not reflect those ticks.
	// This is what makes Snapshots safe to send over the wire.
	c := New("n1")
	c.Tick() // V = {n1:1}
	before := c.Now()

	c.Tick() // V = {n1:2} — must not affect 'before'
	c.Tick() // V = {n1:3}

	require.Equal(t, uint64(1), before.Get("n1"), "snapshot must not see later mutations")
	require.Equal(t, uint64(3), c.Now().Get("n1"))
}

func TestSnapshotEqual(t *testing.T) {
	a := Snapshot{vec: map[string]uint64{"n1": 1, "n2": 2}}
	b := Snapshot{vec: map[string]uint64{"n1": 1, "n2": 2}}
	c := Snapshot{vec: map[string]uint64{"n1": 1, "n2": 3}}
	require.True(t, a.Equal(b))
	require.False(t, a.Equal(c))
}

func TestConcurrentClockOperations(t *testing.T) {
	// 1000 ticks + 1000 receives in parallel. With correct locking, the
	// final n1 entry equals the total number of operations, and no
	// goroutine observes torn state. Run under `go test -race`.
	const N = 1000
	c := New("n1")

	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); c.Tick() }()
		go func(i int) {
			defer wg.Done()
			remote := Snapshot{vec: map[string]uint64{"n2": uint64(i)}}
			c.Receive(remote)
		}(i)
	}
	wg.Wait()

	require.Equal(t, uint64(2*N), c.Now().Get("n1"))
}
