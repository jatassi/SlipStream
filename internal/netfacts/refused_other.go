//go:build !windows

package netfacts

import "syscall"

// refused is the errno of a connection the far end refused.
const refused = syscall.ECONNREFUSED
