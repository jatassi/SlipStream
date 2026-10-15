//go:build !linux

package host

import "context"

// BelowFloor is Linux's check; the other OSes' installers check the floor.
func BelowFloor(context.Context) []string { return nil }
