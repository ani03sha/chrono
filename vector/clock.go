package vector

import (
	"sync"
)

// This is a vector logical clock. Each clock represents a single node's view of the system.
// The clock holds a per-node counter recording how many events the local node has observed
// at every other node it has heard from.
//
// All methods are safe for concurrent use.
type Clock struct {
	mu     sync.RWMutex
	nodeID string
	vec    map[string]uint64
}

// Returns a vector clock for nodeID with no observed events.
func New(nodeID string) *Clock {
	if nodeID == "" {
		panic("vector.New: nodeID must be non-empty")
	}
	return &Clock{
		nodeID: nodeID,
		vec:    make(map[string]uint64),
	}
}

func NewFromMap(nodeID string, initial map[string]uint64) *Clock {
	if nodeID == "" {
		panic("vector.NewFromMap: nodeID must be non-empty")
	}
	vec := make(map[string]uint64, len(initial))
	for k, v := range initial {
		vec[k] = v
	}
	return &Clock{
		nodeID: nodeID,
		vec:    vec,
	}
}

// Returns the clock's owning identifier
func (c *Clock) NodeID() string {
	return c.nodeID
}

// Records an internal event and returns the snapshot of the clock after the increment.
// Rule: vector[self]++
func (c *Clock) Tick() Snapshot {
	c.mu.Lock()
	c.vec[c.nodeID]++
	s := c.snapshotLocked()
	c.mu.Unlock()
	return s
}

// Sends an outbound message and returns the snapshot the caller should attach to the message.
// Same behavior as Tick() but kept distinct so call sites read as documentation of intent.
func (c *Clock) Send() Snapshot {
	c.mu.Lock()
	c.vec[c.nodeID]++
	s := c.snapshotLocked()
	c.mu.Unlock()
	return s
}

// Merges remote into the logical vector, increments the local component, and returns the resulting snapshot.
// The merge is element-wise max.The increment ensures the receive event is strictly later than any prior event
// on either side of the message.
func (c *Clock) Receive(remote Snapshot) Snapshot {
	c.mu.Lock()
	for node, rv := range remote.vec {
		if rv > c.vec[node] {
			c.vec[node] = rv
		}
	}
	c.vec[c.nodeID]++
	s := c.snapshotLocked()
	c.mu.Unlock()
	return s
}

// Returns a snapshot of the current state without recording an event. We should use it only for read-only
// inspection; events that participate in causality must use Tick, Send, or Receive.
func (c *Clock) Now() Snapshot {
	c.mu.RLock()
	s := c.snapshotLocked()
	c.mu.RUnlock()
	return s
}

// Returns an independent copy of the clock's state.
func (c *Clock) snapshotLocked() Snapshot {
	vec := make(map[string]uint64, len(c.vec))
	for k, v := range c.vec {
		vec[k] = v
	}
	return Snapshot{nodeID: c.nodeID, vec: vec}
}

// Immutable view of a vector clock at a single point in time.
// Snapshots are values: copying one is cheap and gives an independent reference.
type Snapshot struct {
	nodeID string
	vec    map[string]uint64
}

// Returns the identifier of the node that produced the snapshot.
func (s Snapshot) NodeID() string {
	return s.nodeID
}

// Returns the counter for nodeID, or 0 if absent.
func (s Snapshot) Get(nodeID string) uint64 {
	return s.vec[nodeID]
}

// Returns the node identifiers known to this snapshot, in unspecified order.
// The slice is freshly allocated; callers may mutate it without affecting the snapshot.
func (s Snapshot) Nodes() []string {
	nodes := make([]string, 0, len(s.vec))
	for k := range s.vec {
		nodes = append(nodes, k)
	}
	return nodes
}

// Reports structural equality: same nodeID and same vec entries. For causal equality (same vec, ignoring nodeID),
// use Compare and check for the Equal relation.
func (s Snapshot) Equal(other Snapshot) bool {
	if s.nodeID != other.nodeID || len(s.vec) != len(other.vec) {
		return false
	}
	for k, v := range s.vec {
		if other.vec[k] != v {
			return false
		}
	}
	return true
}

// Returns an independent copy of the snapshot. Snapshots are designed to be immutable, but Copy exists for code that
// needs a defensive duplicate before passing the snapshot to an unsafe encoder or external library.
func (s Snapshot) Copy() Snapshot {
	vec := make(map[string]uint64, len(s.vec))
	for k, v := range s.vec {
		vec[k] = v
	}
	return Snapshot{nodeID: s.nodeID, vec: vec}
}
