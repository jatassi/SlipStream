package fvad

import "errors"

// Version names the vendored libfvad. A change makes every map stale.
const Version = "libfvad 0.0.0+532ab66"

var (
	ErrMode  = errors.New("a voice detector mode is 0 to 3")
	ErrRate  = errors.New("a voice detector reads 8, 16, 32 or 48 kHz")
	ErrFrame = errors.New("a voice detector frame is 10, 20 or 30 ms")
	ErrNoVAD = errors.New("this build has no voice detector")
)
