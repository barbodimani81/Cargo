package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"test/cargo"
	"time"
)

func main() {
	ctx := context.Background()
	cfg := cargo.Config{
		BatchSize:     10,
		FlushInterval: 2 * time.Second,
	}

	flushfn := func(ctx context.Context, batch []cargo.JSONPayLoad) error {
		for i, v := range batch {
			fmt.Printf("id: %d; val: %s", i, string(v))
		}
		time.Sleep(1 * time.Second)
		return nil
	}

	c := cargo.NewCargo(cfg, flushfn)

	var wg sync.WaitGroup
	worker := 2
	perWorker := 3

	for w := 0; w < worker; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				payload, _ := json.Marshal(map[string]any{
					"worker": w,
					"n":      i,
				})
				_ = c.Add(ctx, payload)
				time.Sleep(500 * time.Millisecond)
			}
		}(w)
	}
	wg.Wait()
	time.Sleep(3 * time.Second)

	_ = c.Flush(ctx)
	c.Close()
}
