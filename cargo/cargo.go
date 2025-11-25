package cargo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type JSONPayload = json.RawMessage

type FlushFunc func(ctx context.Context, batch []JSONPayload) error

type Config struct {
	Name          string
	BatchSize     int
	FlushInterval time.Duration
}

var ErrCargoClosed = fmt.Errorf("cargo is closed")

type Cargo struct {
	cfg     Config
	flushFn FlushFunc

	mu     sync.Mutex
	batch  []JSONPayload
	closed bool

	timer       *time.Timer
	timerActive bool
}

func NewCargo(cfg Config, flushFn FlushFunc) *Cargo {
	if cfg.BatchSize <= 0 {
		panic("BatchSize must be > 0")
	}
	if flushFn == nil {
		panic("flushFn is required")
	}

	c := &Cargo{
		cfg:     cfg,
		flushFn: flushFn,
		batch:   make([]JSONPayload, 0, cfg.BatchSize),
	}

	if c.cfg.Name == "" {
		c.cfg.Name = "default"
	}

	log.Printf("[cargo:%s] NewCargo: batchSize=%d, flushInterval=%s",
		c.cfg.Name, c.cfg.BatchSize, c.cfg.FlushInterval)

	return c
}

func (c *Cargo) Add(ctx context.Context, item JSONPayload) error {
	var toFlush []JSONPayload

	c.mu.Lock()

	if c.closed {
		c.mu.Unlock()
		log.Printf("[cargo:%s] Add: rejected, cargo is closed", c.cfg.Name)
		return ErrCargoClosed
	}

	c.batch = append(c.batch, item)
	log.Printf("[cargo:%s] Add: appended, batchLen=%d", c.cfg.Name, len(c.batch))

	if len(c.batch) == 1 && c.cfg.FlushInterval > 0 && !c.timerActive {
		log.Printf("[cargo:%s] Add: first item in batch, starting timer", c.cfg.Name)
		c.startTimerLocked()
	}

	if len(c.batch) >= c.cfg.BatchSize {
		log.Printf("[cargo:%s] Add: batchSize reached (%d), triggering size flush",
			c.cfg.Name, c.cfg.BatchSize)

		toFlush = make([]JSONPayload, len(c.batch))
		copy(toFlush, c.batch)
		c.batch = c.batch[:0]

		c.stopTimerLocked()
	}

	c.mu.Unlock()

	if toFlush != nil {
		log.Printf("[cargo:%s] Add: performing size-based flush, batchSize=%d",
			c.cfg.Name, len(toFlush))
		return c.flushFn(ctx, toFlush)
	}

	return nil
}

func (c *Cargo) Flush(ctx context.Context) error {
	var toFlush []JSONPayload

	log.Printf("[cargo:%s] Flush: called", c.cfg.Name)

	c.mu.Lock()
	if len(c.batch) > 0 {
		log.Printf("[cargo:%s] Flush: draining batchLen=%d", c.cfg.Name, len(c.batch))
		toFlush = make([]JSONPayload, len(c.batch))
		copy(toFlush, c.batch)
		c.batch = c.batch[:0]
		c.stopTimerLocked()
	} else {
		log.Printf("[cargo:%s] Flush: nothing to flush", c.cfg.Name)
	}
	c.mu.Unlock()

	if toFlush == nil {
		return nil
	}

	log.Printf("[cargo:%s] Flush: performing manual flush, batchSize=%d",
		c.cfg.Name, len(toFlush))
	return c.flushFn(ctx, toFlush)
}

func (c *Cargo) Close(ctx context.Context) error {
	var toFlush []JSONPayload

	log.Printf("[cargo:%s] Close: called", c.cfg.Name)

	c.mu.Lock()

	if c.closed {
		c.mu.Unlock()
		log.Printf("[cargo:%s] Close: already closed", c.cfg.Name)
		return nil
	}
	c.closed = true

	if len(c.batch) > 0 {
		log.Printf("[cargo:%s] Close: draining remaining batchLen=%d",
			c.cfg.Name, len(c.batch))
		toFlush = make([]JSONPayload, len(c.batch))
		copy(toFlush, c.batch)
		c.batch = c.batch[:0]
	} else {
		log.Printf("[cargo:%s] Close: no remaining items", c.cfg.Name)
	}

	c.stopTimerLocked()

	c.mu.Unlock()

	if toFlush == nil {
		return nil
	}

	log.Printf("[cargo:%s] Close: performing final flush, batchSize=%d",
		c.cfg.Name, len(toFlush))
	return c.flushFn(ctx, toFlush)
}

func (c *Cargo) startTimerLocked() {
	if c.timerActive || c.cfg.FlushInterval <= 0 {
		return
	}

	c.timerActive = true
	d := c.cfg.FlushInterval
	log.Printf("[cargo:%s] startTimerLocked: starting timer for %s", c.cfg.Name, d)

	c.timer = time.AfterFunc(d, c.onTimeout)
}

func (c *Cargo) stopTimerLocked() {
	if !c.timerActive {
		return
	}

	log.Printf("[cargo:%s] stopTimerLocked: stopping timer", c.cfg.Name)

	if c.timer != nil {
		c.timer.Stop()
	}

	c.timerActive = false
	c.timer = nil
}

func (c *Cargo) onTimeout() {
	var toFlush []JSONPayload

	c.mu.Lock()

	if !c.timerActive {
		log.Printf("[cargo:%s] onTimeout: timer not active, ignoring", c.cfg.Name)
		c.mu.Unlock()
		return
	}

	log.Printf("[cargo:%s] onTimeout: timer fired, batchLen=%d",
		c.cfg.Name, len(c.batch))

	if len(c.batch) > 0 {
		toFlush = make([]JSONPayload, len(c.batch))
		copy(toFlush, c.batch)
		c.batch = c.batch[:0]
	}

	c.timerActive = false
	c.timer = nil

	c.mu.Unlock()

	if len(toFlush) == 0 {
		log.Printf("[cargo:%s] onTimeout: nothing to flush", c.cfg.Name)
		return
	}

	log.Printf("[cargo:%s] onTimeout: performing timeout-based flush, batchSize=%d",
		c.cfg.Name, len(toFlush))

	_ = c.flushFn(context.Background(), toFlush)
}
