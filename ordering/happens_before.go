package ordering

import (
	"github.com/ani03sha/chrono/hlc"
	"github.com/ani03sha/chrono/vector"
)

// Comparable is the polymorphic happens-before interface. Adapters in this package wrap chrono's clock stamps to satisfy it.
//
// Methods compare against another Comparable; implementations should return false (rather than panic) when called with a
// stamp of a different concrete type: it is meaningless to ask "did this Lamport stamp happen before this HLC stamp?"
type Comparable interface {
	HappensBefore(other Comparable) bool
	Concurrent(other Comparable) bool
}

// LamportStamp wraps a Lamport timestamp.
//
// HappensBefore is exact (strict less-than in the Lamport total order). Concurrent is sound-but-incomplete:
// identical timestamps imply concurrent (the +1 rule prevents causally related events from sharing a stamp),
// but events with different stamps may also be concurrent in reality: Lamport clocks cannot detect that case.
type LamportStamp uint64

func (s LamportStamp) HappensBefore(other Comparable) bool {
	o, ok := other.(LamportStamp)
	if !ok {
		return false
	}
	return uint64(s) < uint64(o)
}

func (s LamportStamp) Concurrent(other Comparable) bool {
	o, ok := other.(LamportStamp)
	if !ok {
		return false
	}
	return uint64(s) == uint64(o)
}

// This wraps an HLC Timestamp. Same caveats as LamportStamp: HappensBefore is exact in the HLC total order,
// but Concurrent is sound-but-incomplete (equal-stamp implies concurrent; the converse does not hold).
type HLCStamp hlc.Timestamp

func (h HLCStamp) HappensBefore(other Comparable) bool {
	o, ok := other.(HLCStamp)
	if !ok {
		return false
	}
	return hlc.Timestamp(h).Less(hlc.Timestamp(o))
}

func (h HLCStamp) Concurrent(other Comparable) bool {
	o, ok := other.(HLCStamp)
	if !ok {
		return false
	}
	return hlc.Timestamp(h).Equal(hlc.Timestamp(o))
}

// This wraps a vector clock Snapshot. This is the only adapter that exposes the *exact* happens-before relation:
// HappensBefore and Concurrent map directly to vector.Compare's Before and Concurrent outcomes.
type VectorStamp struct {
	snap vector.Snapshot
}

// This wraps a Snapshot. The Snapshot is stored by value (Snapshots are immutable, so this is cheap and safe).
func NewVectorStamp(snap vector.Snapshot) VectorStamp {
	return VectorStamp{snap: snap}
}

// Snapshot returns the wrapped Snapshot.
func (v VectorStamp) Snapshot() vector.Snapshot {
	return v.snap
}

func (v VectorStamp) HappensBefore(other Comparable) bool {
	o, ok := other.(VectorStamp)
	if !ok {
		return false
	}
	return vector.Compare(v.snap, o.snap) == vector.Before
}

func (v VectorStamp) Concurrent(other Comparable) bool {
	o, ok := other.(VectorStamp)
	if !ok {
		return false
	}
	return vector.Compare(v.snap, o.snap) == vector.Concurrent
}
