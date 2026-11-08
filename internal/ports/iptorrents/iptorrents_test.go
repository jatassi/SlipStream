package iptorrents

import "testing"

func TestCleanTitle(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"The Quiet Orchard 2021 1080p WEB-DL H264-FKW", "The Quiet Orchard 2021 1080p WEB-DL H264-FKW"},
		{"[REQ] The Quiet Orchard 2021 720p WEB-DL H264-FKW", "The Quiet Orchard 2021 720p WEB-DL H264-FKW"},
		{"The Quiet Orchard 2021 1080p WEB-DL H264-FKW (Requested)", "The Quiet Orchard 2021 1080p WEB-DL H264-FKW"},
		{"{request} The Quiet Orchard 2021", "The Quiet Orchard 2021"},
		{"[ITA] Harbor Lights S02E03 1080p WEB-DL H264-FKW", "Harbor Lights S02E03 1080p WEB-DL H264-FKW"},
		{"[SubGroup] Harbor Lights - 03 [1080p]", "[SubGroup] Harbor Lights - 03 [1080p]"},
		{"Juniper Falls S01E01™ 1080p 🍿-FKW", "Juniper Falls S01E01 1080p -FKW"},
		{"Salt Meadow 2020 1080p -: ", "Salt Meadow 2020 1080p"},
	} {
		if got := cleanTitle(c.in); got != c.want {
			t.Errorf("cleanTitle(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
