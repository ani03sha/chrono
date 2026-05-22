package hlc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoundTripJSON(t *testing.T) {
	in := Timestamp{WallTime: 1_700_000_000_000_000_000, Logical: 42}
	data, err := json.Marshal(in)
	require.NoError(t, err)
	require.Contains(t, string(data), `"wall_time":1700000000000000000`)
	require.Contains(t, string(data), `"logical":42`)

	var out Timestamp
	require.NoError(t, json.Unmarshal(data, &out))
	require.True(t, in.Equal(out))
}

func TestRoundTripBinary(t *testing.T) {
	in := Timestamp{WallTime: 1_700_000_000_000_000_000, Logical: 42}
	data, err := in.MarshalBinary()
	require.NoError(t, err)
	require.Len(t, data, 12, "HLC binary form is always exactly 12 bytes")

	var out Timestamp
	require.NoError(t, (&out).UnmarshalBinary(data))
	require.True(t, in.Equal(out))
}

func TestBinaryFormatStable(t *testing.T) {
	in := Timestamp{WallTime: 999, Logical: 3}
	a, _ := in.MarshalBinary()
	b, _ := in.MarshalBinary()
	require.Equal(t, a, b)
}

func TestUnmarshalRejectsWrongLength(t *testing.T) {
	var out Timestamp
	require.ErrorIs(t, (&out).UnmarshalBinary(nil), ErrCorruptEncoding)
	require.ErrorIs(t, (&out).UnmarshalBinary([]byte{0, 1, 2}), ErrCorruptEncoding)
	require.ErrorIs(t, (&out).UnmarshalBinary(make([]byte, 13)), ErrCorruptEncoding)
}

func TestZeroTimestampRoundTrip(t *testing.T) {
	// Zero value must encode/decode cleanly — Timestamps are often
	// embedded in larger structs that may be partially populated.
	in := Timestamp{}
	data, err := in.MarshalBinary()
	require.NoError(t, err)

	var out Timestamp
	require.NoError(t, (&out).UnmarshalBinary(data))
	require.True(t, in.Equal(out))
}
