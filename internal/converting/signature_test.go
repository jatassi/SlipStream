package converting

import (
	"slices"
	"testing"
)

func TestTheSignatureChangesWithADriverAndNotWithDeviceOrder(t *testing.T) {
	const version = "ffmpeg version 8.1.3-Jellyfin"
	intel := Device{Name: "Intel(R) UHD Graphics 750", Vendor: "Intel Corporation", Driver: "32.0.101.7085"}
	nvidia := Device{Name: "NVIDIA GeForce RTX 3060", Vendor: "NVIDIA", Driver: "32.0.15.6094"}
	base := sign(version, []Device{intel, nvidia})
	if len(base) != 64 {
		t.Fatalf("sign = %q, want 64 hex characters", base)
	}
	updated := intel
	updated.Driver = "32.0.101.6987"
	renamed := intel
	renamed.Name = "Intel(R) UHD Graphics"
	for _, c := range []struct {
		name    string
		version string
		devices []Device
		same    bool
	}{
		{"the devices in another order", version, []Device{nvidia, intel}, true},
		{"a device renamed", version, []Device{renamed, nvidia}, true},
		{"a driver updated", version, []Device{updated, nvidia}, false},
		{"a device removed", version, []Device{intel}, false},
		{"another ffmpeg", "ffmpeg version 8.1.4-Jellyfin", []Device{intel, nvidia}, false},
		{"two drivers joined into one", version, []Device{{Driver: "32.0.101.708532.0.15.6094"}}, false},
	} {
		if got := sign(c.version, slices.Clone(c.devices)); (got == base) != c.same {
			t.Errorf("%s: sign = %s against %s, want same %v", c.name, got, base, c.same)
		}
	}
}
