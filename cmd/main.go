package main

import (
	"context"
	"flag"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"final/cargo"
	"final/generator"
)

func main() {
	start := time.Now()

	objectsCount := flag.Int("count", 100, "number of objects to generate")
	batchSize := flag.Int("batch-size", 10, "batch size")
	timeout := flag.Duration("timeout", 2*time.Second, "flush timeout")
	workers := flag.Int("workers", 4, "number of worker goroutines")
	flag.Parse()

	var generated int64
	var flushed int64

	// 1. generator
	log.Printf("generating %d objects\n", *objectsCount)
	ch, err := generator.ItemGenerator(*objectsCount)
	if err != nil {
		log.Fatalf("generator error: %v", err)
	}

	// 2. cargo
	c, err := cargo.NewCargo(*batchSize, *timeout, func(ctx context.Context, batch []any) error {
		atomic.AddInt64(&flushed, int64(len(batch)))
		return nil
	})
	if err != nil {
		log.Fatalf("cargo error: %v", err)
	}

	// 3. worker pool
	var wg sync.WaitGroup
	wg.Add(*workers)

	for i := 0; i < *workers; i++ {
		go func() {
			defer wg.Done()
			for msg := range ch {
				atomic.AddInt64(&generated, 1)
				if err := c.Add(msg); err != nil {
					log.Printf("cargo add error: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// 4. final flush
	_ = c.Flush()

	// 5. print totals
	log.Printf("Generated items: %d", atomic.LoadInt64(&generated))
	log.Printf("Flushed items:   %d", atomic.LoadInt64(&flushed))

	// duration
	elapsed := time.Since(start)
	log.Printf("Duration: %v", elapsed)
}
