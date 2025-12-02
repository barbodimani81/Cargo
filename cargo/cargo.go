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

	timeout time.Duration
	handler handlerFunc

	done    chan struct{}
	flushCh chan struct{}
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
		done:      make(chan struct{}),
		flushCh:   make(chan struct{}),
	}

	log.Printf("cargo: initialized with batch size %d and timeout %v", size, timeout)
	go c.run()
	return c, nil
}

func (c *Cargo) run() {
	ticker := time.NewTicker(c.timeout)
	defer ticker.Stop()
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
}

// Add adds one item
func (c *Cargo) Add(item any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		return fmt.Errorf("cargo closed")
	default:
	}

	c.batch = append(c.batch, item)
	if len(c.batch) >= c.batchSize {
		select {
		case c.flushCh <- struct{}{}:
			log.Println("cargo: flushing batch")
		default:
		}
	}
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
	var once sync.Once
	once.Do(func() {
		close(c.done)
		log.Println("cargo: closing")
	})
	return nil
}
