package series

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestEpisodeCover(t *testing.T) {
	sh := open(t)
	tree := sh.add(t, newShow(7601, "Lantern Keepers", 0, 4))
	s01e03 := searchTerms(t, sh, itemOf(t, tree, keyOf(7601*100000+1*1000+3)).ID)
	for _, tc := range []struct {
		title string
		want  indexers.Covering
	}{
		{"Lantern.Keepers.S01E03.1080p.WEB-DL", indexers.Covering{Kind: indexers.CoversItem}},
		{"lantern keepers s01e03 720p", indexers.Covering{Kind: indexers.CoversItem}},
		{"Lantern.Keepers.S01E02.1080p.WEB-DL", indexers.Covering{Kind: indexers.CoversOther, Names: "S01E02"}},
		{"Lantern.Keepers.S02E03.1080p.WEB-DL", indexers.Covering{Kind: indexers.CoversOther, Names: "S02E03"}},
		{"Lantern.Keepers.S01.1080p.WEB-DL", indexers.Covering{Kind: indexers.CoversMore, Names: "S01"}},
		{"Lantern.Keepers.S01E03E04.1080p.WEB-DL", indexers.Covering{Kind: indexers.CoversMore, Names: "S01E03E04"}},
		{"Lantern.Keepers.S01E03-E04.1080p.WEB-DL", indexers.Covering{Kind: indexers.CoversMore, Names: "S01E03-E04"}},
		{"Lantern.Keepers.1080p.WEB-DL", indexers.Covering{Kind: indexers.CoversOther}},
	} {
		if got := s01e03.Covers(tc.title); got != tc.want {
			t.Errorf("S01E03 covers %s as %+v, want %+v", tc.title, got, tc.want)
		}
	}
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
		if terms.Mode != "tvsearch" || terms.IDs["tvdbid"] != "7701" || len(terms.IDs) != 1 || terms.Query != "Lantern Keepers" ||
			terms.Season == nil || *terms.Season != tc.season || terms.Episode != tc.episode {
			t.Fatalf("%s's terms are %+v (season %v), want tvsearch by tvdbid 7701, season %d and ep %s", tc.name, terms, terms.Season, tc.season, tc.episode)
		}
		if got := terms.Covers(tc.covered).Kind; got != indexers.CoversItem {
			t.Errorf("%s's terms cover %s as %d, want the episode", tc.name, tc.covered, got)
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
	if got := terms.Covers("Long.Saga.S02E01.1080p"); got.Kind != indexers.CoversItem {
		t.Fatalf("absolute episode 3 covers S02E01 as %+v, want the episode", got)
	}
	if got := terms.Covers("Long.Saga.S01E03.1080p"); got.Kind != indexers.CoversOther {
		t.Fatalf("absolute episode 3 covers S01E03 as %+v, want another episode", got)
	}
}
