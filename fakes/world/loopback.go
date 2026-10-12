package world

import (
	"context"

	"github.com/jatassi/slipstream/fakes/world/internal/loopback"
)

// SetupLoopback makes 127.0.0.2 to 127.0.0.32 answer now and at every
// boot, and says what it changed. On macOS it needs root; elsewhere every
// loopback address already answers and it changes nothing.
func SetupLoopback(ctx context.Context) (string, error) { return loopback.Setup(ctx) }

// RemoveLoopback undoes SetupLoopback and says what it changed.
func RemoveLoopback(ctx context.Context) (string, error) { return loopback.Remove(ctx) }
