package vector

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
)

// This is returned when a Snapshot cannot be decoded from its wire form.
var ErrInvalidEncoding = errors.New("vector: invalid encoding")

// Encodes the snapshot in a compact, self-contained binary form.
//
// Wire format (all multi-byte integers big-endian):
//
//   - 2 bytes: nodeID length (uint16)
//   - N bytes: nodeID (UTF-8)
//   - 2 bytes: number of vec entries (uint16)
//   - Per entry:
//   - 2 bytes: entry nodeID length (uint16)
//   - M bytes: entry nodeID (UTF-8)
//   - 8 bytes: counter value (uint64)
//
// Entries are sorted by nodeID lexicographically so that encoding the
// same snapshot twice produces byte-identical output. Without the sort,
// Go's randomized map iteration would make encodings non-deterministic.
//
// A 10-node vector with 4-byte node IDs encodes to 144 bytes.
func (s Snapshot) MarshalBinary() ([]byte, error) {
	if len(s.nodeID) > math.MaxUint16 {
		return nil, fmt.Errorf("%w: nodeID too long (%d bytes, max %d)", ErrInvalidEncoding, len(s.nodeID), math.MaxUint16)
	}

	if len(s.vec) > math.MaxUint16 {
		return nil, fmt.Errorf("%w: too many entries (%d, max %d)", ErrInvalidEncoding, len(s.vec), math.MaxUint16)
	}

	// Pre-compute the buffer size so we allocate once.
	size := 2 + len(s.nodeID) + 2
	for k := range s.vec {
		if len(k) > math.MaxUint16 {
			return nil, fmt.Errorf("%w: entry nodeID too long (%d bytes)", ErrInvalidEncoding, len(k))
		}
		size += 2 + len(k) + 8
	}

	buf := make([]byte, size)
	off := 0

	binary.BigEndian.PutUint16(buf[off:], uint16(len(s.nodeID)))
	off += 2
	off += copy(buf[off:], s.nodeID)

	binary.BigEndian.PutUint16(buf[off:], uint16(len(s.vec)))
	off += 2

	for _, k := range sortedKeys(s.vec) {
		binary.BigEndian.PutUint16(buf[off:], uint16(len(k)))
		off += 2
		off += copy(buf[off:], k)
		binary.BigEndian.PutUint64(buf[off:], s.vec[k])
		off += 8
	}

	return buf, nil
}

// Decodes a snapshot from MarshalBinary's wire format.
func (s *Snapshot) UnmarshalBinary(data []byte) error {
	r := &reader{buf: data}

	nodeIDLen, err := r.readUint16()
	if err != nil {
		return err
	}
	nodeID, err := r.readBytes(int(nodeIDLen))
	if err != nil {
		return err
	}

	n, err := r.readUint16()
	if err != nil {
		return err
	}
	vec := make(map[string]uint64, n)
	for i := 0; i < int(n); i++ {
		klen, err := r.readUint16()
		if err != nil {
			return err
		}
		k, err := r.readBytes(int(klen))
		if err != nil {
			return err
		}
		v, err := r.readUint64()
		if err != nil {
			return err
		}
		vec[string(k)] = v
	}
	if r.remaining() != 0 {
		return fmt.Errorf("%w: %d trailing bytes", ErrInvalidEncoding, r.remaining())
	}

	s.nodeID = string(nodeID)
	s.vec = vec
	return nil
}

// Walks a byte slice with bounds-checked reads. Centralizing these checks in one place eliminates a class of
// off-by-one bugs that otherwise plagues hand-rolled binary decoders.
type reader struct {
	buf []byte
	off int
}

func (r *reader) readUint16() (uint16, error) {
	if r.off+2 > len(r.buf) {
		return 0, fmt.Errorf("%w: short read for uint16 at offset %d", ErrInvalidEncoding, r.off)
	}
	v := binary.BigEndian.Uint16(r.buf[r.off:])
	r.off += 2
	return v, nil
}

func (r *reader) readUint64() (uint64, error) {
	if r.off+8 > len(r.buf) {
		return 0, fmt.Errorf("%w: short read for uint64 at offset %d", ErrInvalidEncoding, r.off)
	}
	v := binary.BigEndian.Uint64(r.buf[r.off:])
	r.off += 8
	return v, nil
}

func (r *reader) readBytes(n int) ([]byte, error) {
	if n < 0 || r.off+n > len(r.buf) {
		return nil, fmt.Errorf("%w: short read for %d bytes at offset %d", ErrInvalidEncoding, n, r.off)
	}
	b := r.buf[r.off : r.off+n]
	r.off += n
	return b, nil
}

func (r *reader) remaining() int { return len(r.buf) - r.off }

// Returns the keys of m in lexicographic order. The sort is what makes binary encoding deterministic from a Go map.
func sortedKeys(m map[string]uint64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// This is the wire shape for JSON encoding. It is unexported so the JSON shape is not part of the package's API surface,
// only the JSON methods are.
type jsonForm struct {
	NodeID string            `json:"node_id"`
	Vec    map[string]uint64 `json:"vec"`
}

// Encodes the snapshot as JSON of the form {"node_id": "...", "vec": {...}}.
func (s Snapshot) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonForm{NodeID: s.nodeID, Vec: s.vec})
}

// Decodes a snapshot from JSON.
func (s *Snapshot) UnmarshalJSON(data []byte) error {
	var f jsonForm
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidEncoding, err)
	}
	if f.Vec == nil {
		f.Vec = make(map[string]uint64)
	}
	s.nodeID = f.NodeID
	s.vec = f.Vec
	return nil
}
