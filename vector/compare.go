package vector

// Describes the partial order relation between two vector clock Snapshots
type Relation int

const (
	// Indicates a happens-before b: every component of a is at
	// most the corresponding component of b, and at least one is
	// strictly less.
	Before Relation = iota

	// Indicates b happens-before a (the inverse of Before).
	After

	// Indicates the two vectors have identical components.
	Equal

	// Indicates neither vector dominates the other; the
	// events they describe occurred without causal connection.
	Concurrent
)

// Returns human-readable name for the relation.
func (r Relation) String() string {
	switch r {
	case Before:
		return "before"
	case After:
		return "after"
	case Equal:
		return "equal"
	case Concurrent:
		return "concurrent"
	default:
		return "unknown"
	}
}

// Compare returns the partial-order relation between two snapshots.
//
// The algorithm walks the union of node identifiers in the two vectors,
// treating missing entries as zero, and tracks two flags:
//
//   - aLeqB: every component of a is <= the matching component of b
//   - bLeqA: every component of b is <= the matching component of a
//
// These flags map to relations as follows:
//
//   - both true   -> Equal
//   - aLeqB only  -> Before
//   - bLeqA only  -> After
//   - neither     -> Concurrent
//
// nodeID fields are ignored. Only the counter vectors are compared.
func Compare(a, b Snapshot) Relation {
	aLeqB := true
	bLeqA := true

	// Walk a's keys, comparing each against b (zero if missing).
	for node, av := range a.vec {
		bv := b.vec[node]
		if av > bv {
			aLeqB = false
		}
		if bv > av {
			bLeqA = false
		}
	}

	// Walk b's keys that don't appear in a; a's component is the zero
	// value, so we only need to check whether bv exceeds zero.
	for node, bv := range b.vec {
		if _, present := a.vec[node]; present {
			continue
		}
		if bv > 0 {
			bLeqA = false
		}
	}

	switch {
	case aLeqB && bLeqA:
		return Equal
	case aLeqB:
		return Before
	case bLeqA:
		return After
	default:
		return Concurrent
	}
}
