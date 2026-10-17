package music_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/contenttypes/music"
	"github.com/jatassi/slipstream/fakes/contenttypes/music/musicmeta"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

const (
	lanterns   = "a1000000-0000-4000-8000-000000000001"
	marrowBay  = "a1000000-0000-4000-8000-000000000002"
	missing    = "a1000000-0000-4000-8000-000000000003"
	slow       = "a1000000-0000-4000-8000-000000000004"
	prolific   = "a1000000-0000-4000-8000-000000000005"
	unknownArt = "ffffffff-0000-4000-8000-000000000000"
)

func catalog(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("music")
	if err != nil {
		t.Fatal(err)
	}
	one := func(n int) []scenario.MusicRevision {
		return []scenario.MusicRevision{{Name: "One Track", SortName: "One Track", Albums: []scenario.MusicAlbum{{
			ID: fmt.Sprintf("b3000000-0000-4000-8000-%012d", n), Release: fmt.Sprintf("c3000000-0000-4000-8000-%012d", n), Title: "Only", Kind: "Album",
			Tracks: []scenario.MusicTrack{{Recording: "d3000000-0000-4000-8000-000000000001", Title: "Only", Disc: 1}},
		}}}}
	}
	s.Music.Artists = append(s.Music.Artists,
		scenario.MusicArtist{ID: missing, Revisions: one(1)},
		scenario.MusicArtist{ID: slow, Revisions: one(2)},
		scenario.MusicArtist{ID: prolific, Revisions: []scenario.MusicRevision{prolificRevision()}},
	)
	s.Faults = append(s.Faults,
		fault.Rule{ID: "missing", Target: "music-meta", Route: "artist", Params: map[string]string{"mbid": missing}, Fault: fault.Status{Code: 404}, Times: 1},
		fault.Rule{ID: "slow", Target: "music-meta", Route: "artist", Params: map[string]string{"mbid": slow}, Fault: fault.Hang{}, Times: 1},
	)
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	return s
}

func prolificRevision() scenario.MusicRevision {
	rev := scenario.MusicRevision{Name: "Prolific", SortName: "Prolific"}
	for i := 1; i <= 150; i++ {
		rev.Albums = append(rev.Albums, scenario.MusicAlbum{
			ID: fmt.Sprintf("b4000000-0000-4000-8000-%012d", i), Release: fmt.Sprintf("c4000000-0000-4000-8000-%012d", i),
			Title: fmt.Sprintf("Album %03d", i), Kind: "Album", FirstRelease: "2000",
			Tracks: []scenario.MusicTrack{{Recording: "d4000000-0000-4000-8000-000000000000", Title: "Single Track", Disc: 1}},
		})
	}
	last := &rev.Albums[149]
	last.Tracks = nil
	for i := 1; i <= 150; i++ {
		last.Tracks = append(last.Tracks, scenario.MusicTrack{Recording: fmt.Sprintf("d5000000-0000-4000-8000-%012d", i), Title: fmt.Sprintf("Track %03d", i), Length: time.Minute, Disc: 1 + i/101})
	}
	return rev
}

func open(f faketest.Served) contenttype.Type {
	return music.Def.Open(contenttype.Deps{
		HTTP:      f.Client,
		UserAgent: "SlipStream/test ( github.com/jatassi/SlipStream )",
		Log:       slog.New(slog.DiscardHandler),
	}, &music.Endpoints{Metadata: endpoint.MustParseURL(f.Address.URL("/ws/2"))})
}

type tier struct {
	name  string
	serve func(t *testing.T, test func(t *testing.T, f faketest.Served))
}

var tiers = []tier{
	{"loopback", func(t *testing.T, test func(*testing.T, faketest.Served)) {
		test(t, faketest.Loopback(t, musicmeta.Fake, catalog(t)))
	}},
	{"in memory", func(t *testing.T, test func(*testing.T, faketest.Served)) {
		synctest.Test(t, func(t *testing.T) { test(t, faketest.InMemory(t, musicmeta.Fake, catalog(t))) })
	}},
}

func inEachTier(t *testing.T, test func(t *testing.T, f faketest.Served)) {
	for _, tr := range tiers {
		t.Run(tr.name, func(t *testing.T) { tr.serve(t, test) })
	}
}

func lanternsRevision(second bool) works.Fetched {
	track2 := works.Entry{Key: "b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000002", Label: "Paper Boats"}
	track3 := works.Entry{Key: "b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000003", Label: "Low Bridge"}
	detail := `{"country":"GB","disambiguation":"","albums":[` +
		`{"key":"b1000000-0000-4000-8000-000000000001","released":"2001-03-12","primaryType":"Album","release":"c1000000-0000-4000-8000-000000000001","tracks":[` +
		`{"key":"b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000001","disc":1,"number":1,"lengthMs":205000},` +
		`{"key":"b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000002","disc":1,"number":2,"lengthMs":242000},` +
		`{"key":"b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000003","disc":1,"number":3,"lengthMs":178000}]},`
	if second {
		track2.Label = "Paper Ships"
		track3 = works.Entry{Key: "b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000006", Label: "Lantern Song"}
		detail = `{"country":"GB","disambiguation":"","albums":[` +
			`{"key":"b1000000-0000-4000-8000-000000000001","released":"2001-03-12","primaryType":"Album","release":"c1000000-0000-4000-8000-000000000001","tracks":[` +
			`{"key":"b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000001","disc":1,"number":1,"lengthMs":205000},` +
			`{"key":"b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000002","disc":1,"number":2,"lengthMs":242000},` +
			`{"key":"b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000006","disc":1,"number":3,"lengthMs":301000}]},`
	}
	return works.Fetched{
		Label:     "The Lanterns",
		SortLabel: "Lanterns, The",
		Children: []works.Entry{
			{Key: "b1000000-0000-4000-8000-000000000001", Label: "Harbor Lights", Children: []works.Entry{
				{Key: "b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000001", Label: "Tidewater"},
				track2,
				track3,
			}},
			{Key: "b1000000-0000-4000-8000-000000000002", Label: "Night Ferry", Children: []works.Entry{
				{Key: "b1000000-0000-4000-8000-000000000002/d1000000-0000-4000-8000-000000000004", Label: "Night Ferry"},
				{Key: "b1000000-0000-4000-8000-000000000002/d1000000-0000-4000-8000-000000000005", Label: "Signal Fires"},
			}},
		},
		Detail: []byte(detail +
			`{"key":"b1000000-0000-4000-8000-000000000002","released":"2003","primaryType":"EP","release":"c1000000-0000-4000-8000-000000000002","tracks":[` +
			`{"key":"b1000000-0000-4000-8000-000000000002/d1000000-0000-4000-8000-000000000004","disc":1,"number":1,"lengthMs":190000},` +
			`{"key":"b1000000-0000-4000-8000-000000000002/d1000000-0000-4000-8000-000000000005","disc":1,"number":2,"lengthMs":224000}]}]}`),
	}
}

func TestFetchServesEachRevisionInTurn(t *testing.T) {
	inEachTier(t, func(t *testing.T, f faketest.Served) {
		m := open(f)
		for i, want := range []works.Fetched{lanternsRevision(false), lanternsRevision(true), lanternsRevision(true)} {
			got, err := m.Fetch(t.Context(), lanterns)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("fetch %d = %+v\ndetail %s\nwant %+v\ndetail %s", i, got, got.Detail, want, want.Detail)
			}
		}
	})
}

func TestSearchScoresNamesAndSortNames(t *testing.T) {
	inEachTier(t, func(t *testing.T, f faketest.Served) {
		m := open(f)
		for query, want := range map[string][]works.Candidate{
			"lanterns": {{Key: lanterns, Label: "The Lanterns", Fields: []works.FieldValue{works.TextField("country", "GB")}}},
			"o":        {{Key: missing, Label: "One Track"}, {Key: slow, Label: "One Track"}, {Key: marrowBay, Label: "Marrow Bay", Fields: []works.FieldValue{works.TextField("country", "NZ")}}, {Key: prolific, Label: "Prolific"}},
			"nobody":   {},
		} {
			got, err := m.Search(t.Context(), query)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Search(%q) = %+v, want %+v", query, got, want)
			}
		}
	})
}

func TestFaultsMapOntoTheSourceErrors(t *testing.T) {
	inEachTier(t, func(t *testing.T, f faketest.Served) {
		m := open(f)
		for _, c := range []struct {
			key  works.Key
			want string
			is   error
		}{
			{marrowBay, "GET /artist/a1000000-0000-4000-8000-000000000002: 503 Service Unavailable: the metadata source didn't answer", works.ErrUnavailable},
			{missing, "GET /artist/a1000000-0000-4000-8000-000000000003: 404 Not Found: the metadata source has no record with that key", works.ErrNotFound},
			{unknownArt, "GET /artist/ffffffff-0000-4000-8000-000000000000: 404 Not Found: the metadata source has no record with that key", works.ErrNotFound},
		} {
			_, err := m.Fetch(t.Context(), c.key)
			if err == nil || err.Error() != c.want || !errors.Is(err, c.is) {
				t.Errorf("Fetch(%s) = %v, want %s wrapping %v", c.key, err, c.want, c.is)
			}
		}
		for _, key := range []works.Key{marrowBay, missing} {
			got, err := m.Fetch(t.Context(), key)
			if err != nil || got.Label == "" {
				t.Errorf("Fetch(%s) once its fault is spent = %+v, %v", key, got, err)
			}
		}
	})
}

func TestASlowSourceIsUnavailableAtTheFetchTiming(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := open(faketest.InMemory(t, musicmeta.Fake, catalog(t)))
		fetch := works.DefaultTimings().Fetch
		ctx, cancel := context.WithTimeout(t.Context(), fetch)
		defer cancel()
		start := time.Now()
		_, err := m.Fetch(ctx, slow)
		want := "GET /artist/a1000000-0000-4000-8000-000000000004: context deadline exceeded: the metadata source didn't answer"
		if err == nil || err.Error() != want || !errors.Is(err, works.ErrUnavailable) {
			t.Fatalf("Fetch(slow) = %v, want %s", err, want)
		}
		if took := time.Since(start); took != fetch {
			t.Fatalf("Fetch(slow) gave up after %s, want %s", took, fetch)
		}
	})
}

func TestFetchPagesReleaseGroupsAndNumbersTracksAcrossDiscs(t *testing.T) {
	inEachTier(t, func(t *testing.T, f faketest.Served) {
		got, err := open(f).Fetch(t.Context(), prolific)
		if err != nil {
			t.Fatal(err)
		}
		last := got.Children[len(got.Children)-1]
		summary := fmt.Sprintf("%d albums, %s to %s; %s has %d tracks: %s, %s, %s, %s",
			len(got.Children), got.Children[0].Label, last.Label, last.Label, len(last.Children),
			last.Children[0].Label, last.Children[99].Label, last.Children[100].Label, last.Children[149].Label)
		if want := "150 albums, Album 001 to Album 150; Album 150 has 150 tracks: Track 001, Track 100, Track 101, Track 150"; summary != want {
			t.Fatalf("Fetch(prolific) = %s, want %s", summary, want)
		}
	})
}

func TestAnAlbumDatedOffTheCalendarHasNoReleaseDate(t *testing.T) {
	const calendar = "a1000000-0000-4000-8000-000000000006"
	s := catalog(t)
	s.Music.Artists = append(s.Music.Artists, scenario.MusicArtist{ID: calendar, Revisions: []scenario.MusicRevision{{Name: "Calendar", SortName: "Calendar", Albums: []scenario.MusicAlbum{{
		ID: "b6000000-0000-4000-8000-000000000001", Release: "c6000000-0000-4000-8000-000000000001", Title: "Thirteenth", Kind: "Album", FirstRelease: "2001-13",
		Tracks: []scenario.MusicTrack{{Recording: "d6000000-0000-4000-8000-000000000001", Title: "Only", Disc: 1}},
	}}}}})
	got, err := open(faketest.Loopback(t, musicmeta.Fake, s)).Fetch(t.Context(), calendar)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"country":"","disambiguation":"","albums":[{"key":"b6000000-0000-4000-8000-000000000001","released":"","primaryType":"Album","release":"c6000000-0000-4000-8000-000000000001",` +
		`"tracks":[{"key":"b6000000-0000-4000-8000-000000000001/d6000000-0000-4000-8000-000000000001","disc":1,"number":1,"lengthMs":null}]}]}`
	if string(got.Detail) != want {
		t.Fatalf("Fetch(calendar) detail = %s, want %s", got.Detail, want)
	}
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

func TestStoreThenFields(t *testing.T) {
	lib := storagetest.Library(t, node.LibrarySchemas(contenttype.NewSet(music.Def))...)
	m := open(faketest.Loopback(t, musicmeta.Fake, catalog(t)))
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{music.Def.ID(): m}, Timings: works.DefaultTimings(),
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	tree, _, err := w.Add(t.Context(), identity.Admin{}, music.Def.ID(), lanterns, works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	at := works.Placed{Work: tree.ID, Levels: map[works.Key]works.LevelID{}, Items: map[works.Key]works.ItemID{}}
	for _, l := range tree.Levels {
		at.Levels[l.Key] = l.ID
	}
	for _, it := range tree.Items {
		at.Items[it.Key] = it.ID
	}
	level := func(key works.Key) works.LevelID { return at.Levels[key] }
	item := func(key works.Key) works.ItemID { return at.Items[key] }
	track := func(ms int) []works.FieldValue {
		return []works.FieldValue{works.NumberField("disc", 1), works.DurationField("length", time.Duration(ms)*time.Millisecond)}
	}
	want := works.FieldSet{
		Work: []works.FieldValue{works.TextField("country", "GB")},
		Levels: map[works.LevelID][]works.FieldValue{
			level("b1000000-0000-4000-8000-000000000001"): {works.TextField("kind", "Album"), works.DateField("released", works.Date{Year: 2001, Month: 3, Day: 12})},
			level("b1000000-0000-4000-8000-000000000002"): {works.TextField("kind", "EP"), works.DateField("released", works.Date{Year: 2003})},
		},
		Items: map[works.ItemID][]works.FieldValue{
			item("b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000001"): track(205000),
			item("b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000002"): track(242000),
			item("b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000003"): track(178000),
			item("b1000000-0000-4000-8000-000000000002/d1000000-0000-4000-8000-000000000004"): track(190000),
			item("b1000000-0000-4000-8000-000000000002/d1000000-0000-4000-8000-000000000005"): track(224000),
		},
	}
	fields := func(when string) {
		t.Helper()
		var got works.FieldSet
		if err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			got, err = m.Fields(ctx, r, tree.ID)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Fields %s = %+v, want %+v", when, got, want)
		}
	}
	fields("after the add")

	again, err := open(faketest.Loopback(t, musicmeta.Fake, catalog(t))).Fetch(t.Context(), lanterns)
	if err != nil {
		t.Fatal(err)
	}
	if err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return m.Store(ctx, tx, at, again.Detail)
	}); err != nil {
		t.Fatal(err)
	}
	fields("after a second Store")
}

func TestNamingTokensNameATrack(t *testing.T) {
	lib := storagetest.Library(t, node.LibrarySchemas(contenttype.NewSet(music.Def))...)
	m := open(faketest.Loopback(t, musicmeta.Fake, catalog(t)))
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{music.Def.ID(): m}, Timings: works.DefaultTimings(),
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	tree, _, err := w.Add(t.Context(), identity.Admin{}, music.Def.ID(), lanterns, works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	var tokens map[string]string
	if err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		tokens, err = m.NamingTokens(ctx, r, tree.Items[1].ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	templates := m.DefaultTemplates()
	folder, err := naming.Render(templates.Folder, tokens, naming.Core{Target: "FLAC"})
	if err != nil {
		t.Fatal(err)
	}
	file, err := naming.Render(templates.File, tokens, naming.Core{Target: "FLAC"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := folder+"/"+file, "The Lanterns/The Lanterns - 1-02 - Paper Boats FLAC"; got != want {
		t.Fatalf("named %q, want %q", got, want)
	}
}

func timings(t *testing.T, s scenario.Scenario, artist works.Key) map[works.Key]targets.Timing {
	t.Helper()
	lib := storagetest.Library(t, node.LibrarySchemas(contenttype.NewSet(music.Def))...)
	m := open(faketest.Loopback(t, musicmeta.Fake, s))
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{music.Def.ID(): m}, Timings: works.DefaultTimings(),
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	tree, _, err := w.Add(t.Context(), identity.Admin{}, music.Def.ID(), artist, works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	var byID map[works.ItemID]targets.Timing
	if err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		byID, err = m.Timings(ctx, r, []works.WorkID{tree.ID})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	out := map[works.Key]targets.Timing{}
	for _, it := range tree.Items {
		if timing, ok := byID[it.ID]; ok {
			out[it.Key] = timing
		}
	}
	if len(out) != len(byID) {
		t.Fatalf("Timings answered %d Items, %d of them in the Work", len(byID), len(out))
	}
	return out
}

func TestTimingsAreEachTracksLengthAndItsAlbumsDate(t *testing.T) {
	harbor := release.Known(works.Date{Year: 2001, Month: 3, Day: 12})
	ferry := release.Known(works.Date{Year: 2003})
	ms := func(n int) release.Fact[time.Duration] { return release.Known(time.Duration(n) * time.Millisecond) }
	want := map[works.Key]targets.Timing{
		"b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000001": {Released: harbor, Runtime: ms(205000)},
		"b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000002": {Released: harbor, Runtime: ms(242000)},
		"b1000000-0000-4000-8000-000000000001/d1000000-0000-4000-8000-000000000003": {Released: harbor, Runtime: ms(178000)},
		"b1000000-0000-4000-8000-000000000002/d1000000-0000-4000-8000-000000000004": {Released: ferry, Runtime: ms(190000)},
		"b1000000-0000-4000-8000-000000000002/d1000000-0000-4000-8000-000000000005": {Released: ferry, Runtime: ms(224000)},
	}
	if got := timings(t, catalog(t), lanterns); !reflect.DeepEqual(got, want) {
		t.Fatalf("Timings = %+v, want %+v", got, want)
	}
}

func TestAnUndatedAlbumAndAnUntimedTrackStayUnknown(t *testing.T) {
	const undated = "a1000000-0000-4000-8000-000000000007"
	s := catalog(t)
	s.Music.Artists = append(s.Music.Artists, scenario.MusicArtist{ID: undated, Revisions: []scenario.MusicRevision{{Name: "Undated", SortName: "Undated", Albums: []scenario.MusicAlbum{{
		ID: "b7000000-0000-4000-8000-000000000001", Release: "c7000000-0000-4000-8000-000000000001", Title: "Someday", Kind: "Album",
		Tracks: []scenario.MusicTrack{{Recording: "d7000000-0000-4000-8000-000000000001", Title: "Only", Disc: 1}},
	}}}}})
	want := map[works.Key]targets.Timing{"b7000000-0000-4000-8000-000000000001/d7000000-0000-4000-8000-000000000001": {}}
	if got := timings(t, s, undated); !reflect.DeepEqual(got, want) {
		t.Fatalf("Timings = %+v, want one Item with neither a date nor a length", got)
	}
}

func TestDeluxeNamesItsCut(t *testing.T) {
	cuts := append(release.BuiltIn(), open(faketest.Loopback(t, musicmeta.Fake, catalog(t))).Cuts()...)
	for title, want := range map[string]release.CutKey{
		"The Lanterns - Harbor Lights (Deluxe Edition) 2001 FLAC": "deluxe",
		"The Lanterns - Harbor Lights (Deluxe) 2001 FLAC":         "deluxe",
		"The Lanterns - Harbor Lights 2001 FLAC":                  release.Standard,
	} {
		if got := release.Title(title, cuts).Cut; got != want {
			t.Errorf("cut of %q = %s, want %s", title, got, want)
		}
	}
}

func TestTheLibraryAuditPassesWithMusic(t *testing.T) {
	storagetest.AuditLibrary(t, node.LibrarySchemas(contenttype.NewSet(music.Def)))
}
