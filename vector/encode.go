package vector

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// This is returned by UnmarshalBinary when the input bytes do not form a valid vector encoding.
var ErrCorruptEncoding = errors.New("vector: corrupt binary encoding")

// This is the JSON wire shape. It is kept private so we control the format
// independent of the in-memory Snapshot type.
type jsonSnapshot struct {
	NodeID string            `json:"node_id"`
	Vec    map[string]uint64 `json:"vec"`
}

// Encodes the Snapshot as JSON of the form: {"node_id": "n1", "vec": {"n1": 5, "n2": 3}}
func (s Snapshot) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonSnapshot{NodeID: s.nodeID, Vec: s.vec})
}

// Decodes a Snapshot from the JSON format.
func (s *Snapshot) UnmarshalJSON(data []byte) error {
	var j jsonSnapshot
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	s.nodeID = j.NodeID
	if j.Vec == nil {
		s.vec = make(map[string]uint64)
	} else {
		s.vec = j.Vec
	}
	return nil
}

// This method encodes the snapshot in a compact binary form.
//
//	+----------+------------------------------------------------+
//	| count    | repeated (count) times:                        |
//	| uint16   |   id_len  uint16                               |
//	|          |   id      id_len bytes (UTF-8)                 |
//	|          |   value   uint64                               |
//	+----------+------------------------------------------------+
//
// All integers are big-endian. Node IDs are written in sorted order so the encoding is deterministic
// for a given Snapshot — useful for hashing and equality checks against the wire bytes.
//
// Note: the node ID of the originating Clock (Snapshot.NodeID) is NOT included in the binary form.
// The wire format describes a vector, which is what receivers need; the originating node ID is recoverable
// from the message metadata if needed.
func (s Snapshot) MarshalBinary() ([]byte, error) {
	if len(s.vec) > 0xFFFF {
		return nil, fmt.Errorf("vector: too many nodes (%d) for uint16 count", len(s.vec))
	}

	keys := make([]string, 0, len(s.vec))
	for k := range s.vec {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Pre-size the buffer to avoid growth: 2 bytes count + per entry (2 bytes id_len + len(id) + 8 bytes value)
	size := 2
	for _, k := range keys {
		if len(k) > 0xFFFF {
			return nil, fmt.Errorf("vector: node id %q too long for uint16 length", k)
		}
		size += 2 + len(k) + 8
	}
	buff := make([]byte, size)

	off := 0
	binary.BigEndian.PutUint16(buff[off:], uint16(len(keys)))
	off += 2
	for _, k := range keys {
		binary.BigEndian.PutUint16(buff[off:], uint16(len(k)))
		off += 2
		copy(buff[off:], k)
		off += len(k)
		binary.BigEndian.PutUint64(buff[off:], s.vec[k])
		off += 8
	}
	return buff, nil
}

// Decodes a Snapshot from the format produced by MarshalBinary.
// Why is the originating nodeID excluded from the binary wire format? Because the receiver doesn't need it.
// The vector itself encodes everything causal; the originating node ID is just metadata for debugging/logging.
// In real systems the sender's identity comes from the network envelope (TLS cert, RPC metadata, etc.), not the payload.
// We keep it in Snapshot for in-memory ergonomics but drop it on the wire to save bytes.
func (s *Snapshot) UnmarshalBinary(data []byte) error {
	if len(data) < 2 {
		return ErrCorruptEncoding
	}
	count := int(binary.BigEndian.Uint16(data[0:2]))
	off := 2
	vec := make(map[string]uint64, count)
	for i := 0; i < count; i++ {
		if off+2 > len(data) {
			return ErrCorruptEncoding
		}
		idLen := int(binary.BigEndian.Uint16(data[off : off+2]))
		off += 2
		if off+idLen+8 > len(data) {
			return ErrCorruptEncoding
		}
		id := string(data[off : off+idLen])
		off += idLen
		val := binary.BigEndian.Uint64(data[off : off+8])
		off += 8
		vec[id] = val
	}
	if off != len(data) {
		return ErrCorruptEncoding // trailing garbage
	}
	s.nodeID = ""
	s.vec = vec
	return nil
}
