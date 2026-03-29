package server

import (
	"context"
	"testing"
	"time"
)

func TestWithGracefullContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	graceCtx, graceCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer graceCancel()

	opt := WithGracefullContext(graceCtx)
	opts := &RunOptions{}
	opt(opts)

	if opts.gracefullCtx != graceCtx {
		t.Fatal("expected gracefullCtx to be set")
	}

	// Verify the context is usable.
	select {
	case <-opts.gracefullCtx.Done():
		t.Fatal("gracefullCtx should not be done yet")
	default:
	}
	_ = ctx
}
