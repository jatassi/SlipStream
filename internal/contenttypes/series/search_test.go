package series

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/release"
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
		{"Lantern.Keepers.S01E03E04.1080p.WEB-DL", "S01E03E04", []works.ItemID{ep(3), ep(4)}},
		{"Lantern.Keepers.S01E02-E04.1080p.WEB-DL", "S01E02-E04", []works.ItemID{ep(2), ep(3), ep(4)}},
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

func TestAbsoluteOrderSearchesByAiredNumbers(t *testing.T) {
	sh := open(t)
	aired := newShow(7801, "Long Saga", 0, 2, 2)
	aired.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderAbsolute}
	tree := sh.add(t, aired)
	tree = sh.refresh(t, tree.ID, absoluteOf(aired, db.SeasonID(7801*100+50)))
	terms := searchTerms(t, sh, itemOf(t, tree, keyOf(7801*100000+2*1000+1)).ID)
	if terms.Season == nil || *terms.Season != 2 || terms.Episode != "1" {
		t.Fatalf("absolute episode 3 searches season %v ep %q, want its Aired S02E01", terms.Season, terms.Episode)
	}
	episode := itemOf(t, tree, keyOf(7801*100000+2*1000+1)).ID
	if got := covers(sh, terms, "Long.Saga.S02E01.1080p"); !slices.Equal(got, []works.ItemID{episode}) {
		t.Fatalf("S02E01 covers %v, want absolute episode 3", got)
	}
	if got := covers(sh, terms, "Long.Saga.S01E03.1080p"); slices.Contains(got, episode) {
		t.Fatalf("S01E03 covers %v, want another episode than absolute episode 3", got)
	}
}
