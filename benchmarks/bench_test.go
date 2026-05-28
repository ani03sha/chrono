// This package holds performance benchmarks for chrono's clock primitives.
// Run with `make bench` (writes results to BENCHMARKS.md) or `go test -bench=. -benchmem ./benchmarks/...`.
//
// Targets are documented per-benchmark in the source. A regression past target should be treated as a bug or a
// deliberate trade-off that needs to be reflected in the README's benchmark table.
package benchmarks

import (
	"fmt"
	"testing"
	"time"

	"github.com/ani03sha/chrono/hlc"
	"github.com/ani03sha/chrono/internal/wallclock"
	"github.com/ani03sha/chrono/lamport"
	"github.com/ani03sha/chrono/truetime"
	"github.com/ani03sha/chrono/vector"
)

// nodesMap builds an n-entry map of "<prefix><i>" → baseValue+i, used to construct vector clocks pre-populated
// with a known node set.
func nodesMap(prefix string, n int, baseValue uint64) map[string]uint64 {
	out := make(map[string]uint64, n)
	for i := 0; i < n; i++ {
		out[fmt.Sprintf("%s%d", prefix, i)] = baseValue + uint64(i)
	}
	return out
}

// ────────────────────────── Wallclock baseline ──────────────────────────
//
// Establishes the floor: any clock built on top of the wall clock will pay at least this much per Now() call.
// Roughly time.Now().UnixNano().

func BenchmarkWallclockReal(b *testing.B) {
	w := wallclock.Real()
	b.ReportAllocs()
	b.ResetTimer()
	var sink int64
	for i := 0; i < b.N; i++ {
		sink = w.NowNanos()
	}
	_ = sink
}

// ────────────────────────── Lamport ─────────────────────────────────────
//
// Target for all three: < 60 ns/op, 0 allocs/op. Pure uint64 math under a single sync.Mutex.

func BenchmarkLamportTick(b *testing.B) {
	c := lamport.New()
	b.ReportAllocs()
	b.ResetTimer()
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink = c.Tick()
	}
	_ = sink
}

func BenchmarkLamportSend(b *testing.B) {
	c := lamport.New()
	b.ReportAllocs()
	b.ResetTimer()
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink = c.Send()
	}
	_ = sink
}

func BenchmarkLamportReceive(b *testing.B) {
	c := lamport.New()
	b.ReportAllocs()
	b.ResetTimer()
	var sink uint64
	for i := 0; i < b.N; i++ {
		sink = c.Receive(uint64(i))
	}
	_ = sink
}

// ────────────────────────── Vector ──────────────────────────────────────
//
// Vector operations cannot be zero-allocation because they return immutable Snapshots that copy the underlying map.
// We measure them anyway so an *extra* allocation regression is detectable.

// Target: < 200 ns/op.
func BenchmarkVectorTick3Nodes(b *testing.B) {
	c := vector.NewFromMap("n0", nodesMap("n", 3, 1))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Tick()
	}
}

// Target: < 1 µs/op.
func BenchmarkVectorReceive10Nodes(b *testing.B) {
	local := vector.NewFromMap("n0", nodesMap("n", 10, 1))
	remote := vector.NewFromMap("n0", nodesMap("n", 10, 100)).Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		local.Receive(remote)
	}
}

// Target: < 200 ns/op, 0 allocs/op (Compare reads, doesn't allocate).
func BenchmarkVectorCompare10Nodes(b *testing.B) {
	a := vector.NewFromMap("n0", nodesMap("n", 10, 1)).Now()
	other := vector.NewFromMap("n0", nodesMap("n", 10, 1)).Now()
	b.ReportAllocs()
	b.ResetTimer()
	var rel vector.Relation
	for i := 0; i < b.N; i++ {
		rel = vector.Compare(a, other)
	}
	_ = rel
}

// Target: < 500 ns/op, 1 alloc (the returned byte slice).
func BenchmarkVectorMarshal10Nodes(b *testing.B) {
	snap := vector.NewFromMap("n0", nodesMap("n", 10, 1)).Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = snap.MarshalBinary()
	}
}

// Companion to Marshal — should be in the same ballpark.
func BenchmarkVectorUnmarshal10Nodes(b *testing.B) {
	snap := vector.NewFromMap("n0", nodesMap("n", 10, 1)).Now()
	data, _ := snap.MarshalBinary()
	b.ReportAllocs()
	b.ResetTimer()
	var out vector.Snapshot
	for i := 0; i < b.N; i++ {
		_ = (&out).UnmarshalBinary(data)
	}
}

// ────────────────────────── HLC ─────────────────────────────────────────
//
// Targets: < 150 ns/op, 0 allocs/op. The wall clock read is the dominant cost: most of the per-call time is time.Now()
// under the hood, plus the mutex and the four-case switch.

func BenchmarkHLCNow(b *testing.B) {
	c := hlc.New(wallclock.Real())
	b.ReportAllocs()
	b.ResetTimer()
	var sink hlc.Timestamp
	for i := 0; i < b.N; i++ {
		sink = c.Now()
	}
	_ = sink
}

func BenchmarkHLCSend(b *testing.B) {
	c := hlc.New(wallclock.Real())
	b.ReportAllocs()
	b.ResetTimer()
	var sink hlc.Timestamp
	for i := 0; i < b.N; i++ {
		sink = c.Send()
	}
	_ = sink
}

func BenchmarkHLCReceive(b *testing.B) {
	c := hlc.New(wallclock.Real())
	remote := c.Now()
	b.ReportAllocs()
	b.ResetTimer()
	var sink hlc.Timestamp
	for i := 0; i < b.N; i++ {
		sink, _ = c.Receive(remote)
	}
	_ = sink
}

// Companion: 12-byte allocation per call is the absolute minimum for a binary timestamp encoding.
// We want to verify there's nothing extra.
func BenchmarkHLCMarshalBinary(b *testing.B) {
	ts := hlc.Timestamp{WallTime: 1_700_000_000_000_000_000, Logical: 42}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ts.MarshalBinary()
	}
}

// ────────────────────────── TrueTime ────────────────────────────────────
//
// FakeSource is the deterministic source — its cost is just an RLock and a struct copy.
// NTPSource cost is dominated by the periodic network query (not measured here; not a hot-path operation).

func BenchmarkTrueTimeNowFake(b *testing.B) {
	base := time.Unix(1_700_000_000, 0)
	src := truetime.NewFakeSource(truetime.Interval{
		Earliest: base,
		Latest:   base.Add(7 * time.Millisecond),
	})
	c := truetime.New(src)
	b.ReportAllocs()
	b.ResetTimer()
	var sink truetime.Interval
	for i := 0; i < b.N; i++ {
		sink = c.Now()
	}
	_ = sink
}
