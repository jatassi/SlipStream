package series

import (
	"context"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func groupsOf(t *testing.T, sh shelf, work works.WorkID, items ...works.ItemID) []indexers.Group {
	t.Helper()
	var groups []indexers.Group
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		groups, err = sh.s.Groups(ctx, r, work, items)
		return err
	})
	return groups
}

func groupTerms(t *testing.T, sh shelf, work works.WorkID, g indexers.Group) indexers.Terms {
	t.Helper()
	var terms indexers.Terms
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		terms, err = sh.s.GroupTerms(ctx, r, work, g)
		return err
	})
	return terms
}

// sqlLevelKey is the group key a migration builds in SQL for the level of
// TVDB season id.
func sqlLevelKey(t *testing.T, sh shelf, id db.SeasonID) indexers.GroupKey {
	t.Helper()
	var key string
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, `SELECT 'l:' || lower(hex(level_id)) FROM series_season WHERE tvdb_season_id = ?`, int64(id)).Scan(&key)
	})
	return indexers.GroupKey(key)
}

func assertGroups(t *testing.T, got, want []indexers.Group) {
	t.Helper()
	if !slices.EqualFunc(got, want, func(a, b indexers.Group) bool {
		return a.Key == b.Key && a.Label == b.Label && slices.Equal(a.Items, b.Items)
	}) {
		t.Fatalf("groups are\n%+v\nwant\n%+v", got, want)
	}
}

func TestGroupsAreTheLevelsInAiredAndDVDOrderWithEachSpecialAlone(t *testing.T) {
	sh := open(t)
	aired := newShow(7101, "Harbor Lights", 2, 3, 2)
	aired.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderDVD}
	tree := sh.add(t, aired)
	ep := func(season, n int64) works.ItemID { return itemOf(t, tree, keyOf(7101*100000+season*1000+n)).ID }
	items := []works.ItemID{ep(1, 1), ep(0, 1), ep(1, 2), ep(2, 1), ep(0, 2), ep(1, 3), ep(2, 2)}
	want := []indexers.Group{
		{Key: sqlLevelKey(t, sh, 7101*100+1), Label: "season 1", Items: []works.ItemID{ep(1, 1), ep(1, 2), ep(1, 3)}},
		{Key: indexers.ItemGroup(ep(0, 1)), Label: "S00E01", Items: []works.ItemID{ep(0, 1)}},
		{Key: sqlLevelKey(t, sh, 7101*100+2), Label: "season 2", Items: []works.ItemID{ep(2, 1), ep(2, 2)}},
		{Key: indexers.ItemGroup(ep(0, 2)), Label: "S00E02", Items: []works.ItemID{ep(0, 2)}},
	}
	assertGroups(t, groupsOf(t, sh, tree.ID, items...), want)

	dvd := aired
	dvd.Order = tvdb.OrderDVD
	dvd.Episodes = slices.Clone(aired.Episodes)
	dvd.Episodes[0].Number, dvd.Episodes[1].Number = 2, 1
	sh.refresh(t, tree.ID, dvd)
	assertGroups(t, groupsOf(t, sh, tree.ID, items...), want)
}

func TestAbsoluteOrderGroupsByAiredSeasonSoTheSwitchChangesKeys(t *testing.T) {
	sh := open(t)
	aired := newShow(7801, "Long Saga", 1, 2, 2)
	aired.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderAbsolute}
	tree := sh.add(t, aired)
	ep := func(season, n int64) works.ItemID { return itemOf(t, tree, keyOf(7801*100000+season*1000+n)).ID }
	items := []works.ItemID{ep(1, 1), ep(1, 2), ep(2, 1), ep(2, 2), ep(0, 1)}
	before := groupsOf(t, sh, tree.ID, items...)

	abs := absoluteOf(aired, db.SeasonID(7801*100+50))
	unaired := episode{ID: db.EpisodeID(7801*100000 + 3*1000 + 1), Season: 1, Number: 5, Absolute: ptr(int64(5)), AirDate: "2022-04-01"}
	abs.Episodes = append(abs.Episodes, unaired)
	tree = sh.refresh(t, tree.ID, abs)
	extra := itemOf(t, tree, keyOf(int64(unaired.ID))).ID
	after := groupsOf(t, sh, tree.ID, append(items, extra)...)
	assertGroups(t, after, []indexers.Group{
		{Key: indexers.NamedGroup("aired-1"), Label: "season 1", Items: []works.ItemID{ep(1, 1), ep(1, 2)}},
		{Key: indexers.NamedGroup("aired-2"), Label: "season 2", Items: []works.ItemID{ep(2, 1), ep(2, 2)}},
		{Key: indexers.ItemGroup(ep(0, 1)), Label: "S00E01", Items: []works.ItemID{ep(0, 1)}},
		{Key: sqlLevelKey(t, sh, 7801*100+50), Label: "season 1", Items: []works.ItemID{extra}},
	})
	for i := range 2 {
		if before[i].Key == after[i].Key {
			t.Errorf("%s keeps key %s across the switch to Absolute order, want a new key", before[i].Label, before[i].Key)
		}
	}
}

func TestASeriesNamedByAirDateSearchesEachEpisodeAlone(t *testing.T) {
	sh := open(t)
	tree := sh.add(t, newShow(8101, "Night Desk", 0, 2))
	sh.setOptions(t, tree.ID, airDateOn(), false)
	ep := func(n int64) works.ItemID { return itemOf(t, tree, keyOf(8101*100000+1000+n)).ID }
	assertGroups(t, groupsOf(t, sh, tree.ID, ep(1), ep(2)), []indexers.Group{
		{Key: indexers.ItemGroup(ep(1)), Label: "S01E01", Items: []works.ItemID{ep(1)}},
		{Key: indexers.ItemGroup(ep(2)), Label: "S01E02", Items: []works.ItemID{ep(2)}},
	})
}

func TestGroupTermsAskForTheSeasonWithNoEpisodeOrALoneEpisodesOwnTerms(t *testing.T) {
	sh := open(t)
	aired := newShow(7201, "Lantern Keepers", 0, 2, 3)
	aired.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderAbsolute}
	tree := sh.add(t, aired)
	ep := func(season, n int64) works.ItemID { return itemOf(t, tree, keyOf(7201*100000+season*1000+n)).ID }

	season := groupsOf(t, sh, tree.ID, ep(2, 1), ep(2, 3))[0]
	terms := groupTerms(t, sh, tree.ID, season)
	if terms.Mode != "tvsearch" || terms.Claim.IDs["tvdbid"] != "7201" || terms.Query != "Lantern Keepers" ||
		terms.Season == nil || *terms.Season != 2 || terms.Episode != "" {
		t.Fatalf("season 2's terms are %+v (season %v), want tvsearch by tvdbid 7201 for season 2 with no episode", terms, terms.Season)
	}
	if got := covers(sh, terms, "Lantern.Keepers.S02.1080p.WEB-DL"); !slices.Equal(got, []works.ItemID{ep(2, 1), ep(2, 2), ep(2, 3)}) {
		t.Errorf("season 2's terms cover %v with the season pack, want its three episodes", got)
	}

	lone := groupsOf(t, sh, tree.ID, ep(2, 2))[0]
	own := searchTerms(t, sh, ep(2, 2))
	if got := groupTerms(t, sh, tree.ID, lone); got.Mode != own.Mode || got.Query != own.Query || *got.Season != *own.Season || got.Episode != own.Episode || got.Episode != "2" {
		t.Fatalf("a lone episode's group terms are %+v, want its own SearchTerms %+v", got, own)
	}

	tree = sh.refresh(t, tree.ID, absoluteOf(aired, db.SeasonID(7201*100+50)))
	abs := groupsOf(t, sh, tree.ID, ep(2, 1), ep(2, 2))[0]
	if got := groupTerms(t, sh, tree.ID, abs); got.Season == nil || *got.Season != 2 || got.Episode != "" {
		t.Fatalf("in Absolute order Aired season 2's terms search season %v ep %q, want season 2 with no episode", got.Season, got.Episode)
	}
}
