package series

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

func ptr[T any](v T) *T { return &v }

// catalog is the TVDB the tests' Works are fetched from: one record per
// series in each order it lists, answered in the order the Fetching's
// options ask, else in the order last put, as a refetch in a new order
// answers.
type catalog struct {
	mu      sync.Mutex
	records map[works.Key]map[tvdb.Order]record
	latest  map[works.Key]tvdb.Order
}

func (c *catalog) put(rs ...record) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range rs {
		key := keyOf(int64(r.Show.TVDBID))
		if c.records[key] == nil {
			c.records[key] = map[tvdb.Order]record{}
		}
		c.records[key][r.Order] = r
		c.latest[key] = r.Order
	}
}

func (*catalog) Search(context.Context, works.TypeID, string) ([]works.Candidate, error) {
	return nil, nil
}

func (c *catalog) Fetch(_ context.Context, _ works.TypeID, key works.Key, f works.Fetching) (works.Fetched, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	o := options{Order: c.latest[key]}
	if f.Options != nil {
		if err := json.Unmarshal(f.Options, &o); err != nil {
			return works.Fetched{}, fmt.Errorf("decode options %s: %w", f.Options, err)
		}
	}
	r, ok := c.records[key][o.Order]
	if !ok {
		return works.Fetched{}, works.ErrNotFound
	}
	return fetchedOf(r)
}

// fetchedOf is r as the series' Source answers it: each season a level
// in number order with the specials last and aside, each episode an Item
// under its season.
func fetchedOf(r record) (works.Fetched, error) {
	detail, err := encode(r)
	if err != nil {
		return works.Fetched{}, err
	}
	f := works.Fetched{Label: r.Show.EnglishTitle, Detail: detail}
	var specials *works.Entry
	for _, s := range r.Seasons {
		e := works.Entry{Key: keyOf(int64(s.ID)), Label: "Season " + strconv.FormatInt(s.Number, 10), Aside: s.Number == 0}
		for _, ep := range r.Episodes {
			if ep.Season == s.Number {
				e.Children = append(e.Children, works.Entry{Key: keyOf(int64(ep.ID)), Label: "Episode " + strconv.FormatInt(ep.Number, 10)})
			}
		}
		if e.Aside {
			specials = &e
			continue
		}
		f.Children = append(f.Children, e)
	}
	if specials != nil {
		f.Children = append(f.Children, *specials)
	}
	return f, nil
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type noPrefetch struct{}

func (noPrefetch) Prefetch([]artwork.Want) {}

type published struct{ n atomic.Int64 }

func (p *published) Publish(context.Context, works.Changed) { p.n.Add(1) }

type shelf struct {
	works *works.Works
	s     contenttype.Library
	lib   *storage.DB[storage.Library]
	src   *catalog
	sent  *published
}

func open(t *testing.T) shelf {
	t.Helper()
	sh := shelf{
		s:   Def.OpenLibrary(contenttype.LibraryDeps{Region: metadata.Region, Log: slog.New(slog.DiscardHandler)}, Def.Endpoints()),
		lib: storagetest.Library(t, node.LibrarySchemas(contenttype.NewSet(Def))...), src: &catalog{records: map[works.Key]map[tvdb.Order]record{}, latest: map[works.Key]tvdb.Order{}},
		sent: &published{},
	}
	var err error
	sh.works, err = works.Open(t.Context(), works.Config{
		Library: sh.lib, Catalogs: map[works.TypeID]works.Catalog{Def.ID(): sh.s}, Timings: works.DefaultTimings(),
		Source: sh.src, Artwork: noPrefetch{}, Changed: sh.sent, TypesChanged: discard[works.ContentTypesChanged]{},
		TypeChanged: discard[works.TypeChanged]{}, Removed: discard[works.Removed]{}, Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	return sh
}

func (sh shelf) add(t *testing.T, rs ...record) works.Tree {
	t.Helper()
	sh.src.put(rs...)
	tree, _, err := sh.works.Add(t.Context(), identity.Admin{}, Def.ID(), keyOf(int64(rs[0].Show.TVDBID)), works.Adding{Monitor: true})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func (sh shelf) refresh(t *testing.T, id works.WorkID, rs ...record) works.Tree {
	t.Helper()
	sh.src.put(rs...)
	tree, err := sh.works.Refresh(t.Context(), identity.Admin{}, id)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func (sh shelf) read(t *testing.T, f func(ctx context.Context, r storage.Reader[storage.Library]) error) {
	t.Helper()
	if err := sh.lib.Read(t.Context(), f); err != nil {
		t.Fatal(err)
	}
}

// newShow builds a series in Aired order, with its seasons, numbered from
// 1, of episodes counts each, and specials of specials episodes. Season n
// has TVDB id id*100+n, and episode e of season n id id*100000+n*1000+e.
func newShow(id int64, title string, specials int, episodes ...int) record {
	r := record{
		Order: tvdb.OrderAired, Orders: []tvdb.Order{tvdb.OrderAired},
		Show: show{
			TVDBID: db.TVDBID(id), EnglishTitle: title, Status: tvdb.Continuing, FirstAired: "2022-02-18",
			AverageRuntime: 50 * time.Minute, Overview: title + " is a series.",
		},
	}
	add := func(n int64, count int) {
		r.Seasons = append(r.Seasons, season{ID: db.SeasonID(id*100 + n), Number: n, Overview: "Season " + strconv.FormatInt(n, 10) + " of " + title + "."})
		for e := int64(1); e <= int64(count); e++ {
			r.Episodes = append(r.Episodes, episode{
				ID: db.EpisodeID(id*100000 + n*1000 + e), Season: n, Number: e, AiredSeason: ptr(n), AiredNumber: ptr(e),
				AirDate: "2022-03-" + pad(e%28+1), Overview: "Episode " + strconv.FormatInt(e, 10) + ".",
			})
		}
	}
	for i, count := range episodes {
		add(int64(i+1), count)
	}
	if specials > 0 {
		add(0, specials)
	}
	return r
}

// absoluteOf is r in Absolute order: its main seasons' episodes in one
// season of TVDB id seasonID, numbered on from 1, and its specials as
// they are.
func absoluteOf(r record, seasonID db.SeasonID) record {
	a := r
	a.Order = tvdb.OrderAbsolute
	a.Seasons = []season{{ID: seasonID, Number: 1}}
	a.Episodes = nil
	n := int64(0)
	for _, e := range r.Episodes {
		if e.Season > 0 {
			n++
			e.Season, e.Number, e.Absolute = 1, n, ptr(n)
		}
		a.Episodes = append(a.Episodes, e)
	}
	for _, s := range r.Seasons {
		if s.Number == 0 {
			a.Seasons = append(a.Seasons, s)
		}
	}
	return a
}

func pad(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}

func itemOf(t *testing.T, tree works.Tree, key works.Key) works.Item {
	t.Helper()
	for _, it := range tree.Items {
		if it.Key == key {
			return it
		}
	}
	t.Fatalf("no Item %s in %s", key, tree.Label)
	return works.Item{}
}

func fieldOf(fs []works.FieldValue, key string) (works.FieldValue, bool) {
	for _, f := range fs {
		if f.Key == key {
			return f, true
		}
	}
	return works.FieldValue{}, false
}
