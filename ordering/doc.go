// This package provides generic happens-before machinery that works uniformly across chrono's clock types.
// It is the glue layer that lets you write "sort these events causally" once instead of once per clock.
//
// # The Comparable interface
//
// Comparable abstracts two questions: does stamp a happen before stamp b, and are they concurrent?
// Three adapters in this package make chrono's clocks satisfy it:
//
//   - LamportStamp: wraps a Lamport timestamp.
//   - HLCStamp: wraps an HLC Timestamp.
//   - VectorStamp: wraps a vector Snapshot.
//
// Only VectorStamp provides the exact happens-before relation. Lamport and HLC are total-ordered, so HappensBefore
// is exact (strict less-than), but Concurrent is sound-but-incomplete: equal stamps are guaranteed concurrent,
// but events with different stamps may also be concurrent without any way to detect it.
//
// TrueTime intentionally has no adapter — TrueTime intervals describe uncertainty about wall time, not causality,
// and mashing them through HappensBefore would be a category error.
//
// # Event utilities
//
// Event[T Comparable] is a tiny generic carrier for an ID, a stamp, and an opaque Data payload. The three utility functions are:
//
//   - TopologicalSort: total ordering respecting the partial order (Kahn's algorithm; ID tiebreaks among concurrent events).
//   - MaximalElements: events with nothing causally after them.
//   - MinimalElements: events with nothing causally before them.
package ordering
