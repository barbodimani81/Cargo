// cargo package -> struct - new cargo - add - flush - log

package cargo

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type JSONPayLoad json.RawMessage

var ErrCargoClosed = fmt.Errorf("cargo is closed")

type Cargo struct {
	batch []JSONPayLoad
	cfg   Config

	flushFn FlushFunc
	mu      sync.Mutex
	closed  bool

	timer    *time.Timer
	isActive bool
}

type Config struct {
	BatchSize     int
	FlushInterval time.Duration
}

type FlushFunc func(ctx context.Context, batch []JSONPayLoad) error

func NewCargo(cfg Config, fn FlushFunc) *Cargo {
	if cfg.BatchSize <= 0 {
		panic("batch size must be greater than zero")
	}
	if fn == nil {
		panic("a flush function must be provided")
	}
	return &Cargo{
		cfg:     cfg,
		flushFn: fn,
		batch:   make([]JSONPayLoad, 0, cfg.BatchSize),
	}
}

func (c *Cargo) Add(ctx context.Context, item JSONPayLoad) error {
	var toFlush []JSONPayLoad
	c.mu.Lock()

	if c.closed {
		c.mu.Unlock()
		return ErrCargoClosed
	}

	c.batch = append(c.batch, item)

	if len(c.batch) == 1 && c.cfg.FlushInterval > 0 && !c.isActive {
		c.startTimerLocked()
	}

	if len(c.batch) >= c.cfg.BatchSize {
		toFlush = make([]JSONPayLoad, len(c.batch))
		copy(toFlush, c.batch)
		c.batch = c.batch[:0]

		c.stopTimerLocked()
	}
	c.mu.Unlock()

	if toFlush != nil {
		return c.flushFn(ctx, toFlush)
	}

	return nil
}

func (c *Cargo) Flush(ctx context.Context) error {
	var toFlush []JSONPayLoad
	c.mu.Lock()

	if len(c.batch) > 0 {
		toFlush = make([]JSONPayLoad, len(c.batch))
		copy(toFlush, c.batch)
		c.batch = c.batch[:0]
		c.stopTimerLocked()
	}
	c.mu.Unlock()

	if toFlush != nil {
		return c.flushFn(ctx, toFlush)
	}

	return nil
}

func (c *Cargo) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
}

func (c *Cargo) startTimerLocked() {
	if c.isActive || c.cfg.FlushInterval <= 0 {
		return
	}

	if c.timer != nil {
		c.timer.Stop()
	}
	c.isActive = false
	c.timer = nil
}

func (c *Cargo) stopTimerLocked() {
	if !c.isActive {
		return
	}

	if c.timer != nil {
		c.timer.Stop()
	}

	c.isActive = false
	c.timer = nil
}

// onTimeout is called in its own goroutine when the timer fires.
func (c *Cargo) onTimeout() {
	var toFlush []JSONPayLoad

	c.mu.Lock()

	// If timer is not active, someone already stopped it (size flush/manual flush).
	if !c.isActive {
		c.mu.Unlock()
		return
	}

	if len(c.batch) > 0 {
		// snapshot current batch
		toFlush = make([]JSONPayLoad, len(c.batch))
		copy(toFlush, c.batch)
		c.batch = c.batch[:0]
	}

	// timer has fired, so it's no longer active
	c.isActive = false
	c.timer = nil

	c.mu.Unlock()

	if len(toFlush) == 0 {
		return
	}

	// flush outside the lock
	_ = c.flushFn(context.Background(), toFlush)
}
