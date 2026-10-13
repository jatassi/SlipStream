package musicbrainz

import (
	"errors"
	"testing"

	"github.com/jatassi/slipstream/fakes/contenttypes/music/musicmeta"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/works"
)

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

func TestOnlyAnArtistLookupFindsNoRecord(t *testing.T) {
	s, err := scenario.Load("music")
	if err != nil {
		t.Fatal(err)
	}
	f := faketest.Loopback(t, musicmeta.Fake, s)
	c := New(f.Client, endpoint.MustParseURL(f.Address.URL("/ws/2")), "SlipStream/test ( github.com/jatassi/SlipStream )")
	const unknown = "ffffffff-0000-4000-8000-000000000000"
	_, lookup := c.Artist(t.Context(), unknown)
	_, groups := c.Albums(t.Context(), unknown)
	_, _, releases := c.Canonical(t.Context(), unknown)
	for _, r := range []struct {
		err  error
		want string
		is   error
	}{
		{lookup, "GET /artist/ffffffff-0000-4000-8000-000000000000: 404 Not Found: the metadata source has no record with that key", works.ErrNotFound},
		{groups, "GET /release-group: 404 Not Found: the metadata source didn't answer", works.ErrUnavailable},
		{releases, "GET /release: 404 Not Found: the metadata source didn't answer", works.ErrUnavailable},
	} {
		if r.err == nil || r.err.Error() != r.want || !errors.Is(r.err, r.is) {
			t.Errorf("got %v, want %s wrapping %v", r.err, r.want, r.is)
		}
	}
}
