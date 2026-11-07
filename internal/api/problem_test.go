package api

import (
	"testing"
	"time"
)

func TestTimeWordsAddTheDayOnlyOffNowsDayInTheZone(t *testing.T) {
	zone := time.FixedZone("UTC-5", -5*60*60)
	now := time.Date(2026, 11, 7, 20, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		t    time.Time
		want string
	}{
		{time.Date(2026, 11, 7, 19, 5, 0, 0, time.UTC), "14:05"},
		{time.Date(2026, 11, 7, 3, 5, 0, 0, time.UTC), "6 Nov, 22:05"},
		{time.Date(2026, 11, 8, 4, 59, 0, 0, time.UTC), "23:59"},
		{time.Date(2025, 10, 3, 14, 5, 0, 0, time.UTC), "3 Oct 2025, 09:05"},
	} {
		if got := timeWords(c.t, now, zone); got != c.want {
			t.Errorf("timeWords(%s) = %q, want %q", c.t, got, c.want)
		}
	}
}
