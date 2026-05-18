package dotted

// Sibling pairs a stored value with the Version under which it was written. When a KV store detects concurrent writes,
// it returns []Sibling[T] to the application instead of choosing a winner; resolution is done by the application.
type Sibling[T any] struct {
	Version *Version
	Value   T
}

// // ResolveByCustom returns the Sibling chosen by picker.
//
// This is the canonical pattern for application-defined resolution: CRDT-style merges, set union, business-rule precedence,
// prompting the user, etc.
//
// The picker receives the full sibling slice so it can also produce a synthetic merged value if needed (though the return
// type is a single Sibling — wrap the merged value in a fresh Sibling to do that).
//
// Panics if siblings is empty.
func ResolveByCustom[T any](siblings []Sibling[T], picker func([]Sibling[T]) Sibling[T]) Sibling[T] {
	if len(siblings) == 0 {
		panic("dotted: ResolveByCustom called with empty siblings")
	}
	return picker(siblings)
}

// ResolveLatestWriteWins picks the sibling whose Version has the highest sum of dot counters, breaking ties by slice
// position (earlier wins).
//
// This is a TEACHING IMPLEMENTATION. Real last-write-wins resolution uses an external wall-clock timestamp attached to
// each write — but wall clocks in distributed systems are unreliable (see chrono's hlc and truetime packages for why),
// which is precisely the failure mode DVVs exist to avoid.
//
// Using LWW silently loses the value that didn't win; this function exists so that loss is visible at the call site
// rather than buried in storage-engine internals.
//
// Panics if siblings is empty.
func ResolveLatestWriteWins[T any](siblings []Sibling[T]) Sibling[T] {
	if len(siblings) == 0 {
		panic("dotted: ResolveLatestWriteWins called with empty siblings")
	}
	bestIdx := 0
	bestSum := dotSum(siblings[0].Version)
	for i := 0; i < len(siblings); i++ {
		s := dotSum(siblings[i].Version)
		if s > bestSum {
			bestSum = s
			bestIdx = i
		}
	}
	return siblings[bestIdx]
}

func dotSum(v *Version) uint64 {
	var sum uint64
	for _, c := range v.dots {
		sum += c
	}
	return sum
}
