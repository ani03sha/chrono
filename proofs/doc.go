// This package reproduces specific failure scenarios from the distributed systems literature using chrono primitives.
// Each test references the original paper or production incident, sets up the exact scenario described, asserts the
// expected outcome, and (where relevant) demonstrates what would have happened with a naive alternative (raw wall clock,
// last-write-wins, no commit-wait).
//
// These are not unit tests of chrono: those live in each clock's own package. They are executable evidence that chrono  correctly handles
// the scenarios its types are designed to handle.
//
// Run with `make proofs` (verbose) or `go test ./proofs/...`.
//
// # Scenarios
//
//   - TestLamport1978: Lamport (1978), "Time, Clocks, and the Ordering of Events in a Distributed System."
//     Three-process ring producing strictly increasing logical timestamps along every causal chain.
//
//   - TestNTPStepCausesIncorrectOrderingWithWallClock: Demonstrates that sorting events by raw wall time produces
//     wrong results under an NTP backward step, then shows the same scenario sorted by HLC stamps produces the correct order.
//
//   - TestSpannerExternalConsistency — Corbett et al. (2012), "Spanner: Google's Globally-Distributed Database."
//     Commit-wait on a TrueTime interval achieves external consistency: any transaction starting after T1's commit
//     gets a strictly larger timestamp than T1.
//
//   - TestDottedVersionDetectsConcurrentWrites — Preguiça et al. (2010). Two clients writing concurrently from the same
//     starting state produce sibling versions; the negative control test demonstrates what last-write-wins discards.
//
//   - TestHLCPreservesCausalityAcrossSkewedNodes — Kulkarni et al. (2014), as deployed in CockroachDB and others. HLC's receive
//     algorithm preserves causality across nodes whose wall clocks diverge by tens of milliseconds.
package proofs
