package dotted

// Version records the causal history of a stored value as a set of dots compressed into a map of
// actor -> highest contiguous counter.
//
// Version are immutable; all operations return a new Version. The zero value is not usable; obtain
// a version from NewVersion.
type Version struct {
	dots map[string]uint64
}

// Returns an empty version. Every other version descends from this one.
func NewVersion() *Version {
	return &Version{dots: make(map[string]uint64)}
}

// Returns a new Version with actor's counter advanced by 1. The receiver is not modified.
func (v *Version) Increment(actor string) *Version {
	out := &Version{dots: make(map[string]uint64, len(v.dots)+1)}
	for k, c := range v.dots {
		out.dots[k] = c
	}
	out.dots[actor] = out.dots[actor] + 1
	return out
}

// Returns a new Version whose dot for each actor is the pointwise maximum of v's and other's.
// Neither receiver is modified.
//
// Merge is what a node does when it learns of another node's version; during replication or read-repair.
// After Merge, the result descends from both inputs.
func (v *Version) Merge(other *Version) *Version {
	out := &Version{dots: make(map[string]uint64, len(v.dots)+len(other.dots))}
	for k, c := range v.dots {
		out.dots[k] = c
	}
	for k, c := range other.dots {
		if c > out.dots[k] {
			out.dots[k] = c
		}
	}
	return out
}

// Descends reports whether v's history includes other's: for every actor, v's counter is at least as high as other's.
// Missing entries are treated as zero.
//
// Replicated stores ask this question on every incoming write: if the new write descends from the existing value,
// the write is causally newer and replaces it; otherwise the two are equal or concurrent and further analysis is required.
func (v *Version) Descends(other *Version) bool {
	for k, c := range other.dots {
		if v.dots[k] < c {
			return false
		}
	}
	return true
}

// Dominates reports whether v strictly descends from other: Descends plus at least one actor whose counter is strictly
// greater (or an actor present in v but not other).
//
// This is the test for "v is unambiguously a newer version than other." If a.Dominates(b) is true, b can be safely discarded.
func (v *Version) Dominates(other *Version) bool {
	strictly := false
	for k, c := range other.dots {
		vc := v.dots[k]
		if vc < c {
			return false
		}
		if vc > c {
			strictly = true
		}
	}

	if !strictly {
		for k := range v.dots {
			if _, ok := other.dots[k]; !ok {
				return true
			}
		}
	}
	return strictly
}

// Equals reports whether two Versions have identical dot sets.
//
// Since Increment and Merge never store zero-valued counters, equal
// dot maps imply equal causal histories without needing to canonicalize.
func (v *Version) Equals(other *Version) bool {
	if len(v.dots) != len(other.dots) {
		return false
	}
	for k, c := range v.dots {
		if other.dots[k] != c {
			return false
		}
	}
	return true
}

// Concurrent reports whether neither version descends from the other: the writes happened on divergent branches of history.
// A KV store detecting this must surface both versions as siblings rather than pick one.
func (v *Version) Concurrent(other *Version) bool {
	return !v.Descends(other) && !other.Descends(v)
}

// Dots returns a copy of the version's actor→counter map. The returned map is owned by the caller; mutating it does not affect v.
func (v *Version) Dots() map[string]uint64 {
	out := make(map[string]uint64, len(v.dots))
	for k, c := range v.dots {
		out[k] = c
	}
	return out
}
