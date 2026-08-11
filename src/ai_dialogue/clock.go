package aidialogue

import (
	"sync"
	"time"
)

// Clock abstracts time for deterministic testing. The state machine uses it
// to measure elapsed time for timeout detection. Tests inject a FakeClock
// so real 25-second waits are never needed (AC-17).
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// Since returns the duration elapsed since t.
	Since(t time.Time) time.Duration
}

// RealClock uses the system wall clock. Used in production.
type RealClock struct{}

func (RealClock) Now() time.Time                     { return time.Now() }
func (RealClock) Since(t time.Time) time.Duration     { return time.Since(t) }

// FakeClock is a manually-advanced clock for deterministic tests.
// Safe for concurrent use.
type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewFakeClock returns a FakeClock initialized to the given time.
// If start is zero, uses a fixed reference time.
func NewFakeClock(start time.Time) *FakeClock {
	if start.IsZero() {
		start = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	}
	return &FakeClock{now: start}
}

// Now returns the current fake time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Since returns the duration elapsed since t according to the fake clock.
func (c *FakeClock) Since(t time.Time) time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now.Sub(t)
}

// Advance moves the fake clock forward by d.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
