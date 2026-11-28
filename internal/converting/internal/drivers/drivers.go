// Package drivers reads the video adapters a Node has and the version of
// each one's driver, which goes into the Node's ffmpeg signature.
package drivers

import "context"

// Adapter is one video adapter. Driver is its driver's version as the OS
// reports it: the display driver's version on Windows, the VA-API driver
// string or the NVIDIA module version on Linux, the OS build on macOS.
type Adapter struct {
	Name, Vendor, Driver string
}

type Reader func(ctx context.Context) ([]Adapter, error)
