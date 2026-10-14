package front_test

import "golang.org/x/sys/windows"

// Windows reports a refused dial as WSAECONNREFUSED, which
// syscall.ECONNREFUSED does not match there.
const connRefused = windows.WSAECONNREFUSED
