package libtorrent

import (
	"reflect"
	"testing"
)

// TestTheShimStructsMatchGo holds cgo's layout of the shim's structs to the
// one shim.cpp's static_asserts hold the C++ compiler to.
func TestTheShimStructsMatchGo(t *testing.T) {
	want := map[string][]uintptr{
		"lts_error":      {264},
		"lts_status":     {368, 96, 100},
		"lts_add_params": {80, 40, 48},
		"lts_alert":      {320, 16, 280, 312},
		"lts_file":       {24, 16},
		"lts_tracker":    {320, 32, 312},
		"lts_limits":     {32, 16, 24},
	}
	if got := layout(); !reflect.DeepEqual(got, want) {
		t.Fatalf("cgo lays the shim's structs out as %v, want %v as in shim.cpp's static_asserts", got, want)
	}
}
