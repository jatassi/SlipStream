package datetimeutil

import (
	"testing"
	"time"
	_ "time/tzdata"
)

func zone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func utc(y int, mo time.Month, d, h, mi, s int) time.Time {
	return time.Date(y, mo, d, h, mi, s, 0, time.UTC)
}

func TestParseExactAsTheSitesCallIt(t *testing.T) {
	now := utc(2026, 11, 7, 10, 0, 0)
	moscow := Clock{Now: now, Location: zone(t, "Europe/Moscow")}
	shanghai := Clock{Now: now, Location: zone(t, "Asia/Shanghai")}
	tests := []struct {
		site, in, format string
		style            DateTimeStyles
		clock            Clock
		want             time.Time
	}{
		{"TorrentBytes", "2026-11-0514:30:00", "yyyy-MM-ddHH:mm:ss", None, moscow, utc(2026, 11, 5, 11, 30, 0)},
		{"SceneHD", "2026-11-05 14:30:00", "yyyy-MM-dd HH:mm:ss", None, shanghai, utc(2026, 11, 5, 6, 30, 0)},
		{"AnimeBytes", "2026-11-05 14:30:00", "yyyy-MM-dd HH:mm:ss", AssumeUniversal, shanghai, utc(2026, 11, 5, 14, 30, 0)},
		{"MTeamTp", "2026-11-05 14:30:00 +08:00", "yyyy-MM-dd HH:mm:ss zzz", AssumeUniversal, moscow, utc(2026, 11, 5, 6, 30, 0)},
		{"Anidex", "2026-11-05 14:30:00 UTC", "yyyy-MM-dd HH:mm:ss UTC", AssumeUniversal, shanghai, utc(2026, 11, 5, 14, 30, 0)},
		{"SpeedCD", "Thursday, November 5, 2026 2:30PM", "dddd, MMMM d, yyyy h:mmtt", AssumeUniversal, moscow, utc(2026, 11, 5, 14, 30, 0)},
		{"ImmortalSeed", "2026-11-05 02:30 PM", "yyyy-MM-dd hh:mm tt", None, moscow, utc(2026, 11, 5, 11, 30, 0)},
		{"BakaBT", "05 Nov 26", "dd MMM yy", None, shanghai, utc(2026, 11, 4, 16, 0, 0)},
		{"HDTorrents", "05 Nov 2026 14:30:00", "dd MMM yyyy HH:mm:ss", None, moscow, utc(2026, 11, 5, 11, 30, 0)},
		{"XSpeeds", "05-11-2026 14:30", "dd-MM-yyyy HH:mm", None, moscow, utc(2026, 11, 5, 11, 30, 0)},
		{"Libble", "Nov 05 2026, 14:30", "MMM dd yyyy, HH:mm", AssumeUniversal, moscow, utc(2026, 11, 5, 14, 30, 0)},
		{"PixelHD", "Nov 05 2026, 14:30", "MMM dd yyyy, HH:mm", None, shanghai, utc(2026, 11, 5, 6, 30, 0)},
		{"Anidub", "5-11-2026, 14:30", "d-MM-yyyy, HH:mm", AssumeUniversal, moscow, utc(2026, 11, 5, 14, 30, 0)},
		{"MyAnonamouse", "2026-11-05 14:30:00", "yyyy-MM-dd HH:mm:ss", AssumeUniversal, moscow, utc(2026, 11, 5, 14, 30, 0)},
		{"HDBits", "2026 11/05", "yyyy MM/dd", None, moscow, utc(2026, 11, 4, 21, 0, 0)},
	}
	for _, tt := range tests {
		got, err := ParseExact(tt.in, tt.format, tt.style, tt.clock)
		if err != nil || !got.Equal(tt.want) {
			t.Errorf("%s: ParseExact(%q, %q) = %v, %v; want %v", tt.site, tt.in, tt.format, got, err, tt.want)
		}
		if err == nil && got.Location() != tt.clock.Location {
			t.Errorf("%s: ParseExact gave %v, want the clock's zone", tt.site, got.Location())
		}
	}
	if _, err := ParseExact("Friday, November 5, 2026 2:30PM", "dddd, MMMM d, yyyy h:mmtt", None, moscow); err == nil {
		t.Error("ParseExact read a Friday that is a Thursday")
	}
	if _, ok := TryParseExact("S01E02", "yyyy MM/dd", None, moscow); ok {
		t.Error("TryParseExact read an episode number as a date")
	}
}

func TestAdjustToUniversal(t *testing.T) {
	moscow := Clock{Now: utc(2026, 11, 7, 10, 0, 0), Location: zone(t, "Europe/Moscow")}
	got, err := ParseExact("2026-11-05 14:30", "yyyy-MM-dd HH:mm", AdjustToUniversal, moscow)
	if want := utc(2026, 11, 5, 11, 30, 0); err != nil || !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("AdjustToUniversal = %v, %v; want %v in UTC", got, err, want)
	}
	got, err = ParseExact("2026-11-05 14:30", "yyyy-MM-dd HH:mm", AssumeUniversal|AdjustToUniversal, moscow)
	if want := utc(2026, 11, 5, 14, 30, 0); err != nil || !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("AssumeUniversal|AdjustToUniversal = %v, %v; want %v in UTC", got, err, want)
	}
}

func TestParse(t *testing.T) {
	shanghai := Clock{Now: utc(2026, 11, 7, 10, 0, 0), Location: zone(t, "Asia/Shanghai")}
	tests := []struct {
		site, in string
		style    DateTimeStyles
		want     time.Time
	}{
		{"BeyondHD", "2021-11-14 23:26:21", AssumeUniversal, utc(2021, 11, 14, 23, 26, 21)},
		{"Avistaz", "2021-11-14T23:26:21+02:00", AdjustToUniversal, utc(2021, 11, 14, 21, 26, 21)},
		{"FileList", "2020-01-25 22:20:19 +0200", AdjustToUniversal, utc(2020, 1, 25, 20, 20, 19)},
		{"PassThePopcorn", "2016-10-18 23:40:59 +0000", AdjustToUniversal, utc(2016, 10, 18, 23, 40, 59)},
		{"Knaben", "2026-11-05T14:30:00+01:00", AssumeUniversal, utc(2026, 11, 5, 13, 30, 0)},
		{"Nebulance", "2026-11-05T14:30:00.123456Z", AssumeUniversal, utc(2026, 11, 5, 14, 30, 0).Add(123456 * time.Microsecond)},
		{"no offset", "2026-11-05 14:30", None, utc(2026, 11, 5, 6, 30, 0)},
		{"date only", "2026-11-05", AssumeUniversal, utc(2026, 11, 5, 0, 0, 0)},
		{"short offset", "2026-11-05T14:30:00 +8", None, utc(2026, 11, 5, 6, 30, 0)},
		{"invariant", "11/05/2026 2:30 PM", AssumeUniversal, utc(2026, 11, 5, 14, 30, 0)},
		{"RFC 1123", "Thu, 05 Nov 2026 14:30:00 GMT", None, utc(2026, 11, 5, 14, 30, 0)},
		{"SubsPlease", "Thu, 05 Nov 2026 14:30:00 +0100", None, utc(2026, 11, 5, 13, 30, 0)},
		{"RFC 1123 offset with a colon", "Thu, 05 Nov 2026 14:30:00 -05:00", AdjustToUniversal, utc(2026, 11, 5, 19, 30, 0)},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in, tt.style, shanghai)
		if err != nil || !got.Equal(tt.want) {
			t.Errorf("%s: Parse(%q) = %v, %v; want %v", tt.site, tt.in, got, err, tt.want)
		}
	}
	for _, in := range []string{"", "yesterday", "2026-13-05", "2026-11-05 25:00", "2026-11-05T14:30:00+15:00"} {
		if got, err := Parse(in, None, shanghai); err == nil {
			t.Errorf("Parse(%q) = %v; want an error", in, got)
		}
	}
}

func TestUnixTimestampToDateTime(t *testing.T) {
	shanghai := zone(t, "Asia/Shanghai")
	got := UnixTimestampToDateTime(1762353000, Clock{Location: shanghai})
	if want := utc(2025, 11, 5, 14, 30, 0); !got.Equal(want) || got.Location() != shanghai {
		t.Errorf("UnixTimestampToDateTime = %v, want %v in Shanghai", got, want)
	}
	if got := UnixTimestampToDateTime(0, Clock{Location: time.UTC}); !got.Equal(time.Unix(0, 0)) {
		t.Errorf("UnixTimestampToDateTime(0) = %v", got)
	}
}

func TestFromTimeAgoAsTheSitesWriteIt(t *testing.T) {
	moscow := Clock{Now: utc(2026, 11, 7, 10, 0, 0), Location: zone(t, "Europe/Moscow")}
	now := moscow.Now
	tests := []struct {
		site, in string
		want     time.Time
	}{
		{"IPTorrents", "2.5 hours ago ", now.Add(-150 * time.Minute)},
		{"IPTorrents", "1 week ago", now.Add(-7 * 24 * time.Hour)},
		{"PreToMe", "3 days, 4 hours", now.Add(-76 * time.Hour)},
		{"Libble", "5 mins ago", now.Add(-5 * time.Minute)},
		{"PixelHD", "1 year, 2 months ago", now.Add(-425 * 24 * time.Hour)},
		{"now", "Right now", now},
	}
	for _, tt := range tests {
		got, err := FromTimeAgo(tt.in, moscow)
		if err != nil || !got.Equal(tt.want) {
			t.Errorf("%s: FromTimeAgo(%q) = %v, %v; want %v", tt.site, tt.in, got, err, tt.want)
		}
	}
}

func TestFromFuzzyTime(t *testing.T) {
	moscow := Clock{Now: utc(2026, 11, 7, 10, 0, 0), Location: zone(t, "Europe/Moscow")}
	msk := func(y int, mo time.Month, d, h, mi int) time.Time {
		return time.Date(y, mo, d, h, mi, 0, 0, moscow.Location)
	}
	tests := []struct {
		site, in string
		want     time.Time
	}{
		{"Toloka", "2026-11-05 14:30", msk(2026, 11, 5, 14, 30)},
		{"Gazelle", "Nov 5 2026, 14:30", msk(2026, 11, 5, 14, 30)},
		{"Orpheus", "11/05/2026 02:30 PM", msk(2026, 11, 5, 14, 30)},
		{"US order", "05/11/2026", msk(2026, 5, 11, 0, 0)},
		// DateTimeRoutines knows no Russian: "Вчера" (yesterday) is lost and
		// the time is read as today's.
		{"Russian yesterday", "Вчера, 14:22", msk(2026, 11, 7, 14, 22)},
		{"Russian today", "Сегодня 09:05", msk(2026, 11, 7, 9, 5)},
	}
	for _, tt := range tests {
		got, err := FromFuzzyTime(tt.in, moscow)
		if err != nil || !got.Equal(tt.want) {
			t.Errorf("%s: FromFuzzyTime(%q) = %v, %v; want %v", tt.site, tt.in, got, err, tt.want)
		}
	}
	if got, err := FromFuzzyTime("Вчера", moscow); err == nil {
		t.Errorf("FromFuzzyTime(Вчера) = %v; want an error", got)
	}
}

func TestFromUnknownInMoscow(t *testing.T) {
	moscow := Clock{Now: utc(2026, 11, 7, 22, 30, 0), Location: zone(t, "Europe/Moscow")}
	msk := func(d, h, mi int) time.Time { return time.Date(2026, 11, d, h, mi, 0, 0, moscow.Location) }
	tests := []struct {
		in   string
		want time.Time
	}{
		{"today at 14:22", msk(7, 14, 22)},
		{"yesterday 14:22", msk(6, 14, 22)},
		{"1762353000", time.Unix(1762353000, 0)},
		{"Saturday at 08:00", msk(7, 8, 0)},
	}
	for _, tt := range tests {
		got, err := FromUnknown(tt.in, moscow)
		if err != nil || !got.Equal(tt.want) {
			t.Errorf("FromUnknown(%q) = %v, %v; want %v", tt.in, got, err, tt.want)
		}
	}
}
