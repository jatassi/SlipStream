package files_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var judged = []release.Quality{
	release.HDTV720p, release.WEBRip720p, release.WEBDL720p, release.Bluray720p,
	release.HDTV1080p, release.WEBRip1080p, release.WEBDL1080p, release.Bluray1080p,
}

// probeOf is what probing a file of q reads.
func probeOf(q release.Quality) media.Record {
	if q.Info().Resolution == release.R720p {
		return media.Record{Video: media.Video{Width: 1280, Height: 720}}
	}
	return media.Record{Video: media.Video{Width: 1920, Height: 1080}}
}

// shelf adds a Work of levels, each of the Items it lists, wanting from
// profile.
func (l *library) shelf(t testing.TB, key works.Key, profile targets.ProfileID, levels ...[]string) works.Tree {
	t.Helper()
	fetched := works.Fetched{Label: "Shelf " + string(key)}
	for i, things := range levels {
		fetched.Children = append(fetched.Children, box(works.Key(fmt.Sprintf("b%d", i+1)), false, things...))
	}
	l.fx.Put(key, contenttypetest.Record{Tree: fetched})
	tree, _, err := l.works.Add(t.Context(), admin, contenttypetest.ID, key, works.Adding{
		Monitor: true,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, &profile); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

// randomCandidate is a file of one Item of tree, or of it and the next, at q.
func randomCandidate(r *rand.Rand, tree works.Tree, q release.Quality) files.Candidate {
	at := r.IntN(len(tree.Items))
	items := []works.ItemID{tree.Items[at].ID}
	if at+1 < len(tree.Items) && r.IntN(3) == 0 {
		items = append(items, tree.Items[at+1].ID)
	}
	return files.Candidate{Items: items, Cut: release.Standard, Holding: files.HoldingOf(storage.NewUUID(), ".mkv"), Size: 1024, Media: probeOf(q), Quality: q}
}

// importOne lands c of import in, refused or not, in a Write of its own.
func (l *library) importOne(t *testing.T, svc *files.Files, b files.Batch, c files.Candidate) (files.Landed, error) {
	t.Helper()
	b.Candidates = []files.Candidate{c}
	var landed files.Landed
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		landed, err = svc.Import(ctx, tx, files.Landing{Batch: b, PlacedBy: "copy"})
		return err
	})
	if refused := (*files.Refused)(nil); err != nil && !errors.As(err, &refused) {
		t.Fatal(err)
	}
	return landed, err
}

// TestJudgeThenLandAgrees judges random batches over random Works, then
// lands each candidate alone in a random order, all but those Judge
// refuses a name, and checks the landings against the verdicts: a
// candidate Judge says fills a cell is never refused, one that only meets
// cells live files hold always is, and once all have landed each cell
// holds the candidate Judge said fills it.
func TestJudgeThenLandAgrees(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 8)) //nolint:gosec // seeded on purpose so failures replay
	for trial := range 24 {
		t.Run(strconv.Itoa(trial), func(t *testing.T) {
			l := open(t)
			tree := l.shelf(t, "w1", l.profile(t, "Random", randomTargets(t, l, r)...), []string{"Scarf", "Mittens", "Gloves"}, []string{"Hat", "Coat"})
			svc := l.files(files.NoGrabs{})
			live := files.Batch{Work: tree.ID, Root: l.root.ID}
			for range r.IntN(4) {
				_, _ = l.importOne(t, svc, live, randomCandidate(r, tree, judged[r.IntN(len(judged))]))
			}
			b := randomBatch(r, tree, l.cellsOf(t, svc, tree.ID))
			b.Root = l.root.ID
			verdicts := l.judgeBatch(t, svc, b)
			want := predicted(verdicts)
			of := map[files.ID]int{}
			for _, i := range r.Perm(len(b.Candidates)) {
				if verdicts[i].Refusal == nil {
					l.landAsJudged(t, svc, b, i, verdicts[i], of)
				}
			}
			if got := heldBy(l.cellsOf(t, svc, tree.ID), of); fmt.Sprint(got) != fmt.Sprint(want) {
				t.Fatalf("after every landing the candidates fill %v, want Judge's %v", got, want)
			}
		})
	}
}

// randomTargets is one or two Targets of a random run of judged, each
// quality its own tier, so that no two of them tie.
func randomTargets(t *testing.T, l *library, r *rand.Rand) []targets.Target {
	var ts []targets.Target
	for i := range 1 + r.IntN(2) {
		var tiers [][]release.Quality
		for _, q := range judged[:2+r.IntN(len(judged)-1)] {
			tiers = append(tiers, []release.Quality{q})
		}
		r.Shuffle(len(tiers), func(i, j int) { tiers[i], tiers[j] = tiers[j], tiers[i] })
		ts = append(ts, l.target(t, fmt.Sprintf("Target %d", i), tiers...))
	}
	return ts
}

// randomBatch is one to four candidates of tree, of one Import, each of
// its own quality, forcing a random few of cells, none forced twice.
func randomBatch(r *rand.Rand, tree works.Tree, cells []files.Cell) files.Batch {
	b := files.Batch{Work: tree.ID, Import: storage.NewUUID()}
	forced := map[files.CellKey]bool{}
	for i, q := range r.Perm(len(judged))[:1+r.IntN(4)] {
		c := randomCandidate(r, tree, judged[q])
		c.Key = i
		for _, cell := range cells {
			key := files.CellKey{Item: cell.Item.ID, Edition: cell.Edition, Target: cell.Target.ID}
			if slices.Contains(c.Items, key.Item) && !forced[key] && r.IntN(4) == 0 {
				forced[key] = true
				c.Force = append(c.Force, key)
			}
		}
		b.Candidates = append(b.Candidates, c)
	}
	return b
}

// judgeBatch is Judge's verdicts on b.
func (l *library) judgeBatch(t testing.TB, svc *files.Files, b files.Batch) []files.Verdict {
	t.Helper()
	var verdicts []files.Verdict
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		verdicts, err = svc.Judge(ctx, r, b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return verdicts
}

// predicted is each cell verdicts say a candidate fills, to its index.
func predicted(verdicts []files.Verdict) map[files.CellKey]int {
	out := map[files.CellKey]int{}
	for i, v := range verdicts {
		for _, key := range v.Fills {
			out[key] = i
		}
	}
	return out
}

// landAsJudged lands b's candidate i alone, checks its landing against v,
// Judge's verdict on it, and records the file it landed as in of.
func (l *library) landAsJudged(t *testing.T, svc *files.Files, b files.Batch, i int, v files.Verdict, of map[files.ID]int) {
	t.Helper()
	landed, err := l.importOne(t, svc, b, b.Candidates[i])
	switch {
	case err != nil && len(v.Fills) > 0:
		t.Fatalf("candidate %d is refused landing, %v, yet Judge said it fills %v", i, err, v.Fills)
	case err == nil && len(v.Fills) == 0 && heldByLive(v):
		t.Fatalf("candidate %d landed filling %v, yet Judge said live files hold every cell it meets", i, landed.Fills)
	case err == nil:
		of[landed.File.ID] = i
	}
}

// heldBy is the cells of cells whose file is one of of's, to its value.
func heldBy(cells []files.Cell, of map[files.ID]int) map[files.CellKey]int {
	out := map[files.CellKey]int{}
	for _, cell := range cells {
		if cell.File == nil {
			continue
		}
		if i, ok := of[cell.File.ID]; ok {
			out[files.CellKey{Item: cell.Item.ID, Edition: cell.Edition, Target: cell.Target.ID}] = i
		}
	}
	return out
}

// cellsOf is work's cells as svc computes them now.
func (l *library) cellsOf(t *testing.T, svc *files.Files, work works.WorkID) []files.Cell {
	t.Helper()
	var cells []files.Cell
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		cells, err = svc.Cells(ctx, r, work)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return cells
}

// heldByLive reports whether live files hold every cell v meets.
func heldByLive(v files.Verdict) bool {
	for _, h := range v.Lost {
		if h != files.HeldByLive {
			return false
		}
	}
	return true
}

// BenchmarkJudgeSeason measures Judge answering a season pack: 24
// candidates, one per Item of a season, in a Work of two seasons.
func BenchmarkJudgeSeason(b *testing.B) {
	l := open(b)
	hd := l.target(b, "HD", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	var season, next []string
	for e := 1; e <= 24; e++ {
		season, next = append(season, fmt.Sprintf("Episode %d", e)), append(next, fmt.Sprintf("Episode %d again", e))
	}
	tree := l.shelf(b, "season", l.profile(b, "Seasons", hd), season, next)
	batch := files.Batch{Work: tree.ID, Root: l.root.ID, Import: storage.NewUUID()}
	for i := range 24 {
		batch.Candidates = append(batch.Candidates, holdingOf(tree, i, files.HoldingOf(storage.NewUUID(), ".mkv"), 1024, probeOf(release.WEBDL1080p), release.WEBDL1080p))
	}
	svc := l.files(files.NoGrabs{})
	var took time.Duration
	for b.Loop() {
		start := time.Now()
		err := l.lib.Read(b.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			v, err := svc.Judge(ctx, r, batch)
			if err == nil && len(v[23].Fills) != 1 {
				b.Fatalf("the last episode fills %v, want its one cell", v[23].Fills)
			}
			return err
		})
		if err != nil {
			b.Fatal(err)
		}
		took = max(took, time.Since(start))
	}
	if took > 50*time.Millisecond {
		b.Fatalf("Judge took %s at worst, want under 50ms", took)
	}
}
