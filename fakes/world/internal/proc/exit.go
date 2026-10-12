package proc

import (
	"context"
	"fmt"
)

func waitFor(ctx context.Context, pid int, exited <-chan error) error {
	select {
	case err := <-exited:
		if err != nil {
			return fmt.Errorf("wait for PID %d: %w", pid, err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("PID %d did not exit: %w", pid, ctx.Err())
	}
}
