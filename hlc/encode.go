package hlc

import (
	"encoding/binary"
	"encoding/json"
	"errors"
)

// This is returned by UnmarshalBinary when the input bytes do not form a valid HLC encoding (wrong length).
var ErrCorruptEncoding = errors.New("hlc: corrupt binary encoding")

// This is the JSON wire shape. Kept private so we control the format independent of the Timestamp struct.
type jsonTimestamp struct {
	WallTime int64  `json:"wall_time"`
	Logical  uint32 `json:"logical"`
}

// MarshalJSON encodes the Timestamp as {"wall_time": 1234567890, "logical": 5}
func (a Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonTimestamp{WallTime: a.WallTime, Logical: a.Logical})
}

// UnmarshalJSON decodes a Timestamp from the JSON format produced by MarshalJSON.
func (a *Timestamp) UnmarshalJSON(data []byte) error {
	var j jsonTimestamp
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	a.WallTime = j.WallTime
	a.Logical = j.Logical
	return nil
}

// MarshalBinary encodes the Timestamp as 12 bytes:
//
//	+--------------------+--------------+
//	| WallTime  int64 BE | Logical u32  |
//	|  8 bytes           |  4 bytes     |
//	+--------------------+--------------+
//
// The fixed size makes this format friendly to row-oriented storage: every HLC timestamp on disk is the same width.
func (a Timestamp) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint64(buf[0:8], uint64(a.WallTime))
	binary.BigEndian.PutUint32(buf[8:12], a.Logical)
	return buf, nil
}

// UnmarshalBinary decodes a Timestamp from the format produced by MarshalBinary.
func (a *Timestamp) UnmarshalBinary(data []byte) error {
	if len(data) != 12 {
		return ErrCorruptEncoding
	}
	a.WallTime = int64(binary.BigEndian.Uint64(data[0:8]))
	a.Logical = binary.BigEndian.Uint32(data[8:12])
	return nil
}
