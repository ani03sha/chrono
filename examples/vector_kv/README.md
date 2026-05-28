# Vector KV

A toy KV store that detects concurrent writes via vector clocks.

## Run

```shell
go run .
```

## What to look for

Two clients write to the same key from the same starting state. The store cannot pick a winner without losing data, so it keeps both as siblings. The read returns:

```plaintext
store returned 2 row(s):
    [0] "Alice (edited by A)"  version={clientA:2}
    [1] "Alice (edited by B)"  version={clientA:1, clientB:2}

    CONFLICT DETECTED — application must resolve siblings.
```

Neither write descends from the other; their vector clocks tell the store exactly that, and the store surfaces both rather than guess.