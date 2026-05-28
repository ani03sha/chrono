// Program vector_kv demonstrates vector-clock conflict detection in a tiny in-memory key-value store.
// Two clients write concurrently to the same key; the store keeps both writes as siblings rather than
// silently discarding one.
package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ani03sha/chrono/vector"
)

type versionedValue struct {
	value   string
	version vector.Snapshot
}

type store struct {
	rows map[string][]versionedValue
}

func newStore() *store {
	return &store{rows: make(map[string][]versionedValue)}
}

// write applies a new versioned write. If the new version is causally dominated by any existing row, it is stale and dropped.
// Existing rows dominated by the new version are dropped. Anything concurrent with the new version is kept as a sibling.
func (s *store) write(key, value string, version vector.Snapshot) {
	existing := s.rows[key]

	for _, row := range existing {
		rel := vector.Compare(version, row.version)
		if rel == vector.Before || rel == vector.Equal {
			return // incoming write is stale; ignore.
		}
	}

	var keepers []versionedValue
	for _, row := range existing {
		if vector.Compare(row.version, version) == vector.Before {
			continue // row is causally older; drop.
		}
		keepers = append(keepers, row)
	}
	keepers = append(keepers, versionedValue{value: value, version: version})
	s.rows[key] = keepers
}

func (s *store) read(key string) []versionedValue {
	return s.rows[key]
}

func versionStr(snap vector.Snapshot) string {
	nodes := snap.Nodes()
	sort.Strings(nodes)
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		parts = append(parts, fmt.Sprintf("%s:%d", n, snap.Get(n)))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func main() {
	db := newStore()
	clientA := vector.New("clientA")
	clientB := vector.New("clientB")

	fmt.Println("Step 1: Client A writes the initial value.")
	initial := clientA.Send()
	db.write("user:42", "Alice", initial)
	fmt.Printf("  store: \"Alice\" version=%s\n\n", versionStr(initial))

	fmt.Println("Step 2: Client B catches up via replication.")
	clientB.Receive(initial)
	fmt.Printf("  clientB version=%s\n\n", versionStr(clientB.Now()))

	fmt.Println("Step 3: Both clients edit concurrently (neither sees the other).")
	aWrite := clientA.Send()
	bWrite := clientB.Send()
	db.write("user:42", "Alice (edited by A)", aWrite)
	db.write("user:42", "Alice (edited by B)", bWrite)
	fmt.Printf("  A wrote %q  version=%s\n", "Alice (edited by A)", versionStr(aWrite))
	fmt.Printf("  B wrote %q  version=%s\n\n", "Alice (edited by B)", versionStr(bWrite))

	fmt.Println("Step 4: Read.")
	rows := db.read("user:42")
	fmt.Printf("  store returned %d row(s):\n", len(rows))
	for i, r := range rows {
		fmt.Printf("    [%d] %q  version=%s\n", i, r.value, versionStr(r.version))
	}

	if len(rows) > 1 {
		fmt.Println()
		fmt.Println("CONFLICT DETECTED — application must resolve siblings.")
		fmt.Println("(With last-write-wins, one of these writes would have been silently discarded.)")
	}
}
