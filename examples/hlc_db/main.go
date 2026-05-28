// Program hlc_db demonstrates that hybrid logical clocks preserve causality even when two database nodes' wall clocks
// disagree by tens of milliseconds: the property CockroachDB, YugabyteDB, and MongoDB rely on to order replication events
// without GPS-class time synchronization.
package main

import (
	"fmt"
	"time"

	"github.com/ani03sha/chrono/hlc"
	"github.com/ani03sha/chrono/internal/wallclock"
)

func main() {
	base := time.Unix(1_700_000_000, 0)
	fake1 := wallclock.NewFake(base)
	fake2 := wallclock.NewFake(base.Add(30 * time.Millisecond))

	node1 := hlc.New(fake1)
	node2 := hlc.New(fake2)

	fmt.Printf("Node 1 wall clock: %s\n", fake1.Now().Format(time.RFC3339Nano))
	fmt.Printf("Node 2 wall clock: %s  (30ms ahead of Node 1)\n\n",
		fake2.Now().Format(time.RFC3339Nano))

	fmt.Println("Round 1: Node 1 ships an update to Node 2.")
	msg1 := node1.Send()
	fmt.Printf("  Node 1 sends with HLC = %s\n", msg1)

	t2, _ := node2.Receive(msg1)
	fmt.Printf("  Node 2 receives; HLC  = %s\n", t2)

	if msg1.Less(t2) {
		fmt.Println("  → Node 2's HLC strictly exceeds the message's. Causality preserved.")
	}
	fmt.Println()

	fmt.Println("Round 2: Node 2 ships an update to Node 1 (whose wall is BEHIND).")
	msg2 := node2.Send()
	fmt.Printf("  Node 2 sends with HLC = %s\n", msg2)

	t1, _ := node1.Receive(msg2)
	fmt.Printf("  Node 1 receives; HLC  = %s\n", t1)

	if msg2.Less(t1) {
		fmt.Println("  → Node 1 absorbed Node 2's 'future' WallTime. Causality preserved despite skew.")
		fmt.Printf("    (Node 1's WallTime jumped from base to base+30ms; logical = %d.)\n", t1.Logical)
	}
}
