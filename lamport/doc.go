// This package implements a Lamport logical clock as described in Leslie Lamport's 1978 paper
// "Time, Clocks, and the Ordering of Events in a Distributed System".
//
// A Lamport clock assigns a monotonically increasing integer timestamp to every event in a
// distributed system such that if event a causally precedes event b (a → b), then timestamp(a) < timestamp(b).
//
// The converse does NOT hold: two events with ts(a) < ts(b) may be concurrent.
// To detect concurrency, use the vector package instead.
//
// # Algorithm
//
// Each process owns one Clock. Three operations update it:
//
//	internal event:     counter++
//	send a message:     counter++; attach counter to the message
//	receive a message:  counter = max(counter, remote) + 1
//
// The +1 on receive guarantees that the receive event has a strictly larger timestamp than the corresponding
// send, even if the local counter was already ahead.
package lamport
