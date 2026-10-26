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
		t C.lts_tracker
		l C.lts_limits
	)
	return map[string][]uintptr{
		"lts_error":      {unsafe.Sizeof(e)},
		"lts_status":     {unsafe.Sizeof(s), unsafe.Offsetof(s.need_save), unsafe.Offsetof(s.error)},
		"lts_add_params": {unsafe.Sizeof(p), unsafe.Offsetof(p.seed_mode), unsafe.Offsetof(p.discover), unsafe.Offsetof(p.base_uploaded)},
		"lts_alert":      {unsafe.Sizeof(a), unsafe.Offsetof(a.error), unsafe.Offsetof(a.payload), unsafe.Offsetof(a.at_ms)},
		"lts_file":       {unsafe.Sizeof(f), unsafe.Offsetof(f.path)},
		"lts_tracker":    {unsafe.Sizeof(t), unsafe.Offsetof(t.last_error), unsafe.Offsetof(t.start_sent)},
		"lts_limits":     {unsafe.Sizeof(l), unsafe.Offsetof(l.connections), unsafe.Offsetof(l.watch_incoming), unsafe.Offsetof(l.dht)},
	}
}
