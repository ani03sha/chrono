package lamport_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/ani03sha/chrono/lamport"
	"github.com/stretchr/testify/require"
)

func TestNewStartsAtZero(t *testing.T) {
	c := lamport.New()
	require.Equal(t, uint64(0), c.Now())
}

func TestNewAt(t *testing.T) {
	c := lamport.NewAt(67)
	require.Equal(t, uint64(67), c.Now())
}

func TestTickIncrements(t *testing.T) {
	c := lamport.New()

	require.Equal(t, uint64(1), c.Tick())
	require.Equal(t, uint64(2), c.Tick())
	require.Equal(t, uint64(3), c.Tick())
	require.Equal(t, uint64(3), c.Now(), "Now must not advance the clock")
}

func TestSendIncrements(t *testing.T) {
	c := lamport.New()

	require.Equal(t, uint64(1), c.Send())
	require.Equal(t, uint64(2), c.Send())
}

func TestReceiveTakesMaxPlusOne(t *testing.T) {
	c := lamport.NewAt(2)

	got := c.Receive(5)

	require.Equal(t, uint64(6), got, "Receive(5) on counter=2 should return max(2,5)+1=6")
	require.Equal(t, uint64(6), c.Now())
}

func TestReceiveSmallerStillIncrements(t *testing.T) {
	c := lamport.NewAt(5)

	// remoteTs is less than the local counter; max stays at local but the
	// counter still advances because the receive itself is an event.
	got := c.Receive(1)

	require.Equal(t, uint64(6), got)
	require.Equal(t, uint64(6), c.Now())
}

func TestReceiveEqualStillIncrements(t *testing.T) {
	c := lamport.NewAt(5)
	require.Equal(t, uint64(6), c.Receive(5))
}

func TestReset(t *testing.T) {
	c := lamport.NewAt(100)
	c.Reset(0)
	require.Equal(t, uint64(0), c.Now())
}

func TestConcurrentTicks(t *testing.T) {
	const (
		goroutines        = 100
		ticksPerGoroutine = 10
	)

	c := lamport.New()

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < ticksPerGoroutine; j++ {
				c.Tick()
			}
		}()
	}
	wg.Wait()

	require.Equal(t, uint64(goroutines*ticksPerGoroutine), c.Now(),
		"every concurrent Tick must increment exactly once")
}

func TestTimestampMarshalBinaryRoundtrip(t *testing.T) {
	cases := []lamport.Timestamp{
		0, 1, 42, 65535,
		1 << 32,
		1<<64 - 1, // max uint64
	}

	for _, want := range cases {
		b, err := want.MarshalBinary()
		require.NoError(t, err)
		require.Len(t, b, 8, "Lamport timestamp encoding must be exactly 8 bytes")

		var got lamport.Timestamp
		require.NoError(t, got.UnmarshalBinary(b))
		require.Equal(t, want, got)
	}
}

func TestTimestampMarshalIsBigEndian(t *testing.T) {
	b, err := lamport.Timestamp(1).MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, 1}, b)
}

func TestTimestampUnmarshalRejectsWrongLength(t *testing.T) {
	var ts lamport.Timestamp

	err := ts.UnmarshalBinary([]byte{1, 2, 3})

	require.Error(t, err)
	require.True(t, errors.Is(err, lamport.ErrInvalidTimestamp),
		"error must wrap ErrInvalidTimestamp so callers can match with errors.Is")
}

func ExampleClock() {
	a := lamport.New()
	b := lamport.New()

	a.Tick() // a = 1
	a.Tick() // a = 2

	ts := a.Send() // a = 3, message carries ts = 3
	b.Receive(ts)  // b = max(0, 3) + 1 = 4

	// After this exchange, every event with timestamp < 4 happens-before
	// any future event on b.

	fmt.Println("a =", a.Now())
	fmt.Println("b =", b.Now())

	// Output:
	// a = 3
	// b = 4
}
