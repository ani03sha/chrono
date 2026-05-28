package main

import (
	"fmt"
	"sort"
	"sync"

	"github.com/ani03sha/chrono/lamport"
)

type message struct {
	from  string
	text  string
	stamp uint64
}

type event struct {
	process string
	kind    string
	detail  string
	stamp   uint64
}

func main() {
	aToB := make(chan message, 1)
	bToA := make(chan message, 2)

	var (
		mu     sync.Mutex
		events []event
	)
	record := func(e event) {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Process A.
	go func() {
		defer wg.Done()
		c := lamport.New()

		record(event{"A", "tick", "loading config", c.Tick()})
		record(event{"A", "tick", "init complete", c.Tick()})

		s := c.Send()
		aToB <- message{from: "A", text: "hello", stamp: s}
		record(event{"A", "send", `"hello" to B`, s})

		m := <-bToA
		ts := c.Receive(m.stamp)
		record(event{"A", "recv", fmt.Sprintf("%q from B", m.text), ts})

		record(event{"A", "tick", "shutting down", c.Tick()})
	}()

	// Process B.
	go func() {
		defer wg.Done()
		c := lamport.New()

		record(event{"B", "tick", "warming up", c.Tick()})

		m := <-aToB
		ts := c.Receive(m.stamp)
		record(event{"B", "recv", fmt.Sprintf("%q from A", m.text), ts})

		s := c.Send()
		bToA <- message{from: "B", text: "world", stamp: s}
		record(event{"B", "send", `"world" to A`, s})
	}()

	wg.Wait()

	// Sort by Lamport timestamp, with process name as a deterministic
	// tiebreak for concurrent (equal-timestamp) events.
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].stamp != events[j].stamp {
			return events[i].stamp < events[j].stamp
		}
		return events[i].process < events[j].process
	})

	fmt.Println("Events sorted by Lamport timestamp:")
	fmt.Println("-----------------------------------")
	for _, e := range events {
		fmt.Printf("  ts=%d  %s  %-4s  %s\n", e.stamp, e.process, e.kind, e.detail)
	}
	fmt.Println()
	fmt.Println("Notice:")
	fmt.Println("  • A's local events appear in A's order; same for B's.")
	fmt.Println("  • Every send appears before its receive.")
	fmt.Println("  • Events with equal timestamps are concurrent.")
}
