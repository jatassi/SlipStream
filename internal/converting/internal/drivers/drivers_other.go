//go:build !linux && !windows && !darwin

package drivers

import "context"

// Local reads no adapters on this OS.
func Local(string, []string) Reader {
	return func(context.Context) ([]Adapter, error) { return nil, nil }
}
