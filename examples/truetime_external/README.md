# TrueTime External

Side-by-side: external consistency without and with commit-wait.

## Run

```shell
go run .
```

## What to look for

Scenario A picks T1's commit timestamp at `Now().Latest` and T2's at `Now().Earliest`: both from the same 7ms-wide interval. 
T2 ends up *EARLIER* than T1 despite starting later. That's exactly the violation commit-wait is designed to prevent.

Scenario B inserts `CommitWait` between T1's commit and T2's start. The `CommitWait` blocks until the clock is provably past T1's commit; when T2 starts after the wait, its timestamp is strictly later. External consistency: held.

The "real time" delay printed for `CommitWait` is what Spanner pays as its commit latency tax — proportional to the uncertainty interval width.
