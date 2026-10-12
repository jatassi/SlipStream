package world

import (
	"context"
	"time"
)

func Launch(ctx context.Context) {
	t := time.NewTimer(time.Second)
	time.AfterFunc(time.Second, func() {})
	_ = context.WithoutCancel(ctx)
	_, _ = t, context.Background()
}
