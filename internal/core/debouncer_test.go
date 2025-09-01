package core

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestDebouncer_SingleTrigger(t *testing.T) {
	debouncer := NewDebouncer(50 * time.Millisecond)
	var executed int64

	// Trigger once
	debouncer.Trigger(func() {
		atomic.AddInt64(&executed, 1)
	})

	// Wait for execution
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt64(&executed) != 1 {
		t.Errorf("Expected 1 execution, got %d", executed)
	}
}

func TestDebouncer_MultipleTriggers(t *testing.T) {
	debouncer := NewDebouncer(50 * time.Millisecond)
	var executed int64

	// Trigger multiple times rapidly
	for i := 0; i < 5; i++ {
		debouncer.Trigger(func() {
			atomic.AddInt64(&executed, 1)
		})
		time.Sleep(10 * time.Millisecond) // Sleep less than debounce delay
	}

	// Wait for execution
	time.Sleep(100 * time.Millisecond)

	// Should only execute once due to debouncing
	if atomic.LoadInt64(&executed) != 1 {
		t.Errorf("Expected 1 execution due to debouncing, got %d", executed)
	}
}

func TestDebouncer_SequentialTriggers(t *testing.T) {
	debouncer := NewDebouncer(30 * time.Millisecond)
	var executed int64

	// First trigger
	debouncer.Trigger(func() {
		atomic.AddInt64(&executed, 1)
	})

	// Wait for first execution
	time.Sleep(50 * time.Millisecond)

	// Second trigger after delay
	debouncer.Trigger(func() {
		atomic.AddInt64(&executed, 1)
	})

	// Wait for second execution
	time.Sleep(50 * time.Millisecond)

	// Should execute twice
	if atomic.LoadInt64(&executed) != 2 {
		t.Errorf("Expected 2 executions, got %d", executed)
	}
}

func TestDebouncer_Cancel(t *testing.T) {
	debouncer := NewDebouncer(50 * time.Millisecond)
	var executed int64

	// Trigger
	debouncer.Trigger(func() {
		atomic.AddInt64(&executed, 1)
	})

	// Cancel immediately
	debouncer.Cancel()

	// Wait beyond debounce delay
	time.Sleep(100 * time.Millisecond)

	// Should not execute
	if atomic.LoadInt64(&executed) != 0 {
		t.Errorf("Expected 0 executions after cancel, got %d", executed)
	}
}

func TestDebouncer_IsActive(t *testing.T) {
	debouncer := NewDebouncer(50 * time.Millisecond)

	// Initially not active
	if debouncer.IsActive() {
		t.Error("Expected debouncer to be inactive initially")
	}

	// Trigger and check active
	debouncer.Trigger(func() {})

	if !debouncer.IsActive() {
		t.Error("Expected debouncer to be active after trigger")
	}

	// Wait for execution
	time.Sleep(100 * time.Millisecond)

	// Should be inactive after execution
	if debouncer.IsActive() {
		t.Error("Expected debouncer to be inactive after execution")
	}
}

func TestDebouncer_ConcurrentTriggers(t *testing.T) {
	debouncer := NewDebouncer(50 * time.Millisecond)
	var executed int64

	// Start multiple goroutines triggering concurrently
	for i := 0; i < 10; i++ {
		go func() {
			debouncer.Trigger(func() {
				atomic.AddInt64(&executed, 1)
			})
		}()
	}

	// Wait for all goroutines and execution
	time.Sleep(100 * time.Millisecond)

	// Should only execute once despite concurrent triggers
	if atomic.LoadInt64(&executed) != 1 {
		t.Errorf("Expected 1 execution despite concurrent triggers, got %d", executed)
	}
}

func TestDebouncer_RapidFileChanges(t *testing.T) {
	// Simulate rapid file changes like during npm install
	debouncer := NewDebouncer(100 * time.Millisecond)
	var snapshotCount int64

	// Simulate 100 rapid file changes
	for i := 0; i < 100; i++ {
		debouncer.Trigger(func() {
			atomic.AddInt64(&snapshotCount, 1)
		})
		time.Sleep(5 * time.Millisecond) // Very rapid changes
	}

	// Wait for debounced execution
	time.Sleep(200 * time.Millisecond)

	// Should create only 1 snapshot despite 100 changes
	if atomic.LoadInt64(&snapshotCount) != 1 {
		t.Errorf("Expected 1 snapshot for 100 rapid changes, got %d", snapshotCount)
	}
}

func TestDebouncer_DifferentDelays(t *testing.T) {
	tests := []struct {
		delay    time.Duration
		expected int64
	}{
		{10 * time.Millisecond, 1},
		{25 * time.Millisecond, 1},
		{50 * time.Millisecond, 1},
	}

	for _, test := range tests {
		debouncer := NewDebouncer(test.delay)
		var executed int64

		// Multiple rapid triggers
		for i := 0; i < 3; i++ {
			debouncer.Trigger(func() {
				atomic.AddInt64(&executed, 1)
			})
			time.Sleep(test.delay / 4) // Sleep less than delay
		}

		// Wait for execution
		time.Sleep(test.delay * 3)

		if atomic.LoadInt64(&executed) != test.expected {
			t.Errorf("For delay %v, expected %d executions, got %d", 
				test.delay, test.expected, executed)
		}
	}
}