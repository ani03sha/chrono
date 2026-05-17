package lamport

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewStartsAtZero(t *testing.T) {
	c := New()
	require.Equal(t, uint64(0), c.Now())
}

func TestNewAtStartsAtGivenValue(t *testing.T) {
	c := NewAt(85)
	require.Equal(t, uint64(85), c.Now())
	// And the next tick continues from there.
	require.Equal(t, uint64(86), c.Tick())
}

func TestTickIncrements(t *testing.T) {
	c := New()
	require.Equal(t, uint64(1), c.Tick())
	require.Equal(t, uint64(2), c.Tick())
	require.Equal(t, uint64(3), c.Tick())
}

func TestSendIncrements(t *testing.T) {
	c := New()
	require.Equal(t, uint64(1), c.Send())
	require.Equal(t, uint64(2), c.Send())
}

func TestReceiveTakesMaxPlusOne(t *testing.T) {
	// Local counter is behind the remote: jump to remote, then +1.
	c := New()
	require.Equal(t, uint64(11), c.Receive(10))
	require.Equal(t, uint64(11), c.Now())
}

func TestReceiveSmallerStillIncrements(t *testing.T) {
	// Local counter is ahead of the remote: ignore remote, just +1.
	// This is the case that proves the +1 is what guarantees causality —
	// without it, receive would be a no-op when local >= remote, and a
	// receive event would share its timestamp with the preceding event.
	c := NewAt(100)
	require.Equal(t, uint64(101), c.Receive(5))
}

func TestReceiveEqualStillIncrements(t *testing.T) {
	// Edge case: local exactly equals remote. The +1 must still fire.
	c := NewAt(7)
	require.Equal(t, uint64(8), c.Receive(7))
}

func TestNowDoesNotIncrement(t *testing.T) {
	c := NewAt(5)
	require.Equal(t, uint64(5), c.Now())
	require.Equal(t, uint64(5), c.Now())
	require.Equal(t, uint64(5), c.Now())
}

func TestResetOverwrites(t *testing.T) {
	c := NewAt(100)
	c.Reset(0)
	require.Equal(t, uint64(0), c.Now())
	c.Reset(999)
	require.Equal(t, uint64(999), c.Now())
}

func TestCausalChainAcrossThreeClocks(t *testing.T) {
	// Walk through a small distributed scenario by hand to verify the
	// algorithm composes correctly across processes:
	//
	//   P1 ticks twice, then sends to P2.
	//   P2 receives, then sends to P3.
	//   P3 receives, then sends back to P1.
	//
	// Causality requires the final receive on P1 to carry a timestamp
	// strictly greater than every prior event on the chain.
	p1, p2, p3 := New(), New(), New()

	require.Equal(t, uint64(1), p1.Tick())
	require.Equal(t, uint64(2), p1.Tick())

	m1 := p1.Send() // p1 = 3
	require.Equal(t, uint64(3), m1)
	require.Equal(t, uint64(4), p2.Receive(m1)) // p2 = max(0,3)+1 = 4

	m2 := p2.Send() // p2 = 5
	require.Equal(t, uint64(5), m2)
	require.Equal(t, uint64(6), p3.Receive(m2)) // p3 = max(0,5)+1 = 6

	m3 := p3.Send() // p3 = 7
	require.Equal(t, uint64(7), m3)
	require.Equal(t, uint64(8), p1.Receive(m3)) // p1 = max(3,7)+1 = 8
}

func TestConcurrentTicks(t *testing.T) {
	// 1000 goroutines each call Tick once. With correct locking, every
	// tick contributes exactly one increment, so the final counter is
	// exactly 1000 and no two ticks return the same value.
	//
	// Run this under `go test -race` to catch missing locking.
	const N = 1000
	c := New()

	// We collect every value returned by Tick into a set and assert
	// there are no duplicates. This is a stronger guarantee than just
	// "final counter == N": it proves no two ticks observed the same
	// counter value mid-flight.
	var (
		mu   sync.Mutex
		seen = make(map[uint64]struct{}, N)
		wg   sync.WaitGroup
	)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := c.Tick()
			mu.Lock()
			seen[v] = struct{}{}
			mu.Unlock()
		}()
	}
	wg.Wait()

	require.Equal(t, uint64(N), c.Now(), "final counter must equal number of ticks")
	require.Len(t, seen, N, "every tick must return a unique value")
}

func TestConcurrentSendAndReceive(t *testing.T) {
	// Mix Send and Receive concurrently. We can't predict the exact final
	// value (depends on scheduling), but we can assert it's at least the
	// number of operations, since every op contributes >= 1 increment.
	const N = 500
	c := New()

	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); c.Send() }()
		go func(ts uint64) { defer wg.Done(); c.Receive(ts) }(uint64(i))
	}
	wg.Wait()

	require.GreaterOrEqual(t, c.Now(), uint64(2*N))
}
