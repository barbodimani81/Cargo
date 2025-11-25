package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"test/cargo"
)

func main() {
	// log with timestamps + microseconds
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	ctx := context.Background()

	cfg := cargo.Config{
		Name:          "events",
		BatchSize:     10,
		FlushInterval: 2 * time.Second,
	}

	flushFn := func(ctx context.Context, batch []cargo.JSONPayload) error {
		log.Printf("[flush] called, batchSize=%d", len(batch))
		for i, b := range batch {
			fmt.Printf("id: %d; val: %s\n", i, string(b))
		}
		fmt.Println("----")
		return nil
	}

	c := cargo.NewCargo(cfg, flushFn)

	var wg sync.WaitGroup
	workers := 2
	perWorker := 3

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				payload, _ := json.Marshal(map[string]any{
					"worker": id,
					"n":      i,
				})
				_ = c.Add(ctx, payload)
				time.Sleep(500 * time.Millisecond)
			}
		}(w)
	}

	wg.Wait()

	time.Sleep(3 * time.Second)

	_ = c.Close(ctx)
}
