package server

import (
	"testing"
)

func TestCounterWait(t *testing.T) {
	c := newCounter(0)
	c.add(1)
	done := make(chan struct{})
	go func() {
		c.done()
		close(done)
	}()
	c.wait()
}
