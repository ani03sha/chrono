# HLC DB

Two database nodes with wall clocks that differ by 30ms exchange HLC timestamps; each receive preserves causality despite the skew.

## Run

```shell
go run .
```

## What to look for

The second round is the headline: Node 1's wall clock is BEHIND Node 2's, yet after Node 1 receives Node 2's message, its HLC has jumped forward to match. The wall component adopts Node 2's value; the logical component bumps to keep the `receive` strictly after the `send`.

This is the property production databases need to order replication events globally without GPS clocks.
