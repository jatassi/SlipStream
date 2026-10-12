//go:build !windows

package secret

import (
	"errors"
	"fmt"
	"os"
)

// syncDir makes a rename into dir durable.
func syncDir(dir string) error {
	d, err := os.Open(dir) //nolint:gosec // dir is the data dir, chosen by the Node
	if err != nil {
		return fmt.Errorf("sync %s: %w", dir, err)
	}
	if err := d.Sync(); err != nil {
		return errors.Join(fmt.Errorf("sync %s: %w", dir, err), d.Close())
	}
	return d.Close() //nolint:wrapcheck // closing a directory opened only to sync it
}
