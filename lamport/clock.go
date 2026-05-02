package lamport

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
)

// This error is returned when a timestamp is unmarshaled from a byte slice of the wrong length.
// Wire format is exactly 8 bytes.
var ErrInvalidTimestamp = errors.New("lamport: invalid timestamp encoding")

// Lamport logical clock. All methods are safe for concurrent use.
// The zero value is a valid empty clock equivalent to New()
type Clock struct {
	mu      sync.Mutex
	counter uint64
}

// Returns a Lamport clock starting at 0.
func New() *Clock {
	return &Clock{}
}

// Returns a Lamport clock initialized to start. We should use this when restoring a clock from durable
// storage so that the following events have timestamps strictly greater than the every stored event.
func NewAt(start uint64) *Clock {
	return &Clock{counter: start}
}

// Records and internal event and returns the new logical time.
func (c *Clock) Tick() uint64 {
	c.mu.Lock()
	c.counter++
	v := c.counter
	c.mu.Unlock()
	return v
}

// Records the act of sending a message and returns the timestamp the caller should attach to that message.
// It is similar to Tick() but kept distinct so that call sites read a documentation of intent: "this event
// is send" vs. "this event is internal".
func (c *Clock) Send() uint64 {
	c.mu.Lock()
	c.counter++
	v := c.counter
	c.mu.Unlock()
	return v
}

// Records receipt of a message stamped with remoteTs and returns the new local logical time.
// The rule is: counter = max(remoteTs, counter) + 1. Even when remoteTs is less than counter,
// the counter still advances by one, because the receive itself is an event.
func (c *Clock) Receive(remoteTs uint64) uint64 {
	c.mu.Lock()
	if remoteTs > c.counter {
		c.counter = remoteTs
	}
	c.counter++
	v := c.counter
	c.mu.Unlock()
	return v
}

// Returns the logical time without recording an event. We should use this function only for read-only
// inspection (logging, metrics). Events that are part of causality must use Tick, Send, or Receive because
// those are the only methods that advance the clock.
func (c *Clock) Now() uint64 {
	c.mu.Lock()
	v := c.counter
	c.mu.Unlock()
	return v
}

// Resets the counter to "to". This is relevant for tests that need to put a clock to a known state.
// Production code should never call Reset: Lamport clocks are append-only by design, and resetting one
// breaks the causal guarantee of every timestamp issued before the reset.
func (c *Clock) Reset(to uint64) {
	c.mu.Lock()
	c.counter = to
	c.mu.Unlock()
}

// This is a Lamport timestamp suitable for serialization across the wire. It implements encoding.BinaryMarshaler
// and BinaryUnmarshaler so it works transparently with gob, and the binary form (8 bytes, big-endian) is stable
// across architectures and Go versions.
type Timestamp uint64

// Encodes the timestamp as 8 big-endian bytes.
func (t Timestamp) MarshalBinary() ([]byte, error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(t))
	return b, nil
}

// Decodes the timestamp from exactly 8 big-endian bytes.
func (t *Timestamp) UnmarshalBinary(data []byte) error {
	if len(data) != 8 {
		return fmt.Errorf("%w: got %d bytes, want 8", ErrInvalidTimestamp, len(data))
	}
	*t = Timestamp(binary.BigEndian.Uint64(data))
	return nil
}
