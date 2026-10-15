package host

import "testing"

func TestBelowFloor(t *testing.T) {
	for _, tc := range []struct{ what, release, want string }{
		{"kernel", "4.18.0-553.el8.x86_64", ""},
		{"kernel", "4.15.0-20-generic", "kernel 4.15.0-20-generic is older than 4.18, the oldest SlipStream supports"},
		{"glibc", "2.27", "glibc 2.27 is older than 2.28, the oldest SlipStream supports"},
		{"glibc", "2.40", ""},
		{"glibc", "", `cannot read the glibc version: "" is not a version`},
	} {
		floor := kernelFloor
		if tc.what == "glibc" {
			floor = glibcFloor
		}
		if got := belowFloor(tc.what, tc.release, floor); got != tc.want {
			t.Errorf("%s %s: %q, want %q", tc.what, tc.release, got, tc.want)
		}
	}
}
