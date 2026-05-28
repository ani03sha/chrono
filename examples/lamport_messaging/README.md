# Lamport Messaging

Two **goroutines** with Lamport clocks exchange messages over channels.

## Run

```shell
go run .
```

## What to look for

The output is the same set of events twice over, but it isn't. The goroutines record events concurrently; the global log is sorted by Lamport timestamp at the end. Look for:

- Each process's local events appear in that process's order.
- Every send appears strictly before its corresponding receive.
- A and B's first ticks share `ts=1` — they're concurrent, and the Lamport clock honestly reports they have no causal relationship.
