// Package vector implements a vector logical clock: Mattern (1989) and Fidge (1988).
//
// A vector clock generalizes a Lamport clock by tracking each node's view of every other node's logical time.
// This results in a partial order that distinguishes three relations between events:
//
//  1. happens-before: one event causally precedes the other
//  2. equal:          the two events have identical observed history
//  3. concurrent:     neither event causally precedes the other
//
// Lamport clocks cannot distinguish concurrent events from causally ordered ones.
// Vector clocks can, at the cost of O(N) space per timestamp where N is the number of nodes that have ever been observed.
//
// Use Compare to determine the relation between two snapshots.
//
// All Clock methods are safe for concurrent use. Snapshots are immutable values: their internal state is not shared
// with the originating Clock, so a Clock advancing in time does not invalidate any snapshot taken from it.
package vector
