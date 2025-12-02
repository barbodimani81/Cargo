# Cargo + Go

---

## A cargo package which receives and sends arrays

### 1. Data is generated in *generator* package
### 2. Data is *batched* and sent based on timeout or batch size limit
### 3. Received data is *Inserted* into Database

---

#### Example:
```go
for {
    select {
    // timeout flush
    case <-ticker.C:
        log.Println("cargo: ticker fired, flushing batch")
        _ = c.Flush()
    // size-based flush
    case <-c.flushCh:
        _ = c.Flush()
        ticker.Reset(c.timeout)
    // closed channel
    case <-c.done:
        _ = c.Flush()
        return
    }
}
```

---

// chera newcargo []User accept nemikone
// 