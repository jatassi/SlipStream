package web_test

import (
	"maps"
	"testing"

	"github.com/jatassi/slipstream/web"
)

func TestPresets(t *testing.T) {
	want := map[string]web.Preset{
		"desktop": {Width: 1440, Height: 900, DeviceScaleFactor: 1},
		"phone":   {Width: 390, Height: 844, DeviceScaleFactor: 3, IsMobile: true, HasTouch: true},
	}
	if got := web.Presets(); !maps.Equal(got, want) {
		t.Fatalf("Presets() = %+v, want %+v", got, want)
	}
}
