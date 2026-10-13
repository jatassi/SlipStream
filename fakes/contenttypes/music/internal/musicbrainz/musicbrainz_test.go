package musicbrainz

import "testing"

func TestCanonicalRelease(t *testing.T) {
	for _, c := range []struct {
		name     string
		releases []Release
		want     string
	}{
		{"none", nil, ""},
		{"earliest", []Release{{ID: "b", Date: "2002"}, {ID: "a", Date: "2001-05"}, {ID: "c"}}, "a"},
		{"no disambiguation before an earlier date", []Release{{ID: "a", Date: "2001", Disambiguation: "promo"}, {ID: "b", Date: "2004"}}, "b"},
		{"earliest disambiguated when every one is", []Release{{ID: "a", Date: "2003", Disambiguation: "x"}, {ID: "b", Date: "2001", Disambiguation: "y"}}, "b"},
		{"a tie broken by id", []Release{{ID: "d", Date: "2001-01-01"}, {ID: "c", Date: "2001-01-01"}}, "c"},
		{"undated last", []Release{{ID: "a"}, {ID: "b", Date: "2010"}}, "b"},
	} {
		got, ok := canonical(c.releases)
		if got.ID != c.want || ok != (c.want != "") {
			t.Errorf("%s: canonical = %q, %t; want %q", c.name, got.ID, ok, c.want)
		}
	}
}
