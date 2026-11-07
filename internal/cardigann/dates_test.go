package cardigann

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
)

var (
	testZone  = time.FixedZone("UTC+2", 2*3600)
	testClock = Clock{Now: time.Date(2026, 10, 27, 15, 4, 5, 0, time.UTC), Location: testZone}
)

func local(y int, mo time.Month, d, h, mi, s int) time.Time {
	return time.Date(y, mo, d, h, mi, s, 0, testZone)
}

func TestFormatRFC1123Z(t *testing.T) {
	tests := []struct {
		in   time.Time
		want string
	}{
		{local(2021, 3, 4, 5, 6, 7), "Thu, 04 Mar 2021 05:06:07 +2"},
		{time.Date(2021, 3, 4, 5, 6, 7, 0, time.FixedZone("", -5*3600)), "Thu, 04 Mar 2021 05:06:07 -5"},
		{time.Date(2021, 3, 4, 5, 6, 7, 0, time.UTC), "Thu, 04 Mar 2021 05:06:07 +0"},
	}
	for _, tt := range tests {
		if got := formatRFC1123Z(tt.in); got != tt.want {
			t.Errorf("formatRFC1123Z(%v) = %q, want %q", tt.in, got, tt.want)
		}
		back, ok := datetimeutil.TryParseExact(tt.want, datetimeutil.Rfc1123ZPattern, datetimeutil.None, testClock)
		if !ok || !back.Equal(tt.in) {
			t.Errorf("reading %q back = %v, %v", tt.want, back, ok)
		}
	}
}
