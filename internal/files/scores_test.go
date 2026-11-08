package files_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// titled is a slipstream format named after word that matches a release
// name holding it as a word.
func titled(word string) formats.Format {
	value, err := json.Marshal(map[string]string{"value": `\b` + word + `\b`})
	if err != nil {
		panic(err)
	}
	return formats.Format{
		ID: targets.NewFormatID(), Name: word, Dialect: formats.SlipStream,
		Conditions: []formats.Condition{{Name: word, Kind: formats.ReleaseTitle, Fields: value}},
	}
}

// grabbed imports a file of tree's item at q whose Grab's Release was
// named name.
func (l *library) grabbed(t *testing.T, svc *files.Files, tree works.Tree, item int, q release.Quality, name string) {
	t.Helper()
	holding := files.HoldingOf(storage.NewUUID(), ".mkv")
	l.put(t, holding, name)
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, _, err := svc.Import(ctx, tx, files.Incoming{
			WorkID: tree.ID, ItemID: tree.Items[item].ID, Cut: release.Standard, RootID: l.root.ID, Holding: holding,
			Size: int64(len(name)), Probe: files.Probe{Width: 1920, Height: 1080}, Quality: q,
			ReleaseName: name, Title: release.Title(name, l.fx, nil),
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// scores is each filled cell of work's score as files stores it, and as
// a refill from scratch gives it.
func (l *library) scores(t *testing.T, svc *files.Files, work works.WorkID) (stored, scratch map[files.CellKey]int) {
	t.Helper()
	stored = map[files.CellKey]int{}
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		cells, err := svc.Cells(ctx, r, work)
		if err != nil {
			return err
		}
		for _, c := range cells {
			if c.File != nil {
				stored[files.CellKey{Item: c.Item.ID, Edition: c.Edition, Target: c.Target.ID}] = c.Score
			}
		}
		scratch, err = svc.ScoresFromScratch(ctx, r, work)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return stored, scratch
}

func TestARefillFromScratchGivesEveryStoredFillScore(t *testing.T) {
	r := rand.New(rand.NewPCG(23, 5)) //nolint:gosec // seeded on purpose so failures replay
	for run := range 4 {
		s := newScored(t)
		for step := range 40 {
			did := s.step(t, r)
			if stored, scratch := s.scores(t, s.svc, s.tree.ID); !maps.Equal(stored, scratch) {
				t.Fatalf("run %d, step %d, after it %s: stored fill scores %v, a refill from scratch gives %v", run, step, did, stored, scratch)
			}
		}
	}
}

// scored is a Work wanting from two Targets of a library with a format
// for each of words.
type scored struct {
	*library
	svc     *files.Files
	tree    works.Tree
	specs   []targets.Spec
	targets []targets.Target
	formats []formats.Format
}

var words = []string{"HDR", "Atmos", "GRP", "HYBRID"}

func newScored(t *testing.T) *scored {
	t.Helper()
	s := &scored{library: open(t), specs: []targets.Spec{
		{Scores: targets.Scores{Increment: 1}, Name: "Disc", Tiers: [][]release.Quality{{release.Bluray1080p}, {release.WEBDL1080p, release.WEBRip1080p}}, Cutoff: release.Bluray1080p},
		{Scores: targets.Scores{Increment: 1}, Name: "Web", Tiers: [][]release.Quality{{release.WEBDL1080p, release.WEBRip1080p}}, Cutoff: release.WEBDL1080p},
	}}
	for _, spec := range s.specs {
		s.targets = append(s.targets, s.target(t, spec.Name, spec.Tiers...))
	}
	profile := s.profile(t, "Scored", s.targets...)
	s.tree, s.svc = s.work(t, "w1", true, &profile), s.files(files.NoGrabs{})
	for _, w := range words {
		s.formats = append(s.formats, titled(w))
	}
	s.change(t, s.svc, targets.AddFormats{Dialect: formats.SlipStream, Add: s.formats}, files.Mode{Unwanted: files.Keep})
	return s
}

// step imports a file of a random name, scores a Target at random, or
// points a format at another word, and says which it did.
func (s *scored) step(t *testing.T, r *rand.Rand) string {
	t.Helper()
	switch r.IntN(3) {
	case 0:
		name := marking(r, "Scarf.1080p", append(slices.Clone(words), "PROPER"))
		q := []release.Quality{release.WEBDL1080p, release.WEBRip1080p, release.Bluray1080p}[r.IntN(3)]
		s.grabbed(t, s.svc, s.tree, r.IntN(len(s.tree.Items)), q, name)
		return "imported " + name + " as " + q.String()
	case 1:
		i := r.IntN(len(s.specs))
		spec := &s.specs[i]
		spec.Scores.Formats = map[formats.ID]int{}
		for _, f := range s.formats {
			if v := r.IntN(21) - 10; v != 0 && r.IntN(2) == 0 {
				spec.Scores.Formats[f.ID] = v
			}
		}
		spec.Scores.UpgradeUntil, spec.Scores.Increment = r.IntN(10), 1+r.IntN(3)
		spec.Propers = []targets.Propers{targets.PreferAndUpgrade, targets.PreferNoUpgrade, targets.DontPrefer}[r.IntN(3)]
		s.change(t, s.svc, targets.UpdateTarget{ID: s.targets[i].ID, Spec: *spec}, files.Mode{Unwanted: files.Keep})
		return fmt.Sprintf("scored %s %v", spec.Name, spec.Scores.Formats)
	}
	f := s.formats[r.IntN(len(s.formats))]
	edited := titled(words[r.IntN(len(words))])
	edited.ID, edited.Name = f.ID, f.Name
	s.change(t, s.svc, targets.UpdateFormat{ID: f.ID, Format: edited}, files.Mode{Unwanted: files.Keep})
	return "pointed format " + f.Name + " at " + edited.Conditions[0].Name
}

// marking is name with each of words added at random.
func marking(r *rand.Rand, name string, words []string) string {
	parts := []string{name}
	for _, w := range words {
		if r.IntN(2) == 0 {
			parts = append(parts, w)
		}
	}
	return strings.Join(parts, ".")
}

func TestAHeldFileAtTheCutoffIsMetOnlyAtOrAboveUpgradeUntil(t *testing.T) {
	l := open(t)
	hdr, flawed := titled("HDR"), titled("Flawed")
	svc := l.files(files.NoGrabs{})
	web := l.target(t, "Web", []release.Quality{release.WEBDL1080p, release.WEBRip1080p})
	profile := l.profile(t, "Scored", web)
	tree := l.work(t, "w1", true, &profile)
	l.change(t, svc, targets.AddFormats{Dialect: formats.SlipStream, Add: []formats.Format{hdr, flawed}}, files.Mode{Unwanted: files.Keep})
	spec := targets.Spec{
		Name: "Web", Tiers: [][]release.Quality{{release.WEBDL1080p, release.WEBRip1080p}}, Cutoff: release.WEBDL1080p,
		Scores: targets.Scores{Formats: map[formats.ID]int{hdr.ID: 10, flawed.ID: -5}, UpgradeUntil: 10, Increment: 1},
	}
	l.change(t, svc, targets.UpdateTarget{ID: web.ID, Spec: spec}, files.Mode{Unwanted: files.Keep})
	l.grabbed(t, svc, tree, 0, release.WEBDL1080p, "Scarf.1080p.WEB-DL")
	l.grabbed(t, svc, tree, 1, release.WEBDL1080p, "Mittens.1080p.WEB-DL.HDR")
	want := []string{"Scarf.1080p.WEB-DL at 0 upgradable wanted=true", "Mittens.1080p.WEB-DL.HDR at 10 met wanted=false"}
	if got := held(t, l, svc, tree.ID); !slices.Equal(got, want) {
		t.Fatalf("cells at upgrade-until 10 = %v, want %v", got, want)
	}

	l.grabbed(t, svc, tree, 0, release.WEBRip1080p, "Scarf.1080p.WEBRip.HDR")
	want[0] = "Scarf.1080p.WEBRip.HDR at 10 met wanted=false"
	if got := held(t, l, svc, tree.ID); !slices.Equal(got, want) {
		t.Fatalf("cells once a scored file of the cutoff's tier arrives = %v, want it filling Scarf's: %v", got, want)
	}

	spec.Scores.UpgradeUntil = 0
	l.change(t, svc, targets.UpdateTarget{ID: web.ID, Spec: spec}, files.Mode{Unwanted: files.Keep})
	l.grabbed(t, svc, tree, 1, release.WEBDL1080p, "Mittens.1080p.WEB-DL.HDR.Flawed")
	want = []string{"Scarf.1080p.WEBRip.HDR at 10 met wanted=false", "Mittens.1080p.WEB-DL.HDR at 10 met wanted=false"}
	if got := held(t, l, svc, tree.ID); !slices.Equal(got, want) {
		t.Fatalf("cells at upgrade-until 0 = %v, want %v", got, want)
	}
	spec.Scores.Formats[hdr.ID] = 1
	l.change(t, svc, targets.UpdateTarget{ID: web.ID, Spec: spec}, files.Mode{Unwanted: files.Keep})
	want = []string{"Scarf.1080p.WEBRip.HDR at 1 met wanted=false", "Mittens.1080p.WEB-DL.HDR at 1 met wanted=false"}
	if got := held(t, l, svc, tree.ID); !slices.Equal(got, want) {
		t.Fatalf("cells once HDR scores 1 = %v, want the HDR file keeping Mittens' cell over the one scoring -4: %v", got, want)
	}
	other := l.work(t, "w2", true, &profile)
	l.grabbed(t, svc, other, 0, release.WEBDL1080p, "Scarf.1080p.WEB-DL.Flawed")
	want = []string{"Scarf.1080p.WEB-DL.Flawed at -5 upgradable wanted=true", "none at 0 missing wanted=true"}
	if got := held(t, l, svc, other.ID); !slices.Equal(got, want) {
		t.Fatalf("cells of a file scoring below upgrade-until 0 = %v, want it still a Want: %v", got, want)
	}
}

// held is each cell of work as its file's release name, its score and its
// state.
func held(t *testing.T, l *library, svc *files.Files, work works.WorkID) []string {
	t.Helper()
	var out []string
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		cells, err := svc.Cells(ctx, r, work)
		for _, c := range cells {
			name := "none"
			if c.File != nil {
				name = c.File.ReleaseName
			}
			out = append(out, fmt.Sprintf("%s at %d %s wanted=%t", name, c.Score, c.State, c.Wanted()))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
