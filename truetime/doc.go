// This package implements a bounded-uncertainty clock in the style of Google Spanner (Corbett et al., 2012).
// Where every other chrono clock returns a single timestamp, TrueTime returns an interval [Earliest, Latest]
// guaranteed to contain true global time. The uncertainty is exposed as a first-class quantity in the API.
//
// # External consistency via commit-wait
//
// The interval API exists to support external consistency: the property that if transaction T1 commits before T2
// starts in real time, then T1's commit timestamp must be strictly less than T2's. Achieving this with skewed wall
// clocks is impossible without exposing the uncertainty - TrueTime does, and CommitWait uses it:
//
//  1. T1 picks commit time: s = clock.Now().Latest.
//  2. T1 performs commit work.
//  3. T1 calls CommitWait(ctx, clock, s), blocks until clock.Now() is provably past s.
//  4. T1 becomes visible. Any T2 that starts after this point will sample an interval strictly after s.
//
// # Sources
//
// A Clock is parameterized by an UncertaintySource. This package provides:
//
//   - NTPSource: queries an NTP server periodically; uses the reported offset and root dispersion to compute the interval.
//     Intervals widen between queries to account for local clock drift.
//   - FakeSource: lets tests set the interval explicitly. Used by the proofs/ package to reproduce Spanner-class scenarios.
//
// # Concurrency
//
// *Clock and its sources are safe for concurrent use. Interval is a small value type and is safe to copy.
package truetime
