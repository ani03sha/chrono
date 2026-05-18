package vector

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func snap(vec map[string]uint64) Snapshot {
	return Snapshot{vec: vec}
}

func TestCompareTable(t *testing.T) {
	// The six canonical cases from PROJECT_OVERVIEW.md. If any of these
	// regresses, the vector clock comparison is broken in a way that
	// would cause silent data loss in downstream systems.
	cases := []struct {
		name string
		a, b map[string]uint64
		want Relation
	}{
		{"single node, A < B", map[string]uint64{"A": 1}, map[string]uint64{"A": 2}, Before},
		{"single node, A > B", map[string]uint64{"A": 2}, map[string]uint64{"A": 1}, After},
		{"identical two-node vectors", map[string]uint64{"A": 1, "B": 1}, map[string]uint64{"A": 1, "B": 1}, Equal},
		{"divergent two-node vectors", map[string]uint64{"A": 2, "B": 1}, map[string]uint64{"A": 1, "B": 2}, Concurrent},
		{"disjoint single nodes", map[string]uint64{"A": 1}, map[string]uint64{"B": 1}, Concurrent},
		{"empty vs non-empty", map[string]uint64{}, map[string]uint64{"A": 1}, Before},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, Compare(snap(tc.a), snap(tc.b)))
		})
	}
}

func TestCompareEmptyVsEmpty(t *testing.T) {
	// Edge case the table doesn't cover: two empty vectors are Equal,
	// not Concurrent (no node has a strict inequality).
	require.Equal(t, Equal, Compare(snap(map[string]uint64{}), snap(map[string]uint64{})))
}

func TestCompareIgnoresZeroEntries(t *testing.T) {
	// An explicit zero entry must compare the same as a missing entry —
	// both represent "no events seen from this node."
	a := snap(map[string]uint64{"A": 1, "B": 0})
	b := snap(map[string]uint64{"A": 1})
	require.Equal(t, Equal, Compare(a, b))
}

func TestRelationString(t *testing.T) {
	require.Equal(t, "Before", Before.String())
	require.Equal(t, "After", After.String())
	require.Equal(t, "Equal", Equal.String())
	require.Equal(t, "Concurrent", Concurrent.String())
}
