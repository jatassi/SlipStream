package series

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/naming/namingtest"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestTheSeriesConformsToNaming(t *testing.T) {
	namingtest.Conform(t, open(t).s)
}

// named is where the default templates put a 1080p file of each of keys
// of tree's series as it is stored now.
func (sh shelf) named(t *testing.T, tree works.Tree, keys ...works.Key) map[works.Key]string {
	t.Helper()
	var values map[works.WorkID]naming.Values
	sh.read(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		values, err = sh.s.NamingValues(ctx, r, works.Only([]works.WorkID{tree.ID}))
		return err
	})
	out := map[works.Key]string{}
	for _, key := range keys {
		sub := values[tree.ID].Of(works.BareOf(tree), []works.ItemID{itemOf(t, tree, key).ID})
		out[key] = namingtest.Name(t, sh.s, sub, naming.FileFacts{Target: "1080p"})
	}
	return out
}

func TestEachEpisodeNamesWithItsSeriesVariant(t *testing.T) {
	sh := open(t)
	r := newShow(7901, "Late Night", 1, 2)
	r.Orders = []tvdb.Order{tvdb.OrderAired, tvdb.OrderAbsolute}
	r.Episodes[1].AirDate = ""
	tree := sh.add(t, r)
	first, undated, special := keyOf(7901*100000+1*1000+1), keyOf(7901*100000+1*1000+2), keyOf(7901*100000+1)
	check := func(when string, got, want map[works.Key]string) {
		t.Helper()
		for key, w := range want {
			if got[key] != w {
				t.Errorf("%s, %s is named %q, want %q", when, key, got[key], w)
			}
		}
	}
	check("in Aired order", sh.named(t, tree, first, undated, special), map[works.Key]string{
		first:   "Late Night (2022)/Season 01/Late Night - S01E01 - Episode 1 1080p.mkv",
		undated: "Late Night (2022)/Season 01/Late Night - S01E02 - Episode 2 1080p.mkv",
		special: "Late Night (2022)/Specials/Late Night - S00E01 - Episode 1 1080p.mkv",
	})

	sh.setOptions(t, tree.ID, airDateOn(), false)
	check("named by air date", sh.named(t, tree, first, undated), map[works.Key]string{
		first:   "Late Night (2022)/Season 01/Late Night - 2022-03-02 - Episode 1 1080p.mkv",
		undated: "Late Night (2022)/Season 01/Late Night - S01E02 - Episode 2 1080p.mkv",
	})

	sh.setOptions(t, tree.ID, setting.Patch{"nameByAirDate": json.RawMessage("false")}, false)
	tree = sh.refresh(t, tree.ID, absoluteOf(r, db.SeasonID(7901*100+50)))
	check("in Absolute order", sh.named(t, tree, first, undated, special), map[works.Key]string{
		first:   "Late Night (2022)/Season 01/Late Night - S01E01 - 001 - Episode 1 1080p.mkv",
		undated: "Late Night (2022)/Season 01/Late Night - S01E02 - 002 - Episode 2 1080p.mkv",
		special: "Late Night (2022)/Specials/Late Night - S00E01 - Episode 1 1080p.mkv",
	})
}
