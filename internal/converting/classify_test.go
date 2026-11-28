package converting

import (
	"os"
	"path/filepath"
	"testing"
)

// The tails are the bundled ffmpeg's last 2 KiB of error output, recorded
// on a Linux runner with no GPU.
func TestClassify(t *testing.T) {
	cases := []struct {
		tail string
		want string
	}{
		{"unreadable-invalid-data.txt", "unreadable"},
		{"unreadable-missing.txt", "unreadable"},
		{"device-qsv-no-session.txt", "device"},
		{"device-nvenc-no-cuda.txt", "device"},
		{"device-vaapi-no-display.txt", "device"},
		{"device-cuda-no-driver.txt", "device"},
		{"filter-bad-option.txt", "filter"},
		{"filter-no-hw-device.txt", "filter"},
		{"encoder-bad-profile.txt", "encoder"},
		{"disk_full-trailer.txt", "disk_full"},
		{"crash-segv.txt", "crash"},
	}
	for _, c := range cases {
		tail, err := os.ReadFile(filepath.Join("testdata", "tails", c.tail))
		if err != nil {
			t.Fatal(err)
		}
		if got := Classify(Exit{Code: 1, Tail: string(tail)}); got != c.want {
			t.Errorf("Classify(%s) = %q, want %q", c.tail, got, c.want)
		}
	}
}
