package proc

import (
	"context"
	"fmt"
)

// WaitExit returns once the process pid has exited, whether or not it is
// this process's child, or when ctx ends. Its wait outlives a ctx that
// ends first, until the process exits.
func WaitExit(ctx context.Context, pid int) error {
	exited := make(chan error, 1)
	go func() { exited <- waitExit(pid) }()
	select {
	case err := <-exited:
		return err
	case <-ctx.Done():
		return fmt.Errorf("PID %d did not exit: %w", pid, ctx.Err())
	}
}
