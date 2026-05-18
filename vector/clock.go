package vector

import (
	"maps"
	"sync"
)

// This represents the vector clock owned by one process. The zero value is not usable;
// obtain a clock from New or NewFromMap.
//
// All methods are safe for concurrent use.
type Clock struct {
	mu     sync.RWMutex
	nodeID string
	vec    map[string]uint64
}

// Returns a fresh clock for the given node ID with all counters at 0.
// The node ID identifies "this" process's slot in the vector. It must be unique within the system;
// if two processes pick the same node ID, vector arithmetic produces nonsense.
func New(nodeID string) *Clock {
	return &Clock{
		nodeID: nodeID,
		vec:    make(map[string]uint64),
	}
}

// Returns a Clock for nodeID initialized with the given counters.
// The initial map is copied — the caller can mutate it after the call without affecting the Clock.
func NewFromMap(nodeID string, initial map[string]uint64) *Clock {
	vec := make(map[string]uint64, len(initial))
	maps.Copy(vec, initial)
	return &Clock{
		nodeID: nodeID,
		vec:    vec,
	}
}

// Returns this clock's node identifier.
func (c *Clock) NodeID() string {
	return c.nodeID
}

// Records an internal event by incrementing this node's own entry, and returns a snapshot of the
// vector after the increment
func (c *Clock) Tick() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.vec[c.nodeID]++
	return c.snapshotLocked()
}

// Records a message-sent event and returns the Snapshot to attach to the outgoing message.
//
// This is algorithmically identical to Tick. The separate method exists so call sites read
// intentionally and so future instrumentation can distinguish sends from internal events.
func (c *Clock) Send() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.vec[c.nodeID]++
	return c.snapshotLocked()
}

// Merges the incoming snapshot into this clock and records the receive event by incrementing
// this node's own entry. Returns the resulting snapshot.
//
// The merge takes the element-wise maximum of the two vectors, which absorbs the sender's knowledge
// of every process. Counters not present in either vector are treated as zero.
func (c *Clock) Receive(remote Snapshot) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	for node, rv := range remote.vec {
		if rv > c.vec[node] {
			c.vec[node] = rv
		}
	}
	c.vec[c.nodeID]++
	return c.snapshotLocked()
}

// Returns the clock's current snapshot without modifying it. We use this for inspection or logging;
// observation itself is not a vector event.
func (c *Clock) Now() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshotLocked()
}

// Builds an immutable snapshot from the current vector. Caller must hold c.mu (read/write).
func (c *Clock) snapshotLocked() Snapshot {
	cp := make(map[string]uint64, len(c.vec))
	for k, v := range c.vec {
		cp[k] = v
	}
	return Snapshot{
		nodeID: c.nodeID,
		vec:    cp,
	}
}

// This is an immutable point-in-time view of a vector clock. It is safe to store, send and compare;
// mutating the originating clock has no effect on a Snapshot created before the mutation.
type Snapshot struct {
	nodeID string
	vec    map[string]uint64
}

// Returns the node ID that produced this snapshot
func (s Snapshot) NodeID() string {
	return s.nodeID
}

// Returns the counter for the given node, or 0 if no entry exists.
func (s Snapshot) Get(nodeID string) uint64 {
	return s.vec[nodeID]
}

// Returns the set of node IDs that have non-zero entries in this Snapshot. The order is unspecified.
func (s Snapshot) Nodes() []string {
	out := make([]string, 0, len(s.vec))
	for k := range s.vec {
		out = append(out, k)
	}
	return out
}

// Reports whether two Snapshots have identical vectors (ignoring node ID; two Snapshots from different
// originating clocks can still be Equal if their vectors match).
func (s Snapshot) Equal(other Snapshot) bool {
	if len(s.vec) != len(other.vec) {
		return false
	}
	for k, v := range s.vec {
		if other.vec[k] != v {
			return false
		}
	}
	return true
}

// Returns a deep copy of the Snapshot. Snapshots are already immutable by construction, so Copy is rarely needed;
// it exists for cases where a caller wants to be paranoid about aliasing.
func (s Snapshot) Copy() Snapshot {
	cp := make(map[string]uint64, len(s.vec))
	for k, v := range s.vec {
		cp[k] = v
	}
	return Snapshot{nodeID: s.nodeID, vec: cp}
}
