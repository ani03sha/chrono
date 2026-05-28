// Program truetime_external demonstrates external consistency. Two scenarios run side by side: in the first,
// T1 commits without commit-wait and an immediate T2 picks an EARLIER timestamp (a violation).
// In the second, T1 performs commit-wait before exposing its commit, and T2 picks a strictly LATER timestamp.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ani03sha/chrono/truetime"
)

func main() {
	base := time.Unix(1_700_000_000, 0)
	initial := truetime.Interval{
		Earliest: base,
		Latest:   base.Add(7 * time.Millisecond), // 7ms: Spanner-class uncertainty
	}

	fmt.Println("=== Scenario A: no commit-wait ===")
	{
		src := truetime.NewFakeSource(initial)
		clock := truetime.New(src)

		t1 := clock.Now().Latest
		fmt.Printf("T1 picks commit timestamp = %s\n", t1.Format("15:04:05.000000000"))

		t2 := clock.Now().Earliest
		fmt.Printf("T2 starts, picks Earliest = %s\n", t2.Format("15:04:05.000000000"))

		if t2.Before(t1) {
			fmt.Printf("→ VIOLATION: T2 is %v earlier than T1.\n",
				t1.Sub(t2))
			fmt.Println("  An observer who saw T1 commit would see T2 get an EARLIER timestamp.")
		}
	}
	fmt.Println()

	fmt.Println("=== Scenario B: with commit-wait ===")
	{
		src := truetime.NewFakeSource(initial)
		clock := truetime.New(src)

		t1 := clock.Now().Latest
		fmt.Printf("T1 picks commit timestamp = %s\n", t1.Format("15:04:05.000000000"))

		// Simulate the passage of time: a background goroutine advances
		// the source's interval after a short delay.
		go func() {
			time.Sleep(20 * time.Millisecond)
			src.Set(truetime.Interval{
				Earliest: base.Add(30 * time.Millisecond),
				Latest:   base.Add(37 * time.Millisecond),
			})
		}()

		fmt.Println("T1 calls CommitWait — blocking until the clock is provably past t1...")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		start := time.Now()
		if err := truetime.CommitWait(ctx, clock, t1); err != nil {
			fmt.Printf("FAIL: %v\n", err)
			return
		}
		fmt.Printf("CommitWait returned after %v of real time.\n", time.Since(start))

		t2 := clock.Now().Earliest
		fmt.Printf("T2 starts, picks Earliest = %s\n", t2.Format("15:04:05.000000000"))

		if t2.After(t1) {
			fmt.Printf("→ OK: T2 is %v AFTER T1. External consistency holds.\n",
				t2.Sub(t1))
		}
	}
}
