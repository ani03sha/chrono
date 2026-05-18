// This vector implements a vector clock as described in Fidge (1988) and Mattern (1989).
//
// Vector clocks extend Lamport timestamps by attaching one counter per process to every event,
// which makes it possible not just to order causally related events but also to *detect* events
// that have no causal relation at all (concurrent events).
//
// # Algorithm
//
// Each process i maintains a vector V_i indexed by node ID. V_i[j] is i's
// best estimate of the most recent event from process j that i has heard about.
//
// Three operations update the vector:
//
//	internal event:    V_i[i]++
//	send a message:    V_i[i]++; attach a copy of V_i to the message
//	receive vector W:  V_i[k] = max(V_i[k], W[k]) for all k; then V_i[i]++
//
// # Comparison
//
// Two Snapshots can be compared with Compare, which returns one of four Relations: Before, After, Equal, or Concurrent.
// The Concurrent case is what vector clocks add over Lamport clocks: an explicit signal that two events have no causal
// relationship, which is the signal a key-value store needs to surface conflicting writes as siblings instead of silently
// dropping one.
//
// # Concurrency
//
// *Clock is safe for concurrent use. Snapshots are immutable; they own their own copy of the vector —
// so it is safe to pass a Snapshot to another goroutine, serialize it, or compare it long after the
// originating Clock has continued ticking.
//
// # Example
//
//	c1 := vector.New("n1")
//	c2 := vector.New("n2")
//
//	c1.Tick()            // V_c1 = {n1:1}
//	msg := c1.Send()     // V_c1 = {n1:2}; msg snapshot = {n1:2}
//	c2.Receive(msg)      // V_c2 = {n1:2, n2:1}
//
//	rel := vector.Compare(c1.Now(), c2.Now())
//	// rel == vector.Before (c1's vector is dominated by c2's)
package vector
