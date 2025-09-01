package core

import (
	"sync"
	"time"
)

// Debouncer groups rapid events together to prevent spam
// Critical for preventing hundreds of snapshots during npm install, etc.
type Debouncer struct {
	delay time.Duration
	timer *time.Timer
	mu    sync.Mutex
}

// NewDebouncer creates a new debouncer with the specified delay
func NewDebouncer(delay time.Duration) *Debouncer {
	return &Debouncer{
		delay: delay,
	}
}

// Trigger schedules a function to be executed after the debounce delay
// If called again before the delay expires, the previous call is cancelled
// This ensures rapid changes create only ONE snapshot
func (d *Debouncer) Trigger(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Cancel existing timer if any
	if d.timer != nil {
		d.timer.Stop()
	}

	// Create new timer with delay
	d.timer = time.AfterFunc(d.delay, func() {
		fn()
		// Clear timer after execution
		d.mu.Lock()
		d.timer = nil
		d.mu.Unlock()
	})
}

// Cancel stops any pending execution
func (d *Debouncer) Cancel() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}

// IsActive returns true if there's a pending execution
func (d *Debouncer) IsActive() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.timer != nil
}