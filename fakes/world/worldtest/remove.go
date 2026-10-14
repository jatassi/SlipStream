package worldtest

import (
	"os"
	"time"
)

// removeAll is os.RemoveAll that waits out, for up to two seconds, Windows'
// hold on an executable whose process just exited, as t.TempDir's cleanup
// does (go.dev/issue/51442).
func removeAll(dir string) error {
	deadline := time.Now().Add(2 * time.Second)
	for {
		err := os.RemoveAll(dir)
		if !heldBriefly(err) || time.Now().After(deadline) {
			return err //nolint:wrapcheck // os.RemoveAll's error names the path
		}
		time.Sleep(20 * time.Millisecond)
	}
}
