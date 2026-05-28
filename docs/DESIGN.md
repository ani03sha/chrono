# Design rationale

The non-obvious API choices in chrono, with the reasoning behind them.

## Time injection via the wallclock interface

`hlc` and `truetime` depend on the wall clock. Rather than calling
`time.Now()` directly, they accept a `wallclock.Clock` interface. This
exists so tests can drive time deterministically: the `proofs/`
package uses `wallclock.NewFake` to simulate NTP backward steps, the
`hlc/` unit tests step the clock to exercise every branch of the
receive algorithm, and so on.

The interface lives under `internal/` because it's an implementation
detail of chrono's clocks, not part of the public API. Users pass
`wallclock.Real()` once at construction and forget about it. We don't
want library consumers building on top of `wallclock` as a generic time
abstraction; there are better libraries for that.

## Immutable Snapshots and Versions

`vector.Snapshot` and `dotted.Version` are both immutable by design.
You can't mutate them through their public API, and the underlying
maps are unexported.

Why: snapshots and versions get attached to *values*. A KV store
writes `{value, version}` to disk; if the version were mutable, a later
clock advance would corrupt the stored value's recorded history.
Immutability is what makes "store this value with this version" safe.

The cost is one map copy per snapshot. We pay it because the
alternative (copy-on-write with reference counting) adds complexity
that isn't worth it at our scale.

## The four-case HLC receive

`hlc.Receive` uses a `switch` on equality conditions rather than a
chain of `if/else`. This is deliberate. The four cases are mutually
exclusive once `l' = max(l, remote.l, pt)` is computed, and the switch
makes that structure visible. A reader can see at a glance that
exactly one branch fires per call, and the order of cases documents
the precedence (`l' == l == remote.l` is the most specific and must be
checked first).

## Vector clocks use `map[string]uint64`, not `[]uint64`

The alternative would be a fixed-size slice indexed by integer node
ID. That's faster but requires every process to agree on a static node
roster at startup. Adding a node is operationally painful.

The map representation accommodates dynamic membership at the cost of
some performance. Real systems like Riak chose the map for the same
reason, and the benchmark numbers (~130 ns/op for 3-node Tick, ~300
ns/op for 10-node Compare on Apple M3 Pro) reflect the choice. The
floor is dictated by Go map access cost (~15 ns each); we can't go
much below that without giving up the map.

## `vector.Compare` is a package-level function, not a method

Comparison is symmetric, i.e., neither operand is "more important." A
method like `a.Compare(b)` implies a primary subject. The function
form `Compare(a, b)` reads like math. This matches Go `stdlib`
conventions (`reflect.DeepEqual` is also a function for the same
reason).

## Sentinel errors

`ErrMaxDriftExceeded`, `ErrCorruptEncoding`, and friends are package-level sentinel values, not strings. Callers can use `errors.Is` to detect them, and the static type system makes typos visible at compile time. The accompanying `fmt.Errorf("%w: ...", ErrMaxDriftExceeded, ...)` pattern wraps the sentinel with operational context (which timestamp was rejected, what the local clock was) without breaking the detection.

## Pointer receivers on all clock methods

Every `Clock` method has a pointer receiver, even read-only ones like
`Now`. We do this for uniformity — having some methods take pointers
and others take values would invite the "trying to mutate a copy"
bug. The cost is one pointer dereference per call, which is well
within budget.

`Snapshot`, `Version`, `Timestamp`, and `Interval` use value receivers
because they're small immutable structs. Copying them is cheap and
correct. The one exception: `UnmarshalBinary` and `UnmarshalJSON`
methods on these types take pointer receivers because they have to
mutate the receiver. (We learned this the hard way during Phase 2 —
a value-receiver `UnmarshalBinary` silently does nothing useful.)

## Why `proofs/` is a separate package

Three reasons:

1. **Semantic distinction.** Tests in `lamport/`, `vector/`, etc.
   verify that the clocks satisfy their contracts. Tests in `proofs/`
   verify that the contracts solve the *literature scenarios*.
   Different audiences, different intent.

2. **Cross-package dependencies.** A proof for the NTP step scenario
   uses both `hlc` and `wallclock`. Putting it in either package would
   be arbitrary; a dedicated package makes the cross-cutting nature
   explicit.

3. **Marketing.** `proofs/` is the differentiator. Having a top-level
   directory whose entire purpose is "here are runnable reproductions
   of the failure scenarios the literature warns about" is what makes
   the README's claim ("chrono handles X") legible. The file structure
   carries the message.

## Why each example has its own `main.go`

`examples/` exists for users who learn by running code, not by reading
it. Each subdirectory is `package main` so `go run .` works without
ceremony. The output of each example is the deliverable — a narrative
log that makes the conceptual point at the terminal.

We resist the urge to make `examples/` a library of reusable fixtures.
Demo code is *supposed* to be self-contained and slightly redundant;
the moment it shares helpers, it stops being a demo and starts being
a framework.

## Benchmark sinks

Every benchmark assigns its inner-loop result to a variable. This
matters because Go's compiler will dead-code-eliminate calls whose
results are unused if it can prove the call has no observable side
effects. We learned this when an early version of
`BenchmarkHLCMarshalBinary` reported 0.26 ns/op (the entire call had
been eliminated). The fix is the `var sink T` pattern visible in every
benchmark in `benchmarks/bench_test.go`.

For methods that mutate state (`vector.Clock.Tick`, `hlc.Clock.Now`),
the mutation itself is an observable side effect that prevents
elimination — but we still assign to a sink for consistency and to
guard against future refactors that could make those calls
side-effect-free.

## What's NOT in chrono

A list of things people might expect to find but won't:

- **No clock for Raft / Paxos.** Those are consensus protocols, not
  clock primitives. They're consumers of clocks (Raft uses Lamport's
  ideas for log indices, sort of), not new clocks.
- **No CRDT primitives.** Adjacent topic, separate library. Dotted
  version vectors *enable* CRDTs but don't implement them.
- **No distributed clock-synchronization protocol.** chrono doesn't
  implement Berkeley algorithm, Cristian's algorithm, or PTP. Use NTP
  externally and pass the synced time in via the `wallclock`
  interface.
- **No GPS or atomic-clock support.** TrueTime's `UncertaintySource`
  interface is the seam where someone could plug in real GPS, but we
  don't ship a GPS implementation. NTP is the practical alternative.

## API stability promise (pre-1.0)

While the project is below v1.0.0, breaking API changes are allowed
between minor versions and will be noted in `CHANGELOG.md` with an
`[API]` prefix on the relevant entry. Migration guidance will be
included for any deliberately-incompatible change.

After v1.0.0, semantic versioning applies in the strict sense - no
breaking changes within a major version.
