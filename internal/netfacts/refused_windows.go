package netfacts

import "golang.org/x/sys/windows"

// refused is the errno of a connection the far end refused.
const refused = windows.WSAECONNREFUSED
