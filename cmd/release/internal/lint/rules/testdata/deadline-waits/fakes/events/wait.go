package events

import (
	"context"
	"time"
)

func Wait(ctx context.Context) {
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	<-ctx.Done()
}
