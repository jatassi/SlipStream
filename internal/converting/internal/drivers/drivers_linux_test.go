package drivers

import "testing"

func TestTheVAAPIDriverLine(t *testing.T) {
	for _, c := range []struct{ printed, want string }{
		{
			"[AVHWDeviceContext @ 0x5581] libva: VA-API version 1.22.0\n" +
				"[AVHWDeviceContext @ 0x5581] VAAPI driver: Intel iHD driver for Intel(R) Gen Graphics - 24.4.4 ().\n" +
				"At least one output file must be specified\n",
			"Intel iHD driver for Intel(R) Gen Graphics - 24.4.4 ()",
		},
		{"[AVHWDeviceContext @ 0x77] VAAPI driver: Mesa Gallium driver 24.2.8 for AMD Radeon 780M (radeonsi, phoenix, LLVM 19.1.1, DRM 3.59, 6.11.0).\n", "Mesa Gallium driver 24.2.8 for AMD Radeon 780M (radeonsi, phoenix, LLVM 19.1.1, DRM 3.59, 6.11.0)"},
		{"[VAAPI @ 0x55] Failed to open /dev/dri/renderD128 as DRM device node.\nDevice creation failed: -22.\n", ""},
	} {
		if got := vaapiDriver(c.printed); got != c.want {
			t.Errorf("vaapiDriver(%q) = %q, want %q", c.printed, got, c.want)
		}
	}
}
