package ordering

import "sort"

// Event is a stamp-tagged carrier for application data. The ID is used for deterministic tie-breaking among
// concurrent events; Data is opaque.
type Event[T Comparable] struct {
	ID    string
	Stamp T
	Data  any
}

// TopologicalSort returns events in a total order consistent with the partial order induced by Comparable.HappensBefore.
// Concurrent events break by ID (lexicographic). The result is deterministic for a given input.
//
// Uses Kahn's algorithm: repeatedly emit nodes whose predecessors are all already emitted, choosing the lexicographically
// smallest ID among the available ones at each step. Runs in O(n²) - fine for the scale these utilities target.
func TopologicalSort[T Comparable](events []Event[T]) []Event[T] {
	n := len(events)
	if n == 0 {
		return nil
	}

	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		return events[order[i]].ID < events[order[j]].ID
	})

	// Build the DAG: succ[i] = indices j where events[i] -> events[j].
	succ := make([][]int, n)
	inDeg := make([]int, n)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			if events[i].Stamp.HappensBefore(events[j].Stamp) {
				succ[i] = append(succ[i], j)
				inDeg[j]++
			}
		}
	}

	used := make([]bool, n)
	out := make([]Event[T], 0, n)

	for len(out) < n {
		// Find the next event to add: the one with no incoming edges and the smallest ID.
		chosen := -1
		for _, i := range order {
			if !used[i] && inDeg[i] == 0 {
				chosen = i
				break
			}
		}
		if chosen == -1 {
			// Should not happen for a valid partial order — would mean
			// HappensBefore is intransitive. Stop rather than loop.
			break
		}
		used[chosen] = true
		out = append(out, events[chosen])
		for _, s := range succ[chosen] {
			inDeg[s]--
		}
	}

	return out
}

// Returns events that are not HappensBefore any other event in the input - the "frontier" of the partial order.
// Useful for answering "what are the latest causally independent updates?"
//
// Order preserves the input's order (no ID sort applied).
func MaximalElements[T Comparable](events []Event[T]) []Event[T] {
	var out []Event[T]
	for i, e := range events {
		isMax := true
		for j, f := range events {
			if i == j {
				continue
			}
			if e.Stamp.HappensBefore(f.Stamp) {
				isMax = false
				break
			}
		}
		if isMax {
			out = append(out, e)
		}
	}
	return out
}

// Returns events that no other event in the input HappensBefore - the "roots" of the partial order.
//
// Order preserves the input's order (no ID sort applied).
func MinimalElements[T Comparable](events []Event[T]) []Event[T] {
	var out []Event[T]
	for i, e := range events {
		isMin := true
		for j, f := range events {
			if i == j {
				continue
			}
			if f.Stamp.HappensBefore(e.Stamp) {
				isMin = false
				break
			}
		}
		if isMin {
			out = append(out, e)
		}
	}
	return out
}
