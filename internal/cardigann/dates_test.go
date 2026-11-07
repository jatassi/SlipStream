package cardigann

import (
	"testing"
	"time"
)

var (
	testZone  = time.FixedZone("UTC+2", 2*3600)
	testClock = Clock{Now: time.Date(2026, 10, 27, 15, 4, 5, 0, time.UTC), Location: testZone}
)

func local(y int, mo time.Month, d, h, mi, s int) time.Time {
	return time.Date(y, mo, d, h, mi, s, 0, testZone)
}

func TestParseExact(t *testing.T) {
	tests := []struct {
		in, format string
		want       time.Time
	}{
		{"2021-03-04 05:06:07", "yyyy-MM-dd HH:mm:ss", local(2021, 3, 4, 5, 6, 7)},
		{"04/03/21", "dd/MM/yy", local(2021, 3, 4, 0, 0, 0)},
		{"4/3/50", "d/M/yy", local(1950, 3, 4, 0, 0, 0)},
		{"4/3/49", "d/M/yy", local(2049, 3, 4, 0, 0, 0)},
		{"4/3/7", "d/M/y", local(2007, 3, 4, 0, 0, 0)},
		{"17:30", "HH:mm", local(2026, 10, 27, 17, 30, 0)},
		{"Mar 4, 2021 5:06 PM", "MMM d, yyyy h:mm tt", local(2021, 3, 4, 17, 6, 0)},
		{"12:15 AM", "hh:mm tt", local(2026, 10, 27, 0, 15, 0)},
		{"13:00 PM", "HH:mm tt", local(2026, 10, 27, 13, 0, 0)},
		{"2021-03-04 05:06 +0530", "yyyy-MM-dd HH:mm zzz", time.Date(2021, 3, 3, 23, 36, 0, 0, time.UTC)},
		{"2021-03-04 05:06 +05", "yyyy-MM-dd HH:mm zz", time.Date(2021, 3, 4, 0, 6, 0, 0, time.UTC)},
		{"March 4 2021", "MMMM d yyyy", local(2021, 3, 4, 0, 0, 0)},
		{"Thursday 04.03.2021", "dddd dd.MM.yyyy", local(2021, 3, 4, 0, 0, 0)},
		{"2021-03-04T05:06:07+05:30", "yyyy-MM-ddTHH:mm:sszzz", time.Date(2021, 3, 3, 23, 36, 7, 0, time.UTC)},
		{"2021-03-04 05:06:07 -5", "yyyy-MM-dd HH:mm:ss z", time.Date(2021, 3, 4, 10, 6, 7, 0, time.UTC)},
		{"Thu, 04 Mar 2021 05:06:07 +2", rfc1123ZFormat, time.Date(2021, 3, 4, 3, 6, 7, 0, time.UTC)},
		{"05:06:07.250", "HH:mm:ss.fff", local(2026, 10, 27, 5, 6, 7).Add(250 * time.Millisecond)},
		{"2021 at 05h", "yyyy 'at' HH'h'", local(2021, 1, 1, 5, 0, 0)},
		{"2021-03", "yyyy-MM", local(2021, 3, 1, 0, 0, 0)},
	}
	for _, tt := range tests {
		got, ok := parseExact(tt.in, tt.format, testClock)
		if !ok || !got.Equal(tt.want) {
			t.Errorf("parseExact(%q, %q) = %v, %v; want %v", tt.in, tt.format, got, ok, tt.want)
		}
	}
}

func TestParseExactRejects(t *testing.T) {
	tests := []struct{ in, format string }{
		{"Tue, 04 Mar 2021", "ddd, dd MMM yyyy"},
		{"2021-02-30", "yyyy-MM-dd"},
		{"2021-03-04 extra", "yyyy-MM-dd"},
		{"2021/03/04", "yyyy-MM-dd"},
		{"09:00 PM", "HH:mm tt"},
		{"13:00 PM", "hh:mm tt"},
		{"24:00", "HH:mm"},
		{"21-3-4", "yyyy-M-d"},
	}
	for _, tt := range tests {
		if got, ok := parseExact(tt.in, tt.format, testClock); ok {
			t.Errorf("parseExact(%q, %q) = %v; want a failure", tt.in, tt.format, got)
		}
	}
}

func TestParseDateLayout(t *testing.T) {
	tests := []struct {
		in, layout string
		want       time.Time
	}{
		{"2021-03-04 05:06", "2006-01-02 15:04", local(2021, 3, 4, 5, 6, 0)},
		{"Mar 4, 2021", "Jan 2, 2006", local(2021, 3, 4, 0, 0, 0)},
		{"04.03.2021", "dd.MM.yyyy", local(2021, 3, 4, 0, 0, 0)},
		{" 2021-03-04 ", "2006-01-02", local(2021, 3, 4, 0, 0, 0)},
		{"4 March 2021 5:06pm", "2 January 2006 3:04pm", local(2021, 3, 4, 17, 6, 0)},
	}
	for _, tt := range tests {
		got, err := parseDateLayout(tt.in, tt.layout, testClock)
		if err != nil || !got.Equal(tt.want) {
			t.Errorf("parseDateLayout(%q, %q) = %v, %v; want %v", tt.in, tt.layout, got, err, tt.want)
		}
	}
	if _, err := parseDateLayout("yesterday", "2006-01-02", testClock); err == nil {
		t.Error("parseDateLayout(yesterday): want an error")
	}
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
		back, ok := parseExact(tt.want, rfc1123ZFormat, testClock)
		if !ok || !back.Equal(tt.in) {
			t.Errorf("reading %q back = %v, %v", tt.want, back, ok)
		}
	}
}

func TestFromTimeAgo(t *testing.T) {
	now := testClock.local()
	tests := []struct {
		in   string
		want time.Time
	}{
		{"2 hours ago", now.Add(-2 * time.Hour)},
		{"1 day, 2 hours and 3 minutes ago", now.Add(-26*time.Hour - 3*time.Minute)},
		{"1.5 weeks", now.Add(-252 * time.Hour)},
		{"5 min", now.Add(-5 * time.Minute)},
		{"3 mo", now.Add(-90 * 24 * time.Hour)},
		{"1 year", now.Add(-365 * 24 * time.Hour)},
		{"10s", now.Add(-10 * time.Second)},
		{"Just now", now},
	}
	for _, tt := range tests {
		got, err := fromTimeAgo(tt.in, testClock)
		if err != nil || !got.Equal(tt.want) {
			t.Errorf("fromTimeAgo(%q) = %v, %v; want %v", tt.in, got, err, tt.want)
		}
	}
	if _, err := fromTimeAgo("3 fortnights ago", testClock); err == nil {
		t.Error("fromTimeAgo(fortnights): want an error")
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		in   string
		want time.Time
	}{
		{"Thu, 04 Mar 2021 05:06:07 +2", time.Date(2021, 3, 4, 3, 6, 7, 0, time.UTC)},
		{"1614834367", time.Unix(1614834367, 0)},
		{"Just now", testClock.Now},
		{"2 hours ago", testClock.Now.Add(-2 * time.Hour)},
		{"today 14:22", local(2026, 10, 27, 14, 22, 0)},
		{"Today, at 2:22 PM", local(2026, 10, 27, 14, 22, 0)},
		{"yesterday at 1:30 PM", local(2026, 10, 26, 13, 30, 0)},
		{"Tomorrow", local(2026, 10, 28, 0, 0, 0)},
		{"Saturday at 14:22", local(2026, 10, 24, 14, 22, 0)},
		{"Tuesday at 09:00", local(2026, 10, 27, 9, 0, 0)},
		{"03-04 10:00", local(2026, 3, 4, 10, 0, 0)},
		{"4 Mar 10:30", local(2026, 3, 4, 10, 30, 0)},
		{"03/04/2021 10:00", local(2021, 3, 4, 10, 0, 0)},
		{"Mar 4th, 2021", local(2021, 3, 4, 0, 0, 0)},
		{"2021-03-04T05:06:07", local(2021, 3, 4, 5, 6, 7)},
		{"04-Mar-2021 05:06 PM", local(2021, 3, 4, 17, 6, 0)},
		{"2021 Mar 4", local(2021, 3, 4, 0, 0, 0)},
		{"Uploaded Mar 4", local(2026, 3, 4, 0, 0, 0)},
		{"10:15", local(2026, 10, 27, 10, 15, 0)},
	}
	for _, tt := range tests {
		got, err := parseDate(tt.in, testClock)
		if err != nil || !got.Equal(tt.want) {
			t.Errorf("parseDate(%q) = %v, %v; want %v", tt.in, got, err, tt.want)
		}
	}
	for _, in := range []string{"garbage", "4 jul 2021", "JUL 4 2021", "today 25:00"} {
		if got, err := parseDate(in, testClock); err == nil {
			t.Errorf("parseDate(%q) = %v; want an error", in, got)
		}
	}
}
