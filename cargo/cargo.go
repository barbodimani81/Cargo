package cargo

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type handlerFunc func(ctx context.Context, batch []any) error

type Cargo struct {
	mu        sync.Mutex
	batch     []any
	batchSize int
	timeout   time.Duration
	handler   handlerFunc
	ticker    *time.Ticker
	done      chan struct{}
}

func NewCargo(size int, timeout time.Duration, fn handlerFunc) (*Cargo, error) {
	if size <= 0 {
		return nil, fmt.Errorf("batch size must be greater than zero")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be greater than zero")
	}
	if fn == nil {
		return nil, fmt.Errorf("handler func cannot be empty")
	}

	c := &Cargo{
		batch:     make([]any, 0, size),
		batchSize: size,
		timeout:   timeout,
		handler:   fn,
		ticker:    time.NewTicker(timeout),
		done:      make(chan struct{}),
	}

	log.Printf("cargo: initialized with batch size %d and timeout %v", size, timeout)
	go c.run()
	return c, nil
}

func (c *Cargo) run() {
	sizeFlush := len(c.batch) >= c.batchSize
	if sizeFlush {
		err := c.Flush()
		c.ticker.Reset(c.timeout)
		if err != nil {
			log.Printf("cargo: failed to flush batch: %v", err)
		}
	}
	for {
		select {
		case <-c.ticker.C:
			log.Println("cargo: ticker fired, flushing batch")
			_ = c.Flush()
			c.ticker.Reset(c.timeout)
		case <-c.done:
			log.Println("cargo: shutting down ticker")
			c.ticker.Stop()
			return
		}
	}
}

// Add adds one item, flushes on size or timeout.
func (c *Cargo) Add(item any) error {
	c.mu.Lock()
	c.batch = append(c.batch, item)
	//batchLen := len(c.batch)
	//shouldFlush := batchLen >= c.batchSize
	c.mu.Unlock()

	//if shouldFlush {
	//	log.Printf("cargo: batch size reached, flushing [%d items] and resetting timer", batchLen)
	//	c.ticker.Reset(c.timeout)
	//	tickerNow := <-c.ticker.C
	//	log.Printf("time has been resetted: %v", tickerNow)
	//	return c.Flush()
	//}
	return nil
}

// Flush flushes the current batch.
func (c *Cargo) Flush() error {
	c.mu.Lock()
	if len(c.batch) == 0 {
		c.mu.Unlock()
		return nil
	}

	b := c.batch
	c.batch = make([]any, 0, c.batchSize)
	c.mu.Unlock()

	log.Printf("cargo: flushing %d items", len(b))
	return c.handler(context.Background(), b)
}

// Close stops the ticker and flushes any remaining items.
func (c *Cargo) Close() error {
	log.Println("cargo: closing")
	close(c.done)
	return c.Flush()
}
