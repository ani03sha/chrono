// This package implements a Hybrid Logical Clock (HLC) as described by  Kulkarni et al. (2014),
// "Logical Physical Clocks and Consistent Snapshots in Globally Distributed Databases".
//
// HLCs are used by CockroachDB, YugabyteDB, and MongoDB to order events across nodes without requiring tightly
// synchronized wall clocks.
//
// An HLC produces timestamps with two components:
//
//   - WallTime: nanoseconds since the Unix epoch —> approximates physical time.
//   - Logical: a uint32 counter that breaks ties within the same WallTime.
//
// Timestamps form a total order (compare WallTime, then Logical), so callers can use them as commit timestamps
// and sort events globally. Unlike raw wall clocks, HLC timestamps are:
//
//   - Monotonic: an HLC's output never goes backward, even if the  underlying wall clock is stepped backward by NTP.
//   - Causal: if event a precedes event b (locally or by send/receive), then HLC(a) < HLC(b).
//   - Close to physical time: the WallTime stays within a small bound of the wall clock under normal operation.
//
// # Algorithm summary
//
// On any local event (Now or Send):
//
//	pt = wall.Now()
//	l' = max(l, pt)
//	c  = (c+1 if l'==l else 0)
//	l  = l'
//
// On Receive of (remote.l, remote.c):
//
//	pt = wall.Now()
//	l' = max(l, remote.l, pt)
//	if  l' == l == remote.l: c = max(c, remote.c) + 1
//	elif l' == l:             c = c + 1
//	elif l' == remote.l:      c = remote.c + 1
//	else:                     c = 0
//	l  = l'
//
// # Drift protection
//
// NewWithMaxDrift rejects incoming timestamps whose WallTime is more than maxDrift ahead of the local wall clock.
// Without this guard, a single misbehaving sender can permanently warp every receiver's clock by sending a far-future
// timestamp.
//
// # Concurrency
//
// *Clock is safe for concurrent use. Timestamp is a small value type safe to copy and pass by value.
package hlc
