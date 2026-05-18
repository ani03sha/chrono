// This implements a simplified dotted version vector (DVV) as described by Preguiça et al. (2010),
// used by Riak and similar leaderless key-value stores for sibling detection.
//
// A Version is a set of "dots" — (actor, counter) pairs: representing the causal history of a stored value.
// Two Versions are compared with Descends, Dominates, Equals, and Concurrent.
//
// The Concurrent case is the operational core: when a store detects concurrent writes, it must preserve both values
// as siblings rather than silently picking one.
//
// # Difference from vector
//
// Structurally Version is map[string]uint64: identical to a vector Snapshot. The differences are:
//
//   - Semantics: each entry is an "actor"'s highest dot counter, not a "process"'s logical time.
//     Actors are typically clients or storage nodes; they can join and leave dynamically.
//   - Immutability: Increment and Merge return new Versions instead of mutating. Versions are attached
//     to stored values and must not change after the value is written.
//   - API surface: methods are named for sibling-resolution patterns (Descends, Dominates) rather than
//     for process clocks (Compare).
//
// # Concurrency
//
// Versions are immutable, so passing them across goroutines requires no locking.
//
// # Example
//
//	v1 := dotted.NewVersion().Increment("clientA")  // {A:1}
//	v2 := v1.Increment("clientA")                   // {A:2}
//	v3 := v1.Increment("clientB")                   // {A:1, B:1}
//
//	v2.Descends(v1)    // true  — v2 is causally newer
//	v3.Concurrent(v2)  // true  — clientA and clientB both wrote from v1
package dotted
