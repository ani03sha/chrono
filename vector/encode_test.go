package vector

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoundTripJSON(t *testing.T) {
	in := Snapshot{nodeID: "n1", vec: map[string]uint64{"n1": 5, "n2": 3, "n3": 0}}
	data, err := json.Marshal(in)
	require.NoError(t, err)

	var out Snapshot
	require.NoError(t, json.Unmarshal(data, &out))
	require.Equal(t, in.nodeID, out.NodeID())
	require.True(t, in.Equal(out))
}

func TestRoundTripBinary(t *testing.T) {
	in := Snapshot{vec: map[string]uint64{"alpha": 1, "beta": 1_000_000, "gamma": 42}}
	data, err := in.MarshalBinary()
	require.NoError(t, err)

	var out Snapshot
	require.NoError(t, (&out).UnmarshalBinary(data))
	require.True(t, in.Equal(out))
}

func TestBinaryFormatStable(t *testing.T) {
	// Marshalling the same Snapshot twice must produce byte-identical
	// output. We get this from sorting keys before writing — without it,
	// Go map iteration order would make the encoding non-deterministic.
	in := Snapshot{vec: map[string]uint64{"a": 1, "b": 2, "c": 3}}
	a, err := in.MarshalBinary()
	require.NoError(t, err)
	b, err := in.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, a, b)
}

func TestUnmarshalRejectsTruncated(t *testing.T) {
	in := Snapshot{vec: map[string]uint64{"x": 1}}
	data, err := in.MarshalBinary()
	require.NoError(t, err)

	for cut := 0; cut < len(data); cut++ {
		var out Snapshot
		err := (&out).UnmarshalBinary(data[:cut])
		require.ErrorIs(t, err, ErrCorruptEncoding,
			"truncating to %d bytes must error", cut)
	}
}

func TestUnmarshalRejectsTrailingGarbage(t *testing.T) {
	in := Snapshot{vec: map[string]uint64{"x": 1}}
	data, err := in.MarshalBinary()
	require.NoError(t, err)

	var out Snapshot
	err = (&out).UnmarshalBinary(append(data, 0xDE, 0xAD))
	require.ErrorIs(t, err, ErrCorruptEncoding)
}

func TestUnmarshalEmpty(t *testing.T) {
	// A Snapshot with no entries must round-trip too — the count field
	// is just zero, no per-entry bytes follow.
	in := Snapshot{vec: map[string]uint64{}}
	data, err := in.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, []byte{0x00, 0x00}, data)

	var out Snapshot
	require.NoError(t, (&out).UnmarshalBinary(data))
	require.Empty(t, out.Nodes())
}
