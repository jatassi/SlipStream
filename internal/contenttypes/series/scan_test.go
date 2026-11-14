package series

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestHintNamesTheKeyATVDBTagGives(t *testing.T) {
	s := &Series{}
	tests := []struct {
		name string
		want scanning.Hint
	}{
		{"Severance (2022) {tvdb-371980}", scanning.Hint{Title: "Severance", Year: 2022, IDs: []scanning.ID{{Provider: scanning.TVDB, Value: "371980"}}, Key: "371980"}},
		{"Severance (2022) {tmdb-95396}", scanning.Hint{Title: "Severance", Year: 2022, IDs: []scanning.ID{{Provider: scanning.TMDB, Value: "95396"}}}},
		{"Trailer Park Boys (2001)", scanning.Hint{Title: "Trailer Park Boys", Year: 2001}},
	}
	for _, tt := range tests {
		if got := s.Hint(tt.name); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Hint(%q) = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestClaimsAreTheFirstAiredYearAndEveryID(t *testing.T) {
	sh := open(t)
	tree := sh.add(t, severance())
	bare := sh.add(t, newShow(7901, "No TMDB", 0, 1))
	want := map[works.WorkID]scanning.Claim{
		tree.ID: {Year: 2022, IDs: []scanning.ID{
			{Provider: scanning.TVDB, Value: "371980"}, {Provider: scanning.TMDB, Value: "95396"}, {Provider: scanning.IMDb, Value: "tt11280740"},
		}},
		bare.ID: {Year: 2022, IDs: []scanning.ID{{Provider: scanning.TVDB, Value: "7901"}}},
	}
	ids := []works.WorkID{tree.ID, bare.ID}
	for _, ws := range []works.Among[works.WorkID]{works.Only(ids), works.Every(ids)} {
		var got map[works.WorkID]scanning.Claim
		sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			got, err = sh.s.Claims(ctx, r, ws)
			return err
		})
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Claims of every series (%t) = %+v, want %+v", ws.Every(), got, want)
		}
	}
}

func TestYearOfASearchResult(t *testing.T) {
	c := candidateOf(tvdb.Found{ID: 371980, Name: "Severance", Year: 2022})
	if y := (&Series{}).YearOf(c); y != 2022 {
		t.Fatalf("YearOf(%+v) = %d, want 2022", c, y)
	}
	if y := (&Series{}).YearOf(candidateOf(tvdb.Found{ID: 1, Name: "Unaired"})); y != 0 {
		t.Fatalf("YearOf a result with no year = %d, want 0", y)
	}
}

func placer(t *testing.T, sh shelf, work works.WorkID) scanning.Placer {
	t.Helper()
	var place scanning.Placer
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		place, err = sh.s.Place(ctx, r, work)
		return err
	})
	return place
}

// placed is where place puts rel: the key of its Item, or its reason.
func placed(t *testing.T, tree works.Tree, place scanning.Placer, rel string) any {
	t.Helper()
	p := place(rel)
	items, ok := p.Items()
	if !ok {
		return p.Reason()
	}
	var keys []works.Key
	for _, it := range tree.Items {
		if slices.Contains(items, it.ID) {
			keys = append(keys, it.Key)
		}
	}
	if len(keys) == 1 {
		return keys[0]
	}
	if len(keys) > 1 {
		return keys
	}
	t.Fatalf("Place(%q) = Items %v, none of %s's", rel, items, tree.Label)
	return nil
}

func TestPlaceReadsEpisodeMarkers(t *testing.T) {
	sh := open(t)
	tree := sh.add(t, newShow(7001, "Severed", 1, 2, 2))
	place := placer(t, sh, tree.ID)
	ep := func(season, number int64) works.Key { return keyOf(7001*100000 + season*1000 + number) }
	tests := []struct {
		rel  string
		want any
	}{
		{"Season 1/Severed - S01E02 - Half Loop.mkv", ep(1, 2)},
		{"Severed.s02e01.1080p.WEB-DL.mkv", ep(2, 1)},
		{"Severed 2x02.mkv", ep(2, 2)},
		{"Specials/Severed S00E01.mkv", ep(0, 1)},
		{"Severed 1920x1080 S01E01.mkv", ep(1, 1)},
		{"Severed S01E01-720p.mkv", ep(1, 1)},
		{"Severed S01E01E02.mkv", []works.Key{ep(1, 1), ep(1, 2)}},
		{"Severed S01E01-E02.mkv", []works.Key{ep(1, 1), ep(1, 2)}},
		{"Severed S01E01-02.mkv", []works.Key{ep(1, 1), ep(1, 2)}},
		{"Severed 1x01-1x02.mkv", []works.Key{ep(1, 1), ep(1, 2)}},
		{"Severed.S01.1080p.WEB-DL.mkv", scanning.Reason{Code: scanning.NoNumber}},
		{"Severed - Pilot.mkv", scanning.Reason{Code: scanning.NoNumber}},
		{"Season 1/Severed.mkv", scanning.Reason{Code: scanning.NoNumber}},
		{"Severed S03E14.mkv", scanning.Reason{Code: scanning.NoSuchItem, Names: "S03E14"}},
	}
	for _, tt := range tests {
		if got := placed(t, tree, place, tt.rel); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Place(%q) = %v, want %v", tt.rel, got, tt.want)
		}
	}
}

func TestPlaceReadsTheSeriesOrder(t *testing.T) {
	sh := open(t)

	aired := newShow(7601, "Swapped", 0, 2)
	aired.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderDVD}
	tree := sh.add(t, aired)
	dvd := aired
	dvd.Order = tvdb.OrderDVD
	dvd.Episodes = slices.Clone(aired.Episodes)
	dvd.Episodes[0].Number, dvd.Episodes[1].Number = 2, 1
	tree = sh.refresh(t, tree.ID, dvd)
	if got := placed(t, tree, placer(t, sh, tree.ID), "Swapped S01E01.mkv"); got != keyOf(7601*100000+1000+2) {
		t.Errorf("in DVD order S01E01 places %v, want the episode the DVD numbers first", got)
	}

	long := newShow(7801, "Long Saga", 0, 2, 2)
	long.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderAbsolute}
	saga := sh.add(t, long)
	saga = sh.refresh(t, saga.ID, absoluteOf(long, db.SeasonID(7801*100+50)))
	place := placer(t, sh, saga.ID)
	if got := placed(t, saga, place, "Long Saga S02E01.mkv"); got != keyOf(7801*100000+2*1000+1) {
		t.Errorf("in Absolute order S02E01 places %v, want the episode Aired order numbers S02E01", got)
	}
	if got := placed(t, saga, place, "Long Saga S01E03.mkv"); got != (scanning.Reason{Code: scanning.NoSuchItem, Names: "S01E03"}) {
		t.Errorf("in Absolute order S01E03 places %v, want no such episode: releases name Aired numbers", got)
	}
}

func TestPlacePrefersALiveEpisodeAndKeepsWithdrawnOnes(t *testing.T) {
	sh := open(t)
	r := newShow(7701, "Recut", 0, 3)
	tree := sh.add(t, r)
	recut := r
	recut.Episodes = []episode{r.Episodes[0], {
		ID: db.EpisodeID(7701*100000 + 1000 + 4), Season: 1, Number: 2, AiredSeason: ptr(int64(1)), AiredNumber: ptr(int64(2)), AirDate: "2022-03-09",
	}}
	tree = sh.refresh(t, tree.ID, recut)
	place := placer(t, sh, tree.ID)
	if got := placed(t, tree, place, "Recut S01E02.mkv"); got != keyOf(7701*100000+1000+4) {
		t.Errorf("S01E02 places %v, want the live episode over the withdrawn one", got)
	}
	if got := placed(t, tree, place, "Recut S01E03.mkv"); got != keyOf(7701*100000+1000+3) {
		t.Errorf("S01E03 places %v, want the withdrawn episode no live one replaced", got)
	}
}
