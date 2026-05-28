# Which clock should you use?

A decision guide for the impatient, followed by deeper discussion of
each primitive.

## 30-second decision guide

![30 Second Decision Rule](/docs/chrono_clocks.png)

## Lamport

**Use when:** you need a total ordering of events that respects causality,
the events come from a known small set of processes, and the timestamps
themselves don't need to mean anything to humans.

**Don't use when:** you need to *know* whether two events are concurrent
(Lamport can't tell), or when timestamps will appear in logs or UIs.

**Real-world examples:**

- Distributed mutex protocols (Ricart–Agrawala): each process requests
  the lock with a Lamport timestamp; the smallest timestamp wins ties.
- Causally-ordered message delivery in a chat system: tag each message
  with the sender's Lamport timestamp; receivers buffer until they can
  deliver in order.
- Building blocks for higher-level abstractions — most distributed
  systems papers reach for Lamport when modeling causal order.

**Cost:** a single `sync.Mutex`-protected `uint64` per process.
Operations are single-digit nanoseconds; zero allocations.

## Vector

**Use when:** you need to *detect* whether two events are concurrent or
causally related. The classic case is a replicated KV store deciding
whether an incoming write is newer than what's already stored.

**Don't use when:** the set of actors changes faster than you can prune
the vector (mobile clients reconnecting with fresh IDs would blow this
up — use Dotted instead). Also don't use when the vector size becomes
proportional to the system's lifetime; you'll spend more bytes on
metadata than on data.

**Real-world examples:**

- Amazon Dynamo (2007 paper) — each value carries a vector clock; the
  storage layer surfaces conflicts as concurrent versions.
- Voldemort, Riak (pre-2.0) — same pattern.
- Causal consistency layers on top of eventually-consistent stores.

**Cost:** `O(N)` bytes per timestamp where `N` is the number of distinct
actors that have ever written. Operations are sub-microsecond up to
~10 nodes; degrades linearly past that.

## Dotted version vectors

**Use when:** you have a leaderless replicated store where clients
(actors) come and go dynamically and you need conflict detection
without unbounded vector growth.

**Don't use when:** you're in a closed system with a fixed actor set —
a plain vector clock is simpler.

**Real-world examples:**

- Riak 2.0+ uses DVVs to track which actor produced each dot, enabling
  pruning and avoiding the "false sibling" bug that plagued classic
  vector clocks.
- Antidote, Soundcloud's roshi, several CRDT libraries.

**Cost:** structurally identical to vector clocks (`map[string]uint64`).
The win is *semantic* — actors are decoupled from physical nodes and
can be retired.

## Hybrid logical clock

**Use when:** you want timestamps that are causally correct (like
Lamport) *and* close enough to wall time to be human-readable. Most
databases want this — commit timestamps that you can use both for
internal ordering and for "delete rows older than 30 days."

**Don't use when:** you need external consistency (events appearing
ordered to *external* observers in real time). HLC bounds drift from
wall time but doesn't guarantee external consistency on its own — use
TrueTime for that.

**Real-world examples:**

- CockroachDB: every transaction commit gets an HLC timestamp; MVCC
  reads use HLC for snapshot isolation across nodes.
- YugabyteDB: same pattern, slightly different parameters.
- MongoDB: cross-shard transactions since 4.0.
- etcd's MVCC layer uses an HLC variant for the same reason.

**Cost:** 12 bytes per timestamp on the wire. Per-call cost is
dominated by the wall-clock read (~30 ns on modern hardware) plus a
mutex and the four-case switch — about 40 ns total in this library.

## TrueTime / bounded-uncertainty

**Use when:** you need external consistency — the property that "if T1
commits before T2 starts in real wall-clock time, T1's timestamp must
be less than T2's." This is what a globally-distributed database needs
for snapshot reads to be consistent with linearizable writes across
regions.

**Don't use when:** you don't have a tight uncertainty bound. Spanner
gets ~7ms intervals from GPS + atomic clocks; with NTP you'll get tens
to hundreds of ms, which makes commit-wait expensive enough that you
should question whether you actually need external consistency.

**Real-world examples:**

- Google Spanner is the canonical user.
- CockroachDB has experimented with similar ideas but settled on HLC +
  uncertainty windows in its serializable isolation level.

**Cost:** the commit-wait latency tax. Roughly equal to the uncertainty
interval width. Spanner pays ~7ms per commit; with NTP you'd be paying
50–200ms, which is rarely worth it.

## Anti-patterns

- **Raw wall time for distributed event ordering.** NTP corrections
  will step you backward; events will appear out of causal order. Use
  HLC.
- **Last-write-wins in a leaderless store** without surfacing siblings.
  One concurrent write is silently lost — and you'll never see it in
  any log. Use Dotted with explicit sibling resolution.
- **Picking commit timestamps from `clock.Now().Earliest`.** This is
  the "optimistic" end of the uncertainty interval; another node may
  already have committed at that time. Pick from `.Latest` then
  `CommitWait`.
- **Comparing Lamport timestamps from unrelated processes for
  ordering.** The order is meaningful only along a causal chain; for
  unrelated events the order is meaningless. Use vector clocks if you
  need to know whether the chain exists.
- **Building HLC on top of a wall clock that isn't NTP-synced.** The
  `WallTime` field will diverge from reality arbitrarily, defeating
  the "approximate physical time" property that makes HLC useful.

## Cross-cutting concerns

**Persistence.** All these clocks have state. If your process restarts,
you must restore the clock to a value at least as great as the one it
last wrote. `lamport.NewAt`, `vector.NewFromMap`, and persistence of
HLC `Timestamp` to disk cover this. Don't reset to zero on restart —
that violates monotonicity.

**Serialization.** Every clock that gets sent over the wire needs a
defined binary format. `vector.Snapshot` and `hlc.Timestamp` provide
`MarshalBinary`/`UnmarshalBinary`. Wrap them in your own RPC types;
don't depend on Go's `gob` to do the right thing across versions.

**Clock skew tolerance.** HLC bounds drift via `maxDrift`. Configure it
to your environment: ~250ms for inter-region traffic, 50ms within a
data center. Without it, a single bad sender can permanently displace
every receiver's clock.
