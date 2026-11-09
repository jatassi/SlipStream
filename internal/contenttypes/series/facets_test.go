package series

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// severance is a full series record: two seasons of 2 episodes and one
// special.
func severance() record {
	r := newShow(371980, "Severance", 1, 2, 2)
	r.Show.TMDBID = ptr(db.TMDBID(95396))
	r.Show.IMDbID = "tt11280740"
	r.Show.OriginalTitle = "Severance"
	r.Show.OriginalLanguage = "eng"
	r.Show.Country = "usa"
	r.Show.Network = "Apple TV+"
	r.Show.Certification = &certification{Code: "TV-MA", Country: "US"}
	r.Show.Genres = []string{"Drama", "Mystery"}
	r.Show.Aliases = []string{"Separation", "Severance"}
	r.Show.Creators = []person{{Name: "Dan Erickson"}}
	r.Show.Cast = []person{{Name: "Adam Scott", Role: "Mark Scout", Photo: "/adam.jpg"}}
	r.Show.Trailer = &trailer{Site: "YouTube", Key: "xEQP4VVuyrY"}
	r.Show.NextAired = "2026-12-01"
	return r
}

func TestFieldsShowTheRecord(t *testing.T) {
	sh := open(t)
	tree := sh.add(t, severance())
	texts := map[string]string{
		"years": "2022 to present", "network": "Apple TV+", "certification": "TV-MA", "certification_country": "US",
		"status": "Continuing", "english_title": "Severance", "episode_order": "Aired", "tvdb_id": "371980",
		"tmdb_id": "95396", "imdb_id": "tt11280740", "original_language": "eng", "country": "usa",
	}
	for key, want := range texts {
		if f, _ := fieldOf(tree.Fields, key); f.Text != want {
			t.Errorf("%s reads %q, want %q", key, f.Text, want)
		}
	}
	for key, want := range map[string]float64{"year": 2022, "seasons": 2, "episodes": 4} {
		if f, _ := fieldOf(tree.Fields, key); f.Number != want {
			t.Errorf("%s reads %v, want %v", key, f.Number, want)
		}
	}
	if f, _ := fieldOf(tree.Fields, "next_aired"); f.Date != (works.Date{Year: 2026, Month: 12, Day: 1}) {
		t.Errorf("next_aired reads %v", f.Date)
	}
	cast, _ := fieldOf(tree.Fields, "credits")
	if len(cast.People) != 1 || cast.People[0].Role != "Mark Scout" || cast.People[0].Photo.Path != "/adam.jpg" {
		t.Errorf("cast reads %+v", cast.People)
	}
	if f, _ := fieldOf(tree.Fields, "creators"); len(f.People) != 1 || f.People[0].Name != "Dan Erickson" {
		t.Errorf("creators read %+v", f.People)
	}
	ep := itemOf(t, tree, keyOf(371980*100000+2*1000+1))
	if f, _ := fieldOf(ep.Fields, "code"); f.Text != "S02E01" {
		t.Errorf("the first episode of season 2 reads code %q, want S02E01", f.Text)
	}
	if f, _ := fieldOf(ep.Fields, "air_date"); f.Date != (works.Date{Year: 2022, Month: 3, Day: 2}) {
		t.Errorf("its air date reads %v", f.Date)
	}
	special := itemOf(t, tree, keyOf(371980*100000+1))
	if f, _ := fieldOf(special.Fields, "code"); f.Text != "S00E01" {
		t.Errorf("the special reads code %q, want S00E01", f.Text)
	}
}

func TestYearsOfAnEndedSeriesSpanItsRun(t *testing.T) {
	sh := open(t)
	r := newShow(7401, "Short Run", 0, 2)
	r.Show.Status, r.Show.LastAired = tvdb.Ended, "2024-05-01"
	tree := sh.add(t, r)
	if f, _ := fieldOf(tree.Fields, "years"); f.Text != "2022 to 2024" {
		t.Fatalf("years read %q, want 2022 to 2024", f.Text)
	}
}

func TestListFieldsReadOnlyTheListedFields(t *testing.T) {
	sh := open(t)
	tree := sh.add(t, severance())
	want := map[works.WorkID][]works.FieldValue{tree.ID: {
		works.TextField("network", "Apple TV+"),
		works.TextField("certification", "TV-MA"),
		works.TextField("status", "Continuing"),
		works.NumberField("seasons", 2),
		works.NumberField("episodes", 4),
		works.NumberField("year", 2022),
		works.TextsField("genres", []string{"Drama", "Mystery"}),
	}}
	ids := []works.WorkID{tree.ID}
	for _, ws := range []works.Among[works.WorkID]{works.Only(ids), works.Every(ids)} {
		var got map[works.WorkID][]works.FieldValue
		sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			got, err = sh.s.ListFields(ctx, r, ws)
			return err
		})
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ListFields of every series (%t) = %+v, want %+v", ws.Every(), got, want)
		}
	}
}

func TestTitlesAreEveryTitleOnce(t *testing.T) {
	sh := open(t)
	tree := sh.add(t, severance())
	var got []string
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = sh.s.Titles(ctx, r, tree.ID)
		return err
	})
	if want := []string{"Severance", "Separation"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Titles = %q, want %q", got, want)
	}
}

func TestLinksOpenEachHostTheSeriesNames(t *testing.T) {
	sh := open(t)
	tree := sh.add(t, severance())
	want := []works.WebLink{
		{Key: "imdb", Label: "IMDb", URL: "https://www.imdb.com/title/tt11280740", Kind: works.LinkPage},
		{Key: "tvdb", Label: "TVDB", URL: "https://thetvdb.com/dereferrer/series/371980", Kind: works.LinkPage},
		{Key: "tmdb", Label: "TMDB", URL: "https://www.themoviedb.org/tv/95396", Kind: works.LinkPage},
		{Key: "trakt", Label: "Trakt", URL: "https://trakt.tv/search/tvdb/371980?id_type=show", Kind: works.LinkPage},
		{Key: "trailer", Label: "Trailer", URL: "https://www.youtube.com/watch?v=xEQP4VVuyrY", Kind: works.LinkTrailer},
	}
	if got := sh.s.Links(tree.Key, tree.Fields); !reflect.DeepEqual(got, want) {
		t.Fatalf("Links = %+v, want %+v", got, want)
	}
	if got := sh.s.Links(tree.Key, nil); !reflect.DeepEqual(got, []works.WebLink{want[1], want[3]}) {
		t.Fatalf("Links with no fields = %+v, want TVDB and Trakt", got)
	}
}

func TestCodeInAbsoluteOrder(t *testing.T) {
	sh := open(t)
	aired := newShow(7501, "Long Saga", 1, 2, 2)
	aired.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderAbsolute}
	tree := sh.add(t, aired)
	if f, _ := fieldOf(itemOf(t, tree, keyOf(7501*100000+2*1000+2)).Fields, "code"); f.Text != "S02E02" {
		t.Fatalf("in Aired order S02E02 reads %q", f.Text)
	}
	tree = sh.refresh(t, tree.ID, absoluteOf(aired, db.SeasonID(7501*100+50)))
	for key, want := range map[works.Key]string{
		keyOf(7501*100000 + 1*1000 + 1): "1", keyOf(7501*100000 + 2*1000 + 2): "4", keyOf(7501*100000 + 1): "S00E01",
	} {
		if f, _ := fieldOf(itemOf(t, tree, key).Fields, "code"); f.Text != want {
			t.Errorf("in Absolute order %s reads code %q, want %q", key, f.Text, want)
		}
	}
}

func TestOrderChoiceListsOnlyTVDBsOrders(t *testing.T) {
	sh := open(t)
	aired := newShow(7601, "Two Orders", 0, 3)
	aired.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderAbsolute}
	tree := sh.add(t, aired)
	choice := func() setting.ChoiceField {
		t.Helper()
		var fields []setting.Field
		sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			fields, err = sh.s.OptionFields(ctx, r, tree.ID)
			return err
		})
		if len(fields) != 2 {
			t.Fatalf("OptionFields = %+v, want the order and Name by air date", fields)
		}
		c, ok := fields[0].(setting.ChoiceField)
		if !ok {
			t.Fatalf("the order option is %T, want a choice", fields[0])
		}
		return c
	}
	c := choice()
	if c.Key != "order" || c.Label != "Episode order" || c.Value != "default" ||
		!reflect.DeepEqual(c.Options, []setting.ChoiceOption{{Value: "default", Label: "Aired"}, {Value: "absolute", Label: "Absolute"}}) {
		t.Fatalf("the order choice is %+v, want Aired and Absolute with Aired picked", c)
	}
	patch := func(raw string) ([]byte, error) {
		var out []byte
		var perr error
		sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			out, perr = sh.s.PatchOptions(ctx, r, tree.ID, setting.Patch{"order": json.RawMessage(raw)})
			return nil
		})
		return out, perr
	}
	if _, err := patch(`"dvd"`); !errors.Is(err, works.ErrOrderUnavailable) {
		t.Fatalf("patching to DVD order = %v, want ErrOrderUnavailable", err)
	}
	var refused *setting.RangeError
	if _, err := patch(`"sideways"`); !errors.As(err, &refused) {
		t.Fatalf("patching to an order no series has = %v, want a RangeError", err)
	}
	if got, err := patch(`"absolute"`); err != nil || string(got) != `{"order":"absolute"}` {
		t.Fatalf("patching to Absolute = %s, %v", got, err)
	}
	if got, err := patch(`"default"`); err != nil || got != nil {
		t.Fatalf("patching to Aired = %s, %v, want the default's nil", got, err)
	}

	tree = sh.refresh(t, tree.ID, absoluteOf(aired, db.SeasonID(7601*100+50)))
	if c := choice(); c.Value != "absolute" || c.IsDefault {
		t.Fatalf("after a fetch in Absolute order the choice is %+v", c)
	}
	var fetching []byte
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		fetching, err = sh.s.FetchOptions(ctx, r, tree.ID)
		return err
	})
	if string(fetching) != `{"order":"absolute"}` {
		t.Fatalf("FetchOptions = %s, want Absolute", fetching)
	}
}

func TestAirDelayMovesAvailability(t *testing.T) {
	sh := open(t)
	r := newShow(7701, "Night Shift", 0, 3)
	r.Episodes[1].Runtime = 62 * time.Minute
	r.Episodes[2].AirDate = ""
	tree := sh.add(t, r)
	timings := func() map[works.ItemID]targets.Timing {
		t.Helper()
		var got map[works.ItemID]targets.Timing
		sh.read(t, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
			var err error
			got, err = sh.s.Timings(ctx, rd, works.Only([]works.WorkID{tree.ID}))
			return err
		})
		return got
	}
	first, second, third := tree.Items[0].ID, tree.Items[1].ID, tree.Items[2].ID
	got := timings()
	want := map[works.ItemID]targets.Timing{
		first:  {Released: release.Known(works.Date{Year: 2022, Month: 3, Day: 2}), Runtime: release.Known(50 * time.Minute), Basis: "Air date"},
		second: {Released: release.Known(works.Date{Year: 2022, Month: 3, Day: 3}), Runtime: release.Known(62 * time.Minute), Basis: "Air date"},
		third:  {Runtime: release.Known(50 * time.Minute)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Timings = %+v, want %+v", got, want)
	}

	fields, err := sh.works.SetSettings(t.Context(), identity.Admin{}, Def.ID(), setting.Patch{"series.airDelay": json.RawMessage("172800")})
	if err != nil {
		t.Fatal(err)
	}
	var d setting.DurationField
	for _, f := range fields {
		if f, ok := f.(setting.DurationField); ok && f.Key == "series.airDelay" {
			d = f
		}
	}
	if d.Label != "Delay after an air date" || d.Value != 172800 || d.Step != 86400 || d.Max != 30*86400 || d.Default != 0 {
		t.Fatalf("the air delay field is %+v", d)
	}
	got = timings()
	if g := got[first]; g.Released != release.Known(works.Date{Year: 2022, Month: 3, Day: 4}) || g.Basis != "Air date plus 2 days" {
		t.Fatalf("with a 2-day delay the first episode's timing is %+v", g)
	}
	if _, known := got[third].Released.Get(); known {
		t.Fatalf("an episode with no air date counts as released")
	}
	var refused *setting.RangeError
	if _, err := sh.works.SetSettings(t.Context(), identity.Admin{}, Def.ID(), setting.Patch{"series.airDelay": json.RawMessage("2678400")}); !errors.As(err, &refused) {
		t.Fatalf("a 31-day delay = %v, want a RangeError", err)
	}
}

func TestGrowingIsContinuingAndUpcoming(t *testing.T) {
	sh := open(t)
	ids := map[tvdb.Status]works.WorkID{}
	for i, st := range []tvdb.Status{tvdb.Continuing, tvdb.Ended, tvdb.Upcoming} {
		r := newShow(int64(7800+i), string(st), 0, 1)
		r.Show.Status = st
		ids[st] = sh.add(t, r).ID
	}
	var got map[works.WorkID]bool
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = sh.s.Growing(ctx, r, []works.WorkID{ids[tvdb.Continuing], ids[tvdb.Ended], ids[tvdb.Upcoming]})
		return err
	})
	want := map[works.WorkID]bool{ids[tvdb.Continuing]: true, ids[tvdb.Ended]: false, ids[tvdb.Upcoming]: true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Growing = %v, want %v", got, want)
	}
}

func TestKeysOfMapsTMDBAliases(t *testing.T) {
	sh := open(t)
	sh.add(t, severance())
	other := newShow(7901, "No TMDB", 0, 1)
	sh.add(t, other)
	var got []works.Key
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = sh.s.KeysOf(ctx, r, []works.Key{"tmdb-95396", "tmdb-1", "7901", "tmdb-x"})
		return err
	})
	if want := []works.Key{"371980"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("KeysOf = %v, want %v", got, want)
	}
}
