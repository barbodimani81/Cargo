# Cargo: A Mutex-Based Batcher With Size + Timeout Flush

`cargo` is a small Go component that collects items into an in-memory FIFO batch and flushes them when:

* the batch reaches **BatchSize**, or
* **FlushInterval** has passed since the **first item** was added

It is designed for workloads where many goroutines produce data (e.g. JSON events) and a single flushing function persists them (e.g. into Postgres).

The implementation is fully **synchronous**, **mutex-based**, and **safe for concurrent producers**.

---

## Features

* **FIFO batching**
* **Size-based flush**
* **Timeout-based flush**
* **Safe concurrent `Add` calls**
* **Flush function runs outside mutex** (no blocking during DB writes)
* **Graceful shutdown with `Close`**
* Logging hooks to observe internal state transitions

---

## How It Works

### 1. Internal State (protected by a mutex)

* `batch []JSONPayload`
* `timer *time.Timer`
* `timerActive bool`
* `closed bool`

All reads/writes happen under `mu.Lock()` to prevent races.

### 2. Flush Triggers

**Size-based trigger**

Happens inside `Add` when:

```
len(batch) >= BatchSize
```

**Timeout trigger**

A timer is started when the **first item** arrives into an empty batch.
If no size flush occurs within `FlushInterval`, the timer fires and flushes the batch.

### 3. Flush Function

Every flush (size or timeout) happens:

1. Mutex locked:

    * snapshot batch
    * clear internal batch
    * stop timer
2. Mutex unlocked
3. `flushFn` is called with the snapshot

This ensures DB latency does **not** block producers.

### 4. Close()

`Close(ctx)`:

* Marks cargo as closed
* Stops any active timer
* Flushes any remaining items
* Rejects future `Add` calls
