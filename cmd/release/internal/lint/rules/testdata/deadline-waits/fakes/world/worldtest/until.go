package worldtest

import (
	"context"
	"testing"
	"time"
)

func Context(t *testing.T) context.Context { return t.Context() }

func Until(t *testing.T, what string, cond func() (bool, error)) {
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	_ = context.Background()
	t.Skip("no link here")
}
