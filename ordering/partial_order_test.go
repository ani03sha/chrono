package ordering

import (
	"testing"

	"github.com/ani03sha/chrono/vector"
	"github.com/stretchr/testify/require"
)

// scenarioVector builds the four-event partial order described in the
// teaching prose:
//
// a1 and b1 are concurrent; so are a2 and b1. b2 descends from a1, a2,
// and b1.
func scenarioVector(t *testing.T) []Event[VectorStamp] {
	t.Helper()
	ca := vector.New("a")
	cb := vector.New("b")

	a1 := ca.Tick()          // {a:1}
	a2Send := ca.Send()      // {a:2}
	b1 := cb.Tick()          // {b:1}
	b2 := cb.Receive(a2Send) // {a:2, b:2}

	return []Event[VectorStamp]{
		{ID: "a1", Stamp: NewVectorStamp(a1)},
		{ID: "a2", Stamp: NewVectorStamp(a2Send)},
		{ID: "b1", Stamp: NewVectorStamp(b1)},
		{ID: "b2", Stamp: NewVectorStamp(b2)},
	}
}

func TestTopologicalSortRespectsCausality(t *testing.T) {
	events := scenarioVector(t)
	sorted := TopologicalSort(events)

	// We don't pin the exact order — multiple valid topological orders
	// exist — but we pin the causal requirements that any valid order
	// must satisfy.
	idx := make(map[string]int, len(sorted))
	for i, e := range sorted {
		idx[e.ID] = i
	}
	require.Less(t, idx["a1"], idx["a2"], "a1 must precede a2")
	require.Less(t, idx["a1"], idx["b2"], "a1 must precede b2")
	require.Less(t, idx["a2"], idx["b2"], "a2 must precede b2")
	require.Less(t, idx["b1"], idx["b2"], "b1 must precede b2")
}

func TestTopologicalSortDeterministicForConcurrent(t *testing.T) {
	// Two runs on the same input must produce identical output. ID
	// tiebreaking is what makes this hold.
	events := scenarioVector(t)
	a := TopologicalSort(events)
	b := TopologicalSort(events)
	require.Equal(t, len(a), len(b))
	for i := range a {
		require.Equal(t, a[i].ID, b[i].ID)
	}
}

func TestTopologicalSortExactOrderForScenario(t *testing.T) {
	// Given our ID-alphabetical tiebreak, the unique deterministic
	// order for this scenario is a1, a2, b1, b2.
	events := scenarioVector(t)
	sorted := TopologicalSort(events)

	got := make([]string, len(sorted))
	for i, e := range sorted {
		got[i] = e.ID
	}
	require.Equal(t, []string{"a1", "a2", "b1", "b2"}, got)
}

func TestTopologicalSortEmpty(t *testing.T) {
	require.Nil(t, TopologicalSort([]Event[LamportStamp]{}))
}

func TestTopologicalSortSingleEvent(t *testing.T) {
	got := TopologicalSort([]Event[LamportStamp]{
		{ID: "only", Stamp: LamportStamp(1)},
	})
	require.Len(t, got, 1)
	require.Equal(t, "only", got[0].ID)
}

func TestTopologicalSortLamport(t *testing.T) {
	// Lamport: total order on timestamps; alphabetical tiebreak on
	// equal stamps.
	events := []Event[LamportStamp]{
		{ID: "c", Stamp: LamportStamp(3)},
		{ID: "a", Stamp: LamportStamp(1)},
		{ID: "b", Stamp: LamportStamp(3)}, // tied with c → b first by ID
		{ID: "d", Stamp: LamportStamp(5)},
	}
	sorted := TopologicalSort(events)
	require.Equal(t, []string{"a", "b", "c", "d"}, ids(sorted))
}

func TestMaximalElementsScenario(t *testing.T) {
	events := scenarioVector(t)
	max := MaximalElements(events)
	require.Equal(t, []string{"b2"}, ids(max))
}

func TestMinimalElementsScenario(t *testing.T) {
	events := scenarioVector(t)
	min := MinimalElements(events)
	// a1 and b1 are both minimal; their order in the result follows
	// input order.
	require.Equal(t, []string{"a1", "b1"}, ids(min))
}

func TestMaximalElementsAllConcurrent(t *testing.T) {
	// Three concurrent vector stamps from three different processes.
	// All three are both maximal and minimal.
	events := []Event[VectorStamp]{
		{ID: "x", Stamp: NewVectorStamp(vector.New("x").Tick())},
		{ID: "y", Stamp: NewVectorStamp(vector.New("y").Tick())},
		{ID: "z", Stamp: NewVectorStamp(vector.New("z").Tick())},
	}
	require.Len(t, MaximalElements(events), 3)
	require.Len(t, MinimalElements(events), 3)
}

func ids[T Comparable](events []Event[T]) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.ID
	}
	return out
}
