package sim

import (
	"context"
	"time"
)

func Launch(ctx context.Context) {
	time.Sleep(time.Second)
	_, _ = ctx, context.Background()
}
