package grabs

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// seasoned reads names as episodic does, with any SxxEyy as its mark.
type seasoned struct{ episodic }

var episodeMark = regexp.MustCompile(`^S\d\dE\d\d$`)

func (s seasoned) ReadName(p release.Prepared) release.Name {
	n := s.episodic.ReadName(p)
	for _, w := range strings.Split(p.Title.String(), ".") {
		if episodeMark.MatchString(w) {
			n.Marks = []release.Mark{{Words: w, Key: w}}
		}
	}
	return n
}

// stored is a Manual search's rows as the store hands them over, the
// title reading still JSON, and the view's other inputs.
type stored struct {
	in   manualIn
	rows []db.GrabsRelease
	read [][]byte
}

// storedSeasons is a Manual search of seasons seasons of ten Items each,
// every run answered by six Indexers listing a hundred Releases each, of
// which a quarter are 720p and a quarter TS.
func storedSeasons(b *testing.B, seasons int) (*Grabs, stored) {
	b.Helper()
	gr := &Grabs{cfg: Config{Sealer: secrettest.Sealer(b)}}
	covers := map[string]works.ItemID{}
	j := judging{
		expect: indexers.Expect{
			Claim:  release.Claim{Titles: []string{"The Lantern Keepers", "Lantern Keepers"}},
			Covers: func(n release.Name) []works.ItemID { return []works.ItemID{covers[marks(n)]} },
		},
		names: seasoned{}, cuts: release.BuiltIn(), scoring: noFormats,
	}
	s := stored{in: manualIn{
		tree: works.Tree{Summary: works.Summary{Label: "The Lantern Keepers"}}, heard: map[SearchID][]heardRelease{},
		nouns: works.Description{Item: works.Noun{One: "Episode", Many: "Episodes"}, Items: works.ItemsMany}, judging: j,
		blocks: blocks{}, routed: map[indexers.ID]string{}, standings: indexers.Standings{}, now: time.Now(),
		line: func(*indexers.Outcome) string { return "Found 100 Releases" },
	}}
	var sites []indexers.ID
	for k := range 6 {
		id := indexers.ID{UUID: storage.NewUUID()}
		sites = append(sites, id)
		s.in.standings[id] = indexers.Standing{Indexer: id, Automatic: k < 5}
	}
	for season := range seasons {
		level := works.LevelID{UUID: storage.NewUUID()}
		s.in.tree.Levels = append(s.in.tree.Levels, works.Level{ID: level, Label: fmt.Sprintf("Season %d", season+1)})
		for episode := range 10 {
			mark := fmt.Sprintf("S%02dE%02d", season+1, episode+1)
			covers[mark] = s.searched(b, gr, level, mark, sites)
		}
	}
	s.readAll(b, j)
	return gr, s
}

// searched adds to s an Item of level, with its two missing cells and a
// finished run of it, each of sites listing a hundred Releases marked
// mark.
func (s *stored) searched(b *testing.B, gr *Grabs, level works.LevelID, mark string, sites []indexers.ID) works.ItemID {
	b.Helper()
	it := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}, LevelID: level, Monitored: true}
	s.in.tree.Items = append(s.in.tree.Items, it)
	for _, t := range []targets.Target{hd, small} {
		s.in.cells = append(s.in.cells, files.Cell{Item: it, Edition: targets.AnyEdition, Target: t, State: files.Missing, Released: true})
	}
	run := Search{ID: SearchID{UUID: storage.NewUUID()}, ItemID: it.ID, Finished: time.Now()}
	qualities := []string{"1080p.BluRay", "1080p.WEB-DL", "720p.WEB-DL", "1080p.TS"}
	for k, site := range sites {
		name := fmt.Sprintf("Site %d", k)
		run.Indexers = append(run.Indexers, SearchIndexer{Indexer: site, Name: name, Outcome: &indexers.Outcome{Kind: indexers.Found, Releases: 100}})
		for i := range 100 {
			at := k*100 + i
			h := bencode.InfoHash([]byte(fmt.Sprintf("%-20s", fmt.Sprintf("%s%d", mark, at))))
			r := indexers.Release{Indexer: site, Priority: 10 * k, Release: searching.Release{
				Title: fmt.Sprintf("Lantern.Keepers.%s.%s.x264-G%d", mark, qualities[i%4], at), Size: int64(1+i) << 28, Seeders: i,
				InfoHash: &h, Link: searching.LinkOf(fmt.Sprintf("https://site.test/dl/%s/%d", mark, at)),
			}}
			kept, err := encodeKept(r)
			if err != nil {
				b.Fatal(err)
			}
			s.rows = append(s.rows, db.GrabsRelease{
				SearchID: run.ID, Position: int64(at), Title: r.Title, IndexerID: site, IndexerName: name,
				Size: r.Size, Seeders: int64(r.Seeders), Outcome: ReleaseListed, Formats: "[]", Release: &kept,
				LinkSealed: gr.cfg.Sealer.Seal([]byte(r.Offered().Reveal()), keptLinkAD(run.ID, at)),
			})
		}
	}
	s.in.runs = append(s.in.runs, run)
	return it.ID
}

// readAll reads every row's title as a run stores it, on four workers
// since each title takes about a millisecond.
func (s *stored) readAll(b *testing.B, j judging) {
	b.Helper()
	s.read = make([][]byte, len(s.rows))
	var wg sync.WaitGroup
	for w := range 4 {
		wg.Go(func() {
			for i := w; i < len(s.rows); i += 4 {
				var err error
				if s.read[i], err = json.Marshal(release.Title(s.rows[i].Title, j.names, j.cuts)); err != nil {
					b.Error(err)
				}
			}
		})
	}
	wg.Wait()
}

// view decodes s's rows as manualInput does and works out the view.
func (gr *Grabs) view(b *testing.B, s stored) ManualView {
	kept := map[SearchID]int{}
	byRun := map[SearchID][]int{}
	for i, row := range s.rows {
		kept[row.SearchID]++
		byRun[row.SearchID] = append(byRun[row.SearchID], i)
	}
	in := s.in
	var err error
	in.heard, err = gr.heardIn(s.in.manual.ID, kept, func(run SearchID) ([]db.GrabsRelease, error) {
		rows := make([]db.GrabsRelease, len(byRun[run]))
		for k, i := range byRun[run] {
			rows[k] = s.rows[i]
			rows[k].Read = &db.Reading{}
			if err := rows[k].Read.Scan(string(s.read[i])); err != nil {
				return nil, err
			}
		}
		return rows, nil
	})
	if err != nil {
		b.Fatal(err)
	}
	v, err := manualView(in, nil)
	if err != nil {
		b.Fatal(err)
	}
	return v
}

// BenchmarkManualView is the view a GET works out, from the rows a store
// hands over: a season of 10 Items and a Work of 20 seasons, six Indexers
// listing 100 Releases each per Item. Cold decodes every run, as the
// first read does. Warm decodes none, as a read does when no Indexer has
// answered since the last. The design's budgets are 150 ms and 1.5 s.
func BenchmarkManualView(b *testing.B) {
	for _, c := range []struct {
		name    string
		seasons int
	}{{"season", 1}, {"twenty seasons", 20}} {
		b.Run(c.name, func(b *testing.B) {
			gr, s := storedSeasons(b, c.seasons)
			b.Run("cold", func(b *testing.B) { gr.viewEach(b, s, c.seasons, true) })
			b.Run("warm", func(b *testing.B) { gr.viewEach(b, s, c.seasons, false) })
		})
	}
}

// viewEach works out the view of s, seasons seasons, each loop, decoding
// every run first when cold is set.
func (gr *Grabs) viewEach(b *testing.B, s stored, seasons int, cold bool) {
	gr.view(b, s)
	for b.Loop() {
		if cold {
			gr.decoded = decodedRuns{}
		}
		v := gr.view(b, s)
		if len(v.Plan.Rows) != 20*seasons || v.Releases != 6000*seasons {
			b.Fatalf("the view plans %d rows over %d Releases, want %d over %d", len(v.Plan.Rows), v.Releases, 20*seasons, 6000*seasons)
		}
	}
}
