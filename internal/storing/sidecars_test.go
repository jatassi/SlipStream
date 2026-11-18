package storing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/speech"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
	"github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1/storagev1connect"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// gated is a Mapper whose builds wait until the test lets each path
// through, recording the order they started in.
type gated struct {
	mu      sync.Mutex
	started []string
	open    map[string]chan struct{}
	fail    map[string]error
	calls   map[string]int
}

func newGated() *gated {
	return &gated{open: map[string]chan struct{}{}, fail: map[string]error{}, calls: map[string]int{}}
}

func (g *gated) gate(path string) chan struct{} {
	g.mu.Lock()
	defer g.mu.Unlock()
	c, ok := g.open[path]
	if !ok {
		c = make(chan struct{})
		g.open[path] = c
	}
	return c
}

func (g *gated) let(path string) { close(g.gate(path)) }

func (g *gated) Map(ctx context.Context, path string, stream int) (speech.Map, error) {
	g.mu.Lock()
	g.started = append(g.started, filepath.Base(path))
	g.calls[filepath.Base(path)]++
	err := g.fail[filepath.Base(path)]
	g.mu.Unlock()
	select {
	case <-g.gate(filepath.Base(path)):
	case <-ctx.Done():
		return speech.Map{}, ctx.Err() //nolint:wrapcheck // a canceled build, as the real mapper answers it
	}
	if err != nil {
		return speech.Map{}, err
	}
	return speechtest.Fixed{Runs: []speechtest.Run{{Start: time.Second, End: 2 * time.Second}}, Length: 4 * time.Second}.Map(ctx, path, stream)
}

func (g *gated) order() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.started)
}

type sidecarRig struct {
	placing
	local  *storage.DB[storage.Local]
	maps   string
	mapper *gated
	sc     *Sidecars
	calls  SidecarCalls
}

const buildWait = 30 * time.Second

func sidecarTimings() SidecarTimings {
	return SidecarTimings{BuildWait: buildWait, MapCap: 1 << 20, SidecarCap: 64, Queue: 4, SurveyBatch: 3}
}

func newSidecarRig(t *testing.T, draining bool) *sidecarRig {
	t.Helper()
	r := &sidecarRig{placing: newPlacing(t, draining), local: storagetest.Local(t, SidecarSchema()), maps: filepath.Join(t.TempDir(), "speechmaps"), mapper: newGated()}
	r.open(t)
	return r
}

func (r *sidecarRig) open(t *testing.T) {
	t.Helper()
	sc, err := NewSidecars(t.Context(), r.store, r.local, r.maps, r.mapper, sidecarTimings())
	if err != nil {
		t.Fatal(err)
	}
	r.sc, r.calls = sc, SidecarCallsOf(sc)
}

var _ storagev1connect.SidecarServiceHandler = (*Sidecars)(nil)

// film writes a video under the root and answers its path and identity.
func (r *sidecarRig) film(t *testing.T, rel, body string) (Rel, Identity) {
	t.Helper()
	write(t, filepath.Join(r.folder, filepath.FromSlash(rel)), body)
	return mustRel(t, rel), r.identity(t, rel)
}

func (r *sidecarRig) identity(t *testing.T, rel string) Identity {
	t.Helper()
	info, err := r.disk.Stat(t.Context(), filepath.Join(r.folder, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return identityOfInfo(info)
}

func (r *sidecarRig) build(file storage.UUID, path Rel, slots, rank int) (MapAnswer, error) {
	return r.calls.Build(context.Background(), MapBuild{File: file, Root: r.at.Root, Path: path, Stream: -1, Slots: slots, Rank: rank})
}

type answered struct {
	a   MapAnswer
	err error
}

func (r *sidecarRig) buildAsync(file storage.UUID, path Rel, slots, rank int) <-chan answered {
	c := make(chan answered, 1)
	go func() {
		a, err := r.build(file, path, slots, rank)
		c <- answered{a, err}
	}()
	return c
}

func TestABuildJoinsTheRunningOneAndAnswersAtOnceOnceBuilt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		heat, id := r.film(t, "Heat (1995)/Heat (1995).mkv", "film")
		file := storage.NewUUID()
		first, second := r.buildAsync(file, heat, 1, 0), r.buildAsync(file, heat, 1, 0)
		synctest.Wait()
		r.mapper.let("Heat (1995).mkv")
		for _, c := range []<-chan answered{first, second} {
			got := <-c
			if got.err != nil || got.a.State != MapBuilt || !got.a.Identity.Same(id) || got.a.Frames != 400 || got.a.Density != 0.25 {
				t.Fatalf("Build = %+v, %v; want built from %+v with 400 frames at 0.25", got.a, got.err, id)
			}
		}
		again, err := r.build(file, heat, 1, 0)
		if err != nil || again.State != MapBuilt || !again.Identity.Same(id) {
			t.Fatalf("Build of a current map = %+v, %v", again, err)
		}
		if n := r.mapper.calls["Heat (1995).mkv"]; n != 1 {
			t.Errorf("the mapper ran %d times for three calls, want once", n)
		}
		b, err := r.calls.ReadMap(t.Context(), file, r.at.Root, heat, id)
		if err != nil {
			t.Fatal(err)
		}
		m, err := speech.Decode(b)
		if err != nil || m.Header.Frames != 400 || !m.Speech(150) || m.Speech(250) || !bytes.Equal(m.Header.Identity, id.encode()) {
			t.Fatalf("the map read back = %+v, %v", m.Header, err)
		}
	})
}

func TestBuildsPastTheSlotsWaitInRankOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		names := []string{"a.mkv", "b.mkv", "c.mkv", "d.mkv"}
		ranks := []int{0, 5, 1, 1}
		var answers []<-chan answered
		for i, n := range names {
			p, _ := r.film(t, n, n)
			answers = append(answers, r.buildAsync(storage.NewUUID(), p, 1, ranks[i]))
			synctest.Wait()
		}
		time.Sleep(buildWait)
		synctest.Wait()
		if got := <-answers[0]; got.err != nil || got.a.State != MapBuilding {
			t.Fatalf("the build in the slot = %+v, %v; want building", got.a, got.err)
		}
		for _, c := range answers[1:] {
			if got := <-c; got.err != nil || got.a.State != MapQueued {
				t.Fatalf("a build past the slots = %+v, %v; want queued", got.a, got.err)
			}
		}
		for _, n := range []string{"a.mkv", "c.mkv", "d.mkv", "b.mkv"} {
			r.mapper.let(n)
			synctest.Wait()
		}
		if got, want := r.mapper.order(), []string{"a.mkv", "c.mkv", "d.mkv", "b.mkv"}; !slices.Equal(got, want) {
			t.Fatalf("builds started %v, want %v: rank, then call order", got, want)
		}
	})
}

func TestABuildOutlivingItsWaitAnswersBuildingThenBuilt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		heat, _ := r.film(t, "heat.mkv", "film")
		file := storage.NewUUID()
		c := r.buildAsync(file, heat, 2, 0)
		time.Sleep(buildWait)
		synctest.Wait()
		if got := <-c; got.err != nil || got.a.State != MapBuilding {
			t.Fatalf("a build past its wait = %+v, %v; want building", got.a, got.err)
		}
		r.mapper.let("heat.mkv")
		synctest.Wait()
		if got, err := r.build(file, heat, 2, 0); err != nil || got.State != MapBuilt {
			t.Fatalf("the next call = %+v, %v; want built", got, err)
		}
	})
}

func TestAFailedBuildAnswersItsFault(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		for name, c := range map[string]struct {
			err   error
			fault MapFault
			words string
		}{
			"silent.mkv": {speech.ErrNoAudio, NoAudio, ""},
			"broken.mkv": {&speech.DecodeError{First: "Invalid data found when processing input"}, DecodeFailed, "Invalid data found when processing input"},
			"nofvad.mkv": {speech.ErrNodeFault, NodeFault, speech.ErrNodeFault.Error()},
		} {
			p, _ := r.film(t, name, name)
			r.mapper.fail[name] = c.err
			r.mapper.let(name)
			got, err := r.build(storage.NewUUID(), p, 1, 0)
			if err != nil || got.State != MapFailed || got.Fault != c.fault || got.Words != c.words {
				t.Errorf("%s: Build = %+v, %v; want failed with %v %q", name, got, err, c.fault, c.words)
			}
		}
	})
}

func TestAMapOfAFileThatChangedReadsStale(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		heat, id := r.film(t, "heat.mkv", "film")
		file := storage.NewUUID()
		r.mapper.let("heat.mkv")
		if _, err := r.build(file, heat, 1, 0); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		_, now := r.film(t, "heat.mkv", "a longer film")
		for _, asked := range []Identity{id, now} {
			if _, err := r.calls.ReadMap(t.Context(), file, r.at.Root, heat, asked); !errors.Is(err, ErrStale) {
				t.Errorf("ReadMap as %+v after the file changed = %v, want ErrStale", asked, err)
			}
		}
		got, err := r.build(file, heat, 1, 0)
		if err != nil || got.State != MapBuilt || !got.Identity.Same(now) || r.mapper.calls["heat.mkv"] != 2 {
			t.Fatalf("Build after the change = %+v, %v after %d builds; want a second build from %+v", got, err, r.mapper.calls["heat.mkv"], now)
		}
		if _, err := r.calls.ReadMap(t.Context(), file, r.at.Root, heat, now); err != nil {
			t.Fatalf("ReadMap of the rebuilt map = %v", err)
		}
	})
}

func TestPruneKeepsTheListedMapsAndTheSweepClearsWhatNoRowNames(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		ids := r.built(t, 4)
		n, err := r.calls.PruneMaps(t.Context(), ids[0], ids[3], []storage.UUID{ids[1]})
		if err != nil || n != 2 {
			t.Fatalf("PruneMaps = %d, %v; want 2 removed", n, err)
		}
		if got, want := r.held(t), []string{ids[1].String() + mapExt, ids[3].String() + mapExt}; !slices.Equal(got, want) {
			t.Fatalf("after a prune of [0, 3) keeping 1 the maps are %v, want %v", got, want)
		}

		write(t, filepath.Join(r.maps, storage.NewUUID().String()+mapExt), "orphan")
		write(t, filepath.Join(r.maps, ids[1].String()+mapExt+".part"), "cut short")
		if err := os.Remove(filepath.Join(r.maps, ids[3].String()+mapExt)); err != nil {
			t.Fatal(err)
		}
		r.open(t)
		if got, want := r.held(t), []string{ids[1].String() + mapExt}; !slices.Equal(got, want) {
			t.Fatalf("after the sweep the maps are %v, want %v", got, want)
		}
		if _, ok, err := r.sc.mapRow(t.Context(), ids[3]); ok || err != nil {
			t.Errorf("the row of a map that went stays: %v", err)
		}
		if n, err := r.calls.PruneMaps(t.Context(), storage.UUID{}, storage.UUID{}, nil); err != nil || n != 1 {
			t.Fatalf("PruneMaps of everything = %d, %v; want 1", n, err)
		}
	})
}

// built builds n maps and answers their files in order.
func (r *sidecarRig) built(t *testing.T, n int) []storage.UUID {
	t.Helper()
	ids := make([]storage.UUID, n)
	for i := range ids {
		ids[i] = storage.NewUUID()
		name := string(rune('a'+i)) + ".mkv"
		p, _ := r.film(t, name, name)
		r.mapper.let(name)
		if _, err := r.build(ids[i], p, 1, 0); err != nil {
			t.Fatal(err)
		}
	}
	slices.SortFunc(ids, storage.UUID.Compare)
	return ids
}

// held names the files in the maps folder.
func (r *sidecarRig) held(t *testing.T) []string {
	t.Helper()
	es, err := os.ReadDir(r.maps)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range es {
		names = append(names, e.Name())
	}
	return names
}

// there reports whether rel is under the root.
func (r *sidecarRig) there(t *testing.T, rel string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(r.folder, filepath.FromSlash(rel)))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	return err == nil
}

func (r *sidecarRig) mustSidecar(t *testing.T, video Rel, id Identity, tail string, b []byte) Rel {
	t.Helper()
	at, err := r.sidecar(video, id, tail, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	return at
}

func srt(text string) (b []byte, sha [32]byte) {
	b = []byte("1\n00:00:01,000 --> 00:00:02,000\n" + text + "\n")
	return b, sha256.Sum256(b)
}

func (r *sidecarRig) sidecar(video Rel, id Identity, tail string, b []byte, replaces *[32]byte) (Rel, error) {
	return r.calls.Write(context.Background(), SidecarWrite{Root: r.at.Root, Video: video, Identity: id, Tail: tail, Bytes: b, SHA256: sha256.Sum256(b), Replaces: replaces, Ledger: true})
}

func TestWriteSidecarTakesOnlyASidecarsTail(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		heat, id := r.film(t, "Heat (1995)/Heat (1995).mkv", "film")
		b, _ := srt("Hello")
		for _, tail := range []string{
			"en.hi.srt", "en.default.srt", "hi.srt.txt", "en/x.srt", `en\x.srt`, "../en.srt", "..srt", ".en.srt", "EN.srt", "En.srt", "en.vtt", "english.srt",
			"en-us.srt", "en.sdh.forced.srt", "0123abcd.EN.srt", "0123ABCD.en.srt", "123abcd.en.srt", "en.srt ",
		} {
			if _, err := r.sidecar(heat, id, tail, b, nil); !isInvalid(err) {
				t.Errorf("tail %q = %v, want refused", tail, err)
			}
		}
		for _, tail := range []string{"en.srt", "pt-BR.forced.sdh.ass", "zho.ssa", "0123abcd.en.forced.srt"} {
			got, err := r.sidecar(heat, id, tail, b, nil)
			if err != nil || got.String() != "Heat (1995)/Heat (1995)."+tail {
				t.Errorf("tail %q = %v, %v", tail, got, err)
			}
		}
		if entries, _ := os.ReadDir(filepath.Join(r.folder, "Heat (1995)")); len(entries) != 5 {
			t.Errorf("the folder holds %d files, want the film and four Sidecars", len(entries))
		}
	})
}

func isInvalid(err error) bool {
	return err != nil && !errors.Is(err, ErrStale) && !errors.Is(err, ErrOccupied)
}

func TestWriteSidecarRefusals(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		heat, id := r.film(t, "heat.mkv", "film")
		held, heldID := r.film(t, HoldingDir+"/0190.mkv", "film")
		b, sha := srt("Hello")
		write := func(w SidecarWrite) error { _, err := r.calls.Write(t.Context(), w); return err }
		base := SidecarWrite{Root: r.at.Root, Video: heat, Identity: id, Tail: "en.srt", Bytes: b, SHA256: sha}
		with := func(f func(*SidecarWrite)) SidecarWrite { w := base; f(&w); return w }
		other := id
		other.Seen.Modified = other.Seen.Modified.Add(time.Second)
		bigger := id
		bigger.Size++
		for name, c := range map[string]struct {
			w     SidecarWrite
			stale bool
		}{
			"in holding":      {with(func(w *SidecarWrite) { w.Video, w.Identity = held, heldID }), true},
			"no video":        {with(func(w *SidecarWrite) { w.Video = mustRel(t, "gone.mkv") }), true},
			"another mtime":   {with(func(w *SidecarWrite) { w.Identity = other }), true},
			"another size":    {with(func(w *SidecarWrite) { w.Identity = bigger }), true},
			"a wrong hash":    {with(func(w *SidecarWrite) { w.SHA256[0] ^= 1 }), false},
			"over the cap":    {with(func(w *SidecarWrite) { w.Bytes = make([]byte, 65); w.SHA256 = sha256.Sum256(w.Bytes) }), false},
			"an unknown root": {with(func(w *SidecarWrite) { w.Root = storage.NewUUID() }), false},
		} {
			err := write(c.w)
			if c.stale != errors.Is(err, ErrStale) || err == nil {
				t.Errorf("%s: Write = %v, want stale %v", name, err, c.stale)
			}
		}
		if entries, _ := os.ReadDir(r.folder); len(entries) != 3 {
			t.Errorf("the root holds %d entries after the refusals, want the film, holding and the marker", len(entries))
		}

		d := newSidecarRig(t, true)
		heat, id = d.film(t, "heat.mkv", "film")
		if _, err := d.sidecar(heat, id, "en.srt", b, nil); !notReady(t, err).Draining {
			t.Errorf("a write while draining = %v, want NotReady draining", err)
		}
	})
}

func TestWriteSidecarOverOtherBytesIsOccupied(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		heat, id := r.film(t, "heat.mkv", "film")
		found := write(t, filepath.Join(r.folder, "heat.en.srt"), "found by the Admin")
		foundSHA := sha256.Sum256([]byte("found by the Admin"))
		b, sha := srt("Hello")
		var occupied *OccupiedError
		if _, err := r.sidecar(heat, id, "en.srt", b, nil); !errors.As(err, &occupied) || occupied.SHA256 != foundSHA {
			t.Fatalf("a write onto a found subtitle = %v, want Occupied naming its hash", err)
		}
		wrong := sha
		if _, err := r.sidecar(heat, id, "en.srt", b, &wrong); !errors.As(err, &occupied) {
			t.Fatalf("a write replacing a hash that isn't there = %v, want Occupied", err)
		}
		if got, _ := os.ReadFile(found); string(got) != "found by the Admin" {
			t.Fatalf("the found subtitle holds %q", got)
		}
		if _, err := r.sidecar(heat, id, "en.srt", b, &foundSHA); err != nil {
			t.Fatalf("a write replacing the found hash = %v", err)
		}
		if _, err := r.sidecar(heat, id, "en.srt", b, nil); err != nil {
			t.Fatalf("the same bytes again = %v, want written at once", err)
		}
		if got, _ := os.ReadFile(found); !bytes.Equal(got, b) {
			t.Fatalf("the Sidecar holds %q", got)
		}
		if _, err := os.Stat(found + ".part"); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("a part file stayed: %v", err)
		}
	})
}

func TestReadSidecarAnswersOnlyBytesWithTheirHash(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		heat, id := r.film(t, "heat.mkv", "film")
		b, sha := srt("Hello")
		at := r.mustSidecar(t, heat, id, "en.srt", b)
		if got, err := r.calls.Read(t.Context(), r.at.Root, at, sha); err != nil || !bytes.Equal(got, b) {
			t.Fatalf("Read = %q, %v", got, err)
		}
		other, otherSHA := srt("Hello again")
		write(t, filepath.Join(r.folder, "heat.en.srt"), string(other))
		var changed *ChangedError
		if _, err := r.calls.Read(t.Context(), r.at.Root, at, sha); !errors.As(err, &changed) || changed.SHA256 != otherSHA {
			t.Fatalf("Read of other bytes = %v, want Changed naming them", err)
		}
		if err := os.Remove(filepath.Join(r.folder, "heat.en.srt")); err != nil {
			t.Fatal(err)
		}
		if _, err := r.calls.Read(t.Context(), r.at.Root, at, sha); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("Read of nothing = %v, want fs.ErrNotExist", err)
		}
	})
}

func TestRemoveSidecarTakesOnlyBytesWithTheirHash(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		heat, id := r.film(t, "heat.mkv", "film")
		b, sha := srt("Hello")
		at := r.mustSidecar(t, heat, id, "en.srt", b)
		other, otherSHA := srt("Hello again")
		write(t, filepath.Join(r.folder, "heat.en.srt"), string(other))
		var changed *ChangedError
		if err := r.calls.Remove(t.Context(), r.at.Root, at, sha); !errors.As(err, &changed) || changed.SHA256 != otherSHA {
			t.Fatalf("Remove of other bytes = %v, want Changed naming them", err)
		}
		if !r.there(t, "heat.en.srt") {
			t.Fatal("Remove took bytes of another hash")
		}
		if err := r.calls.Remove(t.Context(), r.at.Root, at, otherSHA); err != nil {
			t.Fatalf("Remove by hash = %v", err)
		}
		if r.there(t, "heat.en.srt") {
			t.Fatal("Remove left the file")
		}
		if err := r.calls.Remove(t.Context(), r.at.Root, at, otherSHA); err != nil {
			t.Fatalf("Remove of nothing = %v, want removed", err)
		}
		if err := r.calls.Remove(t.Context(), r.at.Root, heat, sha); err == nil || errors.Is(err, ErrChanged) {
			t.Fatalf("Remove of the video = %v, want refused", err)
		}
	})
}

func TestTheLedgerKeepsAHashPastItsRemoveAndListsStemsOfAMissingVideo(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newSidecarRig(t, false)
		heat, id := r.film(t, "Heat/Heat.mkv", "film")
		b, sha := srt("Hello")
		r.mustSidecar(t, heat, id, "en.srt", b)
		second := r.mustSidecar(t, heat, id, "0123abcd.en.srt", b)
		if err := r.calls.Remove(t.Context(), r.at.Root, second, sha); err != nil {
			t.Fatal(err)
		}
		moved, _ := srt("Moved")
		if _, err := r.calls.Write(t.Context(), SidecarWrite{Root: r.at.Root, Video: heat, Identity: id, Tail: "it.srt", Bytes: moved, SHA256: sha256.Sum256(moved)}); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(r.folder, "Heat", "Heat.fr.srt"), "found")
		write(t, filepath.Join(r.folder, "Heat", "Heat.nfo"), "not a subtitle")
		write(t, filepath.Join(r.folder, "Old", "Heat 1995.de.ass"), "parked")
		write(t, filepath.Join(r.folder, HoldingDir, "0190.es.srt"), "held")
		if err := os.Rename(filepath.Join(r.folder, "Heat", "Heat.mkv"), filepath.Join(r.folder, "Heat", "Heat.moved")); err != nil {
			t.Fatal(err)
		}
		listed, err := r.calls.List(t.Context(), r.at.Root, media.DefaultVideos, []SidecarListing{{
			Video: heat, Also: []Rel{mustRel(t, "Old/Heat 1995.mkv"), mustRel(t, HoldingDir+"/0190.mkv")},
		}})
		if err != nil {
			t.Fatal(err)
		}
		if len(listed) != 1 || !listed[0].VideoMissing {
			t.Fatalf("listed %+v, want one missing video", listed)
		}
		want := []listedCompanion{{"Heat/Heat.en.srt", "en.srt", true}, {"Heat/Heat.fr.srt", "fr.srt", false}, {"Heat/Heat.it.srt", "it.srt", false}, {"Old/Heat 1995.de.ass", "de.ass", false}, {HoldingDir + "/0190.es.srt", "es.srt", false}}
		if got := listedCompanions(listed[0]); !slices.Equal(got, want) {
			t.Fatalf("companions = %+v, want %+v", got, want)
		}
		if err := r.calls.Remove(t.Context(), r.at.Root, mustRel(t, "Heat/Heat.en.srt"), sha); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(r.folder, "Heat", "Heat.en.srt"), string(b))
		listed, err = r.calls.List(t.Context(), r.at.Root, media.DefaultVideos, []SidecarListing{{Video: heat}})
		if err != nil || len(listed[0].Companions) != 3 || !listed[0].Companions[0].Ledger {
			t.Fatalf("after both removes the same bytes list as %+v, %v; want the ledger to hold them", listed, err)
		}
		if _, err := r.calls.List(t.Context(), r.at.Root, media.DefaultVideos, make([]SidecarListing, 4)); err == nil {
			t.Error("a listing past the survey batch was answered")
		}
	})
}

type listedCompanion struct {
	path, tail string
	ledger     bool
}

func listedCompanions(l SidecarListed) []listedCompanion {
	var out []listedCompanion
	for _, c := range l.Companions {
		out = append(out, listedCompanion{c.Path.String(), c.Tail, c.Ledger})
	}
	return out
}
