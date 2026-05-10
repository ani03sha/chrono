package vector_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ani03sha/chrono/vector"
	"github.com/stretchr/testify/require"
)

func TestRoundTripBinary(t *testing.T) {
	cases := []struct {
		name   string
		nodeID string
		vec    map[string]uint64
	}{
		{"empty", "n1", map[string]uint64{}},
		{"single entry", "n1", map[string]uint64{"n1": 1}},
		{"multi entry", "n1", map[string]uint64{"n1": 1, "n2": 2, "n3": 3}},
		{"unicode node ids", "n_α", map[string]uint64{"n_α": 1, "n_β": 2}},
		{"large counter", "n1", map[string]uint64{"n1": 1<<63 + 7}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := vector.NewFromMap(tc.nodeID, tc.vec).Now()

			data, err := original.MarshalBinary()
			require.NoError(t, err)

			var decoded vector.Snapshot
			require.NoError(t, decoded.UnmarshalBinary(data))

			require.Equal(t, original.NodeID(), decoded.NodeID())
			for _, n := range original.Nodes() {
				require.Equal(t, original.Get(n), decoded.Get(n))
			}
		})
	}
}

func TestBinaryFormatStable(t *testing.T) {
	// Encoding the same snapshot twice must produce byte-identical
	// output. Without sorting keys before encoding, Go's randomized map
	// iteration would make this fail intermittently.
	s := vector.NewFromMap("n1", map[string]uint64{
		"node_a": 1, "node_b": 2, "node_c": 3, "node_d": 4, "node_e": 5,
	}).Now()

	first, err := s.MarshalBinary()
	require.NoError(t, err)

	for i := 0; i < 100; i++ {
		later, err := s.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, first, later,
			"encoding is not stable across calls (iteration %d)", i)
	}
}

func TestRoundTripJSON(t *testing.T) {
	original := vector.NewFromMap("n1", map[string]uint64{
		"n1": 1, "n2": 2,
	}).Now()

	data, err := json.Marshal(original)
	require.NoError(t, err)
	require.Contains(t, string(data), `"node_id":"n1"`)

	var decoded vector.Snapshot
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.Equal(t, original.NodeID(), decoded.NodeID())
	require.Equal(t, original.Get("n1"), decoded.Get("n1"))
	require.Equal(t, original.Get("n2"), decoded.Get("n2"))
}

func TestUnmarshalBinaryRejectsCorrupt(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"empty input", nil},
		{"truncated nodeID length prefix", []byte{0x00}},
		{"nodeID length exceeds remaining input",
			[]byte{0x00, 0x05, 'a', 'b'}}, // claims 5 bytes, only 2 follow
		{"truncated entry count",
			[]byte{0x00, 0x02, 'n', '1', 0x00}},
		{"truncated entry payload",
			[]byte{0x00, 0x02, 'n', '1', 0x00, 0x01, 0x00, 0x02, 'a'}}, // missing value
		{"trailing bytes after valid encoding", []byte{
			0x00, 0x02, 'n', '1', // nodeID = "n1"
			0x00, 0x00, // 0 entries
			0xFF, // unexpected trailing byte
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var s vector.Snapshot
			err := s.UnmarshalBinary(tc.data)

			require.Error(t, err)
			require.True(t, errors.Is(err, vector.ErrInvalidEncoding),
				"every decode failure must wrap ErrInvalidEncoding so callers can match with errors.Is")
		})
	}
}
