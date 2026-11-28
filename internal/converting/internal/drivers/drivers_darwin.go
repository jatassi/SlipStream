package drivers

import (
	"context"
	"fmt"

	"golang.org/x/sys/unix"
)

// Local reads the OS build, which carries the macOS GPU drivers.
func Local(string, []string) Reader {
	return func(context.Context) ([]Adapter, error) {
		build, err := unix.Sysctl("kern.osversion")
		if err != nil {
			return nil, fmt.Errorf("read the OS build: %w", err)
		}
		return []Adapter{{Name: "macOS", Vendor: "Apple", Driver: build}}, nil
	}
}
