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
	stopped chan struct{}

	closeOnce sync.Once
}

func NewCargo(size int, timeout time.Duration, fn handlerFunc) (*Cargo, error) {
	if err := configValidation(size, timeout, fn); err != nil {
		return nil, err
	}

	c := &Cargo{
		batch:     make([]any, 0, size),
		batchSize: size,
		timeout:   timeout,
		handler:   fn,
		done:      make(chan struct{}),
		flushCh:   make(chan struct{}, 1),
		stopped:   make(chan struct{}),
	}

	log.Printf("cargo: initialized with batch size %d and timeout %v", size, timeout)
	go c.run()
	return c, nil
}

func (c *Cargo) run() {
	ticker := time.NewTicker(c.timeout)
	defer ticker.Stop()
	//defer close(c.stopped)
	for {
		select {
		// timeout flush
		case <-ticker.C:
			log.Println("cargo: ticker fired, flushing batch")
			if err := c.Flush(); err != nil {
				log.Printf("cargo: failed to timeout flush batch: %v", err)
			}
		// size-based flush
		case <-c.flushCh:
			if err := c.Flush(); err != nil {
				log.Printf("cargo: failed to size flush batch: %v", err)
			}
			ticker.Reset(c.timeout)
		// closed channel
		case <-c.done:
			if err := c.Flush(); err != nil {
				log.Printf("cargo: failed to close channel: %v", err)
			}
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

func (c *Cargo) Close() error {
	c.closeOnce.Do(func() {
		close(c.done)
		log.Println("cargo: closing")
	})
	<-c.stopped
	return nil
}

func configValidation(size int, timeout time.Duration, fn handlerFunc) error {
	if size <= 0 {
		return fmt.Errorf("batch size must be greater than zero")
	}
	if timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}
	if fn == nil {
		return fmt.Errorf("handler func cannot be empty")
	}
	return nil
}
