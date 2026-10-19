package libtorrent

// #include "shim.h"
import "C"

import "unsafe"

// layout is cgo's size and field offsets of each struct that crosses the
// shim, which shim.cpp's static_asserts pin for the C++ side.
func layout() map[string][]uintptr {
	var (
		e C.lts_error
		s C.lts_status
		p C.lts_add_params
		a C.lts_alert
		f C.lts_file
	)
	return map[string][]uintptr{
		"lts_error":      {unsafe.Sizeof(e)},
		"lts_status":     {unsafe.Sizeof(s), unsafe.Offsetof(s.need_save), unsafe.Offsetof(s.error)},
		"lts_add_params": {unsafe.Sizeof(p), unsafe.Offsetof(p.seed_mode), unsafe.Offsetof(p.base_uploaded)},
		"lts_alert":      {unsafe.Sizeof(a), unsafe.Offsetof(a.error), unsafe.Offsetof(a.payload)},
		"lts_file":       {unsafe.Sizeof(f), unsafe.Offsetof(f.path)},
	}
}
