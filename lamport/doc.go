// This package implements a Lamport logical clock as described in Leslie Lamport's lengendary paper:
// "Time, Clocks, and the Ordering of Events in a Distributed System" (Communications of the ACM, 1978).
//
// A Lamport clock assigns a monotonically increasing integer to every event in a distributed system.
// The assignment guarantees that if event a happens-before event b (a → b), then C(a) < C(b).
// The converse is not true: C(a) < C(b) does not imply a → b: the events may be concurrent and their timestamps
// merely happen to differ.
//
// The algorithm is simple:
//
//   - On an internal event:        counter++
//   - On sending a message:        counter++; attach counter to message
//   - On receiving message ts r:   counter = max(counter, r) + 1
//
// Lamport clocks establish a partial order consistent with causality but cannot detect concurrency.
// We need to use vector clocks (chrono/vector) when we need to distinguish "a happened before b" from "a and b are concurrent".
//
// All methods on Clock are safe for concurrent use.
package lamport
