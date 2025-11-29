package cargo

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type handlerFunc func(ctx context.Context, batch []any) error

type Config struct {
	BatchSize   int
	Timeout     time.Duration
	HandlerFunc handlerFunc
}

type Cargo struct {
	mu    sync.Mutex
	batch []any
	cfg   Config
}

// --- validation helpers ---

func BatchSizeValidation(size int) error {
	if size <= 0 {
		return fmt.Errorf("batch size must be greater than zero")
	}
	return nil
}

func TimeoutValidation(timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}
	return nil
}

// --- constructor ---

func NewCargo(size int, timeout time.Duration, fn handlerFunc) (*Cargo, error) {
	if err := BatchSizeValidation(size); err != nil {
		return nil, err
	}
	if err := TimeoutValidation(timeout); err != nil {
		return nil, err
	}
	if fn == nil {
		return nil, fmt.Errorf("handler func cannot be empty")
	}

	return &Cargo{
		batch: make([]any, 0, size),
		cfg: Config{
			BatchSize:   size,
			Timeout:     timeout,
			HandlerFunc: fn,
		},
	}, nil
}

// Add adds one item, flushes on size or timeout.
func (c *Cargo) Add(item any) error {
	c.mu.Lock()
	wasEmpty := len(c.batch) == 0

	c.batch = append(c.batch, item)
	shouldFlush := len(c.batch) >= c.cfg.BatchSize
	c.mu.Unlock()

	// size-based flush
	if shouldFlush {
		return c.Flush()
	}

	// timeout-based: only schedule when batch transitions from empty -> non-empty
	if wasEmpty {
		timeout := c.cfg.Timeout
		go func() {
			time.Sleep(timeout)
			_ = c.Flush()
		}()
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
	// reset with capacity BatchSize
	c.batch = make([]any, 0, c.cfg.BatchSize)
	c.mu.Unlock()

	return c.cfg.HandlerFunc(context.Background(), b)
}

// Close is optional; just flushes any remaining items.
func (c *Cargo) Close() error {
	return c.Flush()
}
