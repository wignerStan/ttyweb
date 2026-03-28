package server

import (
	"testing"
	"time"
)

func TestCounterAddDone(t *testing.T) {
	c := newCounter(0)

	count := c.add(1)
	if count != 1 {
		t.Fatalf("Expected count=1 after add(1), got %d", count)
	}

	count = c.done()
	if count != 0 {
		t.Fatalf("Expected count=0 after done(), got %d", count)
	}
}

func TestCounterMultiple(t *testing.T) {
	c := newCounter(0)

	count := c.add(5)
	if count != 5 {
		t.Fatalf("Expected count=5 after add(5), got %d", count)
	}

	for i := 0; i < 3; i++ {
		count = c.done()
	}
	if count != 2 {
		t.Fatalf("Expected count=2 after 3 done() calls, got %d", count)
	}
}

func TestCounterTimerReset(t *testing.T) {
	c := newCounter(time.Second)

	// add(1) should stop the timer
	count := c.add(1)
	if count != 1 {
		t.Fatalf("Expected count=1 after add(1), got %d", count)
	}

	// done() should reset the timer since we go back to 0
	count = c.done()
	if count != 0 {
		t.Fatalf("Expected count=0 after done(), got %d", count)
	}

	// Verify the timer was reset — check if it has fired or is pending
	select {
	case <-c.timer().C:
		// Timer was not reset properly (already fired)
		t.Fatal("Timer should have been reset to 1 second, but it already fired")
	default:
		// Timer is pending as expected
	}

	// Stop the timer to clean up
	c.timer().Stop()
}

func TestCounterZeroDuration(t *testing.T) {
	// newCounter(0) should not panic — the timer event is drained in the constructor
	c := newCounter(0)

	if c == nil {
		t.Fatal("newCounter(0) returned nil")
	}

	count := c.count()
	if count != 0 {
		t.Fatalf("Expected initial count=0, got %d", count)
	}

	// add and done should work normally with zero duration
	count = c.add(2)
	if count != 2 {
		t.Fatalf("Expected count=2 after add(2), got %d", count)
	}

	count = c.done()
	if count != 1 {
		t.Fatalf("Expected count=1 after done(), got %d", count)
	}
}
