package storing

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/media"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// stamp sets the file at path's modified time.
func stamp(t *testing.T, path string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveIfRemovesOnlyTheFileAsked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		at := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC).Add(250*time.Millisecond + 400*time.Microsecond)
		for _, c := range []struct {
			name     string
			size     int64
			modified time.Time
			want     Removal
			stays    bool
		}{
			{"the same to the millisecond", 4, at.Truncate(time.Millisecond), Removed, false},
			{"another size", 5, at, Changed, true},
			{"a millisecond later", 4, at.Add(time.Millisecond), Changed, true},
		} {
			p := newPlacing(t, false)
			path := write(t, filepath.Join(p.folder, "Heat (1995)", "Heat.mkv"), "film")
			stamp(t, path, at)
			got, err := p.remote.RemoveIf(t.Context(), p.at, mustRel(t, "Heat (1995)/Heat.mkv"), c.size, c.modified)
			if err != nil || got != c.want {
				t.Errorf("%s: RemoveIf = %s, %v; want %s", c.name, got, err, c.want)
			}
			if _, err := os.Stat(path); (err == nil) != c.stays {
				t.Errorf("%s: the file is there: %v, want %v", c.name, err == nil, c.stays)
			}
		}
		p := newPlacing(t, false)
		if got, err := p.remote.RemoveIf(t.Context(), p.at, mustRel(t, "Heat (1995)/Heat.mkv"), 4, at); err != nil || got != Gone {
			t.Errorf("RemoveIf of no file = %s, %v; want gone", got, err)
		}
	})
}

func TestRemoveIfWaitsOutAFileAnotherProgramHoldsOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		at := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		for _, c := range []struct {
			name  string
			held  int
			want  Removal
			stays bool
		}{
			{"for a few tries", 4, Removed, false},
			{"past the bound", 1000, 0, true},
		} {
			p := newPlacing(t, false)
			path := write(t, filepath.Join(p.folder, "Heat (1995)", "Heat.mkv"), "film")
			stamp(t, path, at)
			p.disk.held[path] = c.held
			got, err := p.remote.RemoveIf(t.Context(), p.at, mustRel(t, "Heat (1995)/Heat.mkv"), 4, at)
			if got != c.want || (err != nil) != c.stays {
				t.Errorf("%s: RemoveIf = %s, %v; want %s", c.name, got, err, c.want)
			}
			if _, err := os.Stat(path); (err == nil) != c.stays {
				t.Errorf("%s: the file is there: %v, want %v", c.name, err == nil, c.stays)
			}
		}
	})
}

func TestOpenLibraryServesWhileTheNodeDrainsStorage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPlacing(t, true)
		path := write(t, filepath.Join(p.folder, "Heat (1995)", "Heat.mkv"), "film")
		got, err := p.store.OpenLibrary(t.Context(), p.at.Root, mustRel(t, "Heat (1995)/Heat.mkv"))
		if err != nil || got != path {
			t.Errorf("OpenLibrary while draining = %q, %v; want %q", got, err, path)
		}
		_, err = p.store.OpenLibrary(t.Context(), p.at.Root, mustRel(t, "Heat (1995)/Heat.nfo"))
		if a, ok := errors.AsType[*source.Absent](err); !ok || a.Reason != source.Gone {
			t.Errorf("OpenLibrary of a file that isn't there = %v, want Absent Gone", err)
		}
	})
}

func TestProbeMoveKeepsOnlyCompanionsInTheFilesFolder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		write(t, filepath.Join(r.folder, "Heat (1995)", "Heat.mkv"), "film")
		write(t, filepath.Join(r.folder, "Heat (1995)", "Heat.en.srt"), "ours")
		write(t, filepath.Join(r.folder, "Heat (1995)", "Heat.nfo"), "nfo")
		write(t, filepath.Join(r.folder, "Heat (1995)", "Extras", "Heat.de.srt"), "elsewhere")
		write(t, filepath.Join(r.folder, "Heat.fr.srt"), "above")
		sha := sha256.Sum256([]byte("ours"))
		if err := putLedger(t.Context(), r.local, r.at.Root, sha, mustRel(t, "Heat (1995)/Heat.en.srt")); err != nil {
			t.Fatal(err)
		}
		film, gone := Library{r.at.Root, mustRel(t, "Heat (1995)/Heat.mkv")}, Library{r.at.Root, mustRel(t, "Ronin (1998)/Ronin.mkv")}
		probed, err := r.remote.ProbeMove(t.Context(), node, ProbeMove{
			Sources: []ProbeRoot{{Root: r.at.Root, Rels: []Rel{film.Rel, gone.Rel}}}, Videos: media.DefaultVideos,
		})
		if err != nil {
			t.Fatal(err)
		}
		if f := probed.Files[film]; f.Size != 4 || f.Seen.Modified.IsZero() {
			t.Errorf("the film probed as %+v, want 4 bytes and its modified time", f)
		}
		if !slices.Equal(probed.Missing, []Library{gone}) {
			t.Errorf("missing = %v, want %v", probed.Missing, gone)
		}
		var got []string
		for _, c := range probed.Companions[film] {
			got = append(got, c.Rel.String()+map[bool]string{true: " ledger"}[c.Ledger])
		}
		slices.Sort(got)
		if want := []string{"Heat (1995)/Heat.en.srt ledger", "Heat (1995)/Heat.nfo"}; !slices.Equal(got, want) {
			t.Errorf("companions = %q, want %q", got, want)
		}
	})
}

func TestProbeMoveReadsTheFilesAtADestinationPath(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPlacing(t, false)
		write(t, filepath.Join(p.folder, "Heat (1995)", "HEAT.mkv"), "another")
		at := Library{p.at.Root, mustRel(t, "Heat (1995)/Heat.mkv")}
		probed, err := p.remote.ProbeMove(t.Context(), node, ProbeMove{Destinations: []ProbeRoot{{Root: p.at.Root, Rels: []Rel{at.Rel, mustRel(t, HoldingDir+"/u.0.mkv")}}}})
		if err != nil {
			t.Fatal(err)
		}
		if o := probed.Occupants[at]; len(o) != 1 || o[0].Rel.String() != "Heat (1995)/HEAT.mkv" || o[0].Size != 7 {
			t.Errorf("occupants = %+v, want HEAT.mkv of 7 bytes", o)
		}
		if len(probed.Occupants) != 1 {
			t.Errorf("an empty holding path has occupants: %+v", probed.Occupants)
		}
	})
}

func TestProbeMoveRefusesAPageOverTheLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPlacing(t, false)
		rels := make([]Rel, MaxProbePage+1)
		for i := range rels {
			rels[i] = mustRel(t, "a.mkv")
		}
		_, err := p.remote.ProbeMove(t.Context(), node, ProbeMove{Destinations: []ProbeRoot{{Root: p.at.Root, Rels: rels}}})
		if rpc.CodeOf(err) != rpc.CodeInvalidArgument {
			t.Errorf("a page of %d paths = %v, want it refused", len(rels), err)
		}
	})
}

func TestPlaceSidecarRecordsTheLedger(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		write(t, filepath.Join(r.folder, "old", "Heat.en.srt"), "ours")
		to := mustRel(t, HoldingDir+"/u.1.srt")
		if _, err := r.remote.PlaceSidecar(t.Context(), r.at, Library{r.at.Root, mustRel(t, "old/Heat.en.srt")}, to, true); err != nil {
			t.Fatal(err)
		}
		sha := sha256.Sum256([]byte("ours"))
		held, err := ledgerHolds(t.Context(), r.local, r.at.Root, [][]byte{sha[:]})
		if err != nil || len(held) != 1 {
			t.Errorf("the ledger holds %d rows of the placed Sidecar, %v; want 1", len(held), err)
		}
		if _, err := r.remote.Place(t.Context(), r.at, Library{r.at.Root, mustRel(t, "old/Heat.en.srt")}, mustRel(t, HoldingDir+"/u.2.srt"), true); err != nil {
			t.Fatal(err)
		}
	})
}

// The longest path a root's budget allows on Linux, the longest of the
// three systems' default limits, and the longest file name.
const (
	longestRel  = 4096 - 2
	longestName = 255
)

// TestProbePageFitsTheLink builds a page's worst answer: 2,000 source
// files, each with 20 companions, every path and name the longest a root
// allows, and asks that it and its request both fit one link message.
func TestProbePageFitsTheLink(t *testing.T) {
	const files, companions = 2_000, 20
	id := &storagev1.Identity{ModifiedUnixMs: time.Now().UnixMilli(), FileId: make([]byte, 16), VolumeId: strings.Repeat("f", 16)}
	req := &storagev1.ProbeMoveRequest{VideoExtensions: media.DefaultVideos.Extensions()}
	res := &storagev1.ProbeMoveResponse{}
	src := &storagev1.ProbeRoot{RootId: storage.NewUUID().Bytes()}
	probed := &storagev1.ProbedSource{}
	for range files {
		src.Rels = append(src.Rels, strings.Repeat("r", longestRel))
		f := &storagev1.ProbedFile{Size: 1 << 40, Identity: id}
		for range companions {
			f.Companions = append(f.Companions, &storagev1.ProbedCompanion{Name: strings.Repeat("c", longestName), Size: 1 << 30, Identity: id, Ledger: true})
		}
		probed.Files = append(probed.Files, f)
	}
	req.Sources, res.Sources = []*storagev1.ProbeRoot{src}, []*storagev1.ProbedSource{probed}
	for name, m := range map[string]proto.Message{"request": req, "answer": res} {
		if n := proto.Size(m); n >= link.MaxMessage {
			t.Errorf("the %s is %d bytes, want under link.MaxMessage %d", name, n, link.MaxMessage)
		}
	}
}
