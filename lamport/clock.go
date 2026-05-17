package lamport

import "sync"

// A Lamport logical clock. The 0 value is not reusable; obtain a clock from New or NewAt.
// All methods are safe for concurrent use.
type Clock struct {
	mu      sync.Mutex
	counter uint64
}

// Returns a clock whose counter starts at 0. The first event observed (via Tick, Send, Receive)
// will have timestamp 1.
func New() *Clock {
	return &Clock{}
}

// Returns a Clock whose counter start at the given value. This is the constructor to use when
// restoring a clock from persisted state: the counter resumes where it left off, preserving the
// monotonicity guarantee across restarts.
func NewAt(start uint64) *Clock {
	return &Clock{counter: start}
}

// Records an internal event and resumes its timestamp. An "internal event" is any local event that
// is not a message send or receive. For e.g., updating a local state, performing a computation, or
// writing a log entry.
func (c *Clock) Tick() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter++
	return c.counter
}

// Records a message-sent event and returns the timestamp to attach to the outgoing message.
// This is identical to Tick; provided as a distinct method so call sites read intentionally
// and so future instrumentation can distinguish sends from internal events.
func (c *Clock) Send() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter++
	return c.counter
}

// This records the receipt of a message carrying timestamp remote and returns this clock's new timestamp.
// The new value is max(local, remote) + 1. The +1 is what enforces the causal-order guarantee: even if
// local already exceeds remote, the receive event must be strictly later than the send.
func (c *Clock) Receive(remote uint64) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter = max(c.counter, remote) + 1
	return c.counter
}

// Returns the Clock's current counter without modifying it. Inspection is not itself a Lamport event;
// Now exists for logging, metrics and tests.
func (c *Clock) Now() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counter
}

// Overwrites the counter with the specified value. It is used for testing purposes.
func (c *Clock) Reset(to uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter = to
}
