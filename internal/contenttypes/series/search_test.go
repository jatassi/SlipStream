package series

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestEpisodeCover(t *testing.T) {
	sh := open(t)
	tree := sh.add(t, newShow(7601, "Lantern Keepers", 0, 4))
	ep := func(n int64) works.ItemID { return itemOf(t, tree, keyOf(7601*100000+1*1000+n)).ID }
	s01e03 := searchTerms(t, sh, ep(3))
	var work indexers.Expect
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		work, err = sh.s.Expect(ctx, r, tree.ID)
		return err
	})
	if !reflect.DeepEqual(work.Claim, s01e03.Claim) {
		t.Fatalf("the Work expects %+v and S01E03's terms %+v, want one claim", work.Claim, s01e03.Claim)
	}
	for _, tc := range []struct {
		title, words string
		want         []works.ItemID
	}{
		{"Lantern.Keepers.S01E03.1080p.WEB-DL", "S01E03", []works.ItemID{ep(3)}},
		{"lantern keepers s01e03 720p", "S01E03", []works.ItemID{ep(3)}},
		{"Lantern.Keepers.S01E02.1080p.WEB-DL", "S01E02", []works.ItemID{ep(2)}},
		{"Lantern.Keepers.S02E03.1080p.WEB-DL", "S02E03", nil},
		{"Lantern.Keepers.S01.1080p.WEB-DL", "S01", []works.ItemID{ep(1), ep(2), ep(3), ep(4)}},
		{"Lantern.Keepers.S01E03E04.1080p.WEB-DL", "S01E03-E04", []works.ItemID{ep(3), ep(4)}},
		{"Lantern.Keepers.S01E02-E04.1080p.WEB-DL", "S01E02-E04", []works.ItemID{ep(2), ep(3), ep(4)}},
		{"Lantern.Keepers.1x01x02.720p.HDTV", "S01E01-E02", []works.ItemID{ep(1), ep(2)}},
		{"Lantern Keepers S01-S02 1080p", "S01-S02", []works.ItemID{ep(1), ep(2), ep(3), ep(4)}},
		{"Lantern.Keepers.S01.Part.2.720p", "S01 Part 2", nil},
		{"Lantern.Keepers.1080p.WEB-DL", "", nil},
	} {
		name := release.Title(tc.title, sh.s, nil).Name
		var words string
		if len(name.Marks) > 0 {
			words = name.Marks[0].Words
		}
		if got := s01e03.Covers(name); words != tc.words || !slices.Equal(got, tc.want) {
			t.Errorf("%s names %q and covers %v, want %q covering %v", tc.title, words, got, tc.words, tc.want)
		}
		if got := work.Covers(name); !slices.Equal(got, tc.want) {
			t.Errorf("by the Work's Expect %s covers %v, want %v", tc.title, got, tc.want)
		}
	}
}

// covers is the episodes title covers by terms, read as a series reads it.
func covers(sh shelf, terms indexers.Terms, title string) []works.ItemID {
	return terms.Covers(release.Title(title, sh.s, nil).Name)
}

func searchTerms(t *testing.T, sh shelf, item works.ItemID) indexers.Terms {
	t.Helper()
	var terms indexers.Terms
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		terms, err = sh.s.SearchTerms(ctx, r, item)
		return err
	})
	return terms
}

func TestSearchTermsAskForTheEpisodeByTVDBIDSeasonAndNumber(t *testing.T) {
	sh := open(t)
	tree := sh.add(t, newShow(7701, "Lantern Keepers", 1, 3))
	for _, tc := range []struct {
		name    string
		key     works.Key
		season  int
		episode string
		covered string
	}{
		{"an episode", keyOf(7701*100000 + 1*1000 + 3), 1, "3", "Lantern.Keepers.S01E03.1080p"},
		{"a special", keyOf(7701*100000 + 1), 0, "1", "Lantern.Keepers.S00E01.1080p"},
	} {
		terms := searchTerms(t, sh, itemOf(t, tree, tc.key).ID)
		if terms.Mode != "tvsearch" || terms.Claim.IDs["tvdbid"] != "7701" || len(terms.Claim.IDs) != 1 || terms.Query != "Lantern Keepers" ||
			terms.Season == nil || *terms.Season != tc.season || terms.Episode != tc.episode {
			t.Fatalf("%s's terms are %+v (season %v), want tvsearch by tvdbid 7701, season %d and ep %s", tc.name, terms, terms.Season, tc.season, tc.episode)
		}
		if got := covers(sh, terms, tc.covered); !slices.Equal(got, []works.ItemID{itemOf(t, tree, tc.key).ID}) {
			t.Errorf("%s's terms cover %v with %s, want the episode", tc.name, got, tc.covered)
		}
	}
	if got := sh.s.Categories(); len(got) == 0 || !newznab.Meets([]newznab.ID{5040}, got) || newznab.Meets([]newznab.ID{2000}, got) {
		t.Fatalf("Categories = %v, want the TV family 5000", got)
	}
}

func TestAbsoluteOrderSearchesAndCoversByAiredNumbers(t *testing.T) {
	sh := open(t)
	aired := newShow(7801, "Long Saga", 0, 2, 2)
	aired.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderAbsolute}
	tree := sh.add(t, aired)
	tree = sh.refresh(t, tree.ID, absoluteOf(aired, db.SeasonID(7801*100+50)))
	abs := func(season, n int64) works.ItemID { return itemOf(t, tree, keyOf(7801*100000+season*1000+n)).ID }
	terms := searchTerms(t, sh, abs(2, 1))
	if terms.Season == nil || *terms.Season != 2 || terms.Episode != "1" {
		t.Fatalf("absolute episode 3 searches season %v ep %q, want its Aired S02E01", terms.Season, terms.Episode)
	}
	for _, tc := range []struct {
		title string
		want  []works.ItemID
	}{
		{"Long.Saga.S02E01.1080p", []works.ItemID{abs(2, 1)}},
		{"Long.Saga.S01E03.1080p", nil},
		{"Long.Saga.S01E01E02.720p.HDTV", []works.ItemID{abs(1, 1), abs(1, 2)}},
		{"Long.Saga.S02.1080p.WEB-DL", []works.ItemID{abs(2, 1), abs(2, 2)}},
		{"[SubsPlease] Long Saga - 03 (1080p)", []works.ItemID{abs(2, 1)}},
		{"[SubsPlease] Long Saga - 03-04 (1080p)", []works.ItemID{abs(2, 1), abs(2, 2)}},
		{"[SubsPlease] Long Saga - 05 (1080p)", nil},
	} {
		if got := covers(sh, terms, tc.title); !slices.Equal(got, tc.want) {
			t.Errorf("in Absolute order %s covers %v, want %v", tc.title, got, tc.want)
		}
	}
}

func TestExpectClaimsTheSeriesTitlesYearSlackAndOriginalLanguage(t *testing.T) {
	sh := open(t)
	r := newShow(7901, "Night Shift", 0, 2)
	r.Show.OriginalTitle, r.Show.OriginalLanguage, r.Show.Aliases = "Yakgeun", "kor", []string{"Night Shift", "The Night Shift"}
	tree := sh.add(t, r)
	claim := func() indexers.Terms {
		t.Helper()
		return searchTerms(t, sh, tree.Items[0].ID)
	}
	got := claim()
	if !slices.Equal(got.Claim.Titles, []string{"Night Shift", "Yakgeun", "The Night Shift"}) || got.Claim.Year != release.Known(2022) || got.Claim.Slack != 1 {
		t.Fatalf("the claim is %+v, want the English title, original title and new alias, first aired 2022, slack 1", got.Claim)
	}
	if got.Original != release.Known(release.Language("ko")) {
		t.Fatalf("the original language is %+v, want ko from TVDB's kor", got.Original)
	}

	slack := func(raw string) (setting.CountField, error) {
		t.Helper()
		fields, err := sh.works.SetSettings(t.Context(), identity.Admin{}, Def.ID(), setting.Patch{"series.yearSlack": json.RawMessage(raw)})
		for _, f := range fields {
			if f, ok := f.(setting.CountField); ok && f.Key == "series.yearSlack" {
				return f, err
			}
		}
		return setting.CountField{}, err
	}
	for _, tc := range []struct {
		raw  string
		want int
	}{{"0", 0}, {"10", 10}, {"null", 1}} {
		f, err := slack(tc.raw)
		if err != nil {
			t.Fatalf("a year slack of %s = %v", tc.raw, err)
		}
		if f.Label != "Release year can differ by" || f.Value != int64(tc.want) || f.Default != 1 || f.IsDefault != (tc.raw == "null") {
			t.Fatalf("after %s the year slack field is %+v", tc.raw, f)
		}
		if got := claim().Claim.Slack; got != tc.want {
			t.Fatalf("after a year slack of %s the claim's slack is %d, want %d", tc.raw, got, tc.want)
		}
	}
	for _, raw := range []string{"-1", "11"} {
		var refused *setting.RangeError
		if _, err := slack(raw); !errors.As(err, &refused) {
			t.Fatalf("a year slack of %s = %v, want a RangeError", raw, err)
		}
	}
}
