package server

import (
	"context"
	"testing"
	"time"
)

func TestWithGracefulContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	graceCtx, graceCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer graceCancel()

	opt := WithGracefulContext(graceCtx)
	opts := &RunOptions{}
	opt(opts)

	if opts.gracefulCtx != graceCtx {
		t.Fatal("expected gracefulCtx to be set")
	}

	// Verify the context is usable.
	select {
	case <-opts.gracefulCtx.Done():
		t.Fatal("gracefulCtx should not be done yet")
	default:
	}
	_ = ctx
}
