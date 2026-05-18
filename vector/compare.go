package vector

// Describes the causal relationship between two vectors.
type Relation int

const (
	// Before means the first Snapshot causally precedes the second.
	Before Relation = iota

	// After means the second Snapshot causally precedes the first.
	After

	// Equal means the two Snapshots represent the same set of events.
	Equal

	// Concurrent means neither Snapshot precedes the other. The events
	// they describe happened on divergent branches of history with no
	// causal link between them. This is the case vector clocks add over
	// Lamport clocks — and the case key-value stores must detect to
	// avoid silently dropping concurrent writes.
	Concurrent
)

// Returns a human readable name for the relation
func (r Relation) String() string {
	switch r {
	case Before:
		return "Before"
	case After:
		return "After"
	case Equal:
		return "Equal"
	case Concurrent:
		return "Concurrent"
	default:
		return "Unknown"
	}
}

// Reports the causal relationship between the two snapshots.
func Compare(a, b Snapshot) Relation {
	aLessOrEq := true
	bLessOrEq := true

	// Walk a's keys first. Any key in b but not in a is handled in the second loop.
	// Together they cover the union without an explicit set-union allocation.
	for k, av := range a.vec {
		bv := b.vec[k]
		if av > bv {
			aLessOrEq = false
		}
		if bv > av {
			bLessOrEq = false
		}
	}

	for k, bv := range b.vec {
		if _, seen := a.vec[k]; seen {
			continue // already compared in the first loop
		}
		// av = 0 by absence; bv is whatever in b
		if bv > 0 {
			bLessOrEq = false
		}
	}

	switch {
	case aLessOrEq && bLessOrEq:
		return Equal
	case aLessOrEq:
		return Before
	case bLessOrEq:
		return After
	default:
		return Concurrent
	}
}
