//go:build unix

package front_test

import "syscall"

const connRefused = syscall.ECONNREFUSED
