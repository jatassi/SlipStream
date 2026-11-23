package worldtest

import (
	"context"
	"os"
	"time"
)

// removeAll is os.RemoveAll that waits out, for up to two seconds, Windows'
// hold on an executable whose process just exited, as t.TempDir's cleanup
// does (go.dev/issue/51442). A hold that outlasts the wait names the
// processes that hold the file.
func removeAll(ctx context.Context, dir string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	// poll fails only once the hold outlasts ctx, and err is then that hold.
	var err error
	_ = poll(ctx, func() (bool, error) {
		err = os.RemoveAll(dir)
		return !heldBriefly(err), nil
	})
	return withHolders(err)
}
