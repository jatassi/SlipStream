package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// following is the Conversions a conversion Import reports to, keeping
// each report.
type following struct {
	mu      sync.Mutex
	reports []conversions.Report
}

func (f *following) Follow(_ context.Context, _ *storage.Tx[storage.Library], id conversions.ID, r conversions.Report) (conversions.Followed, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reports = append(f.reports, r)
	return conversions.Followed{Conversion: id}, nil
}

func (*following) Followed(context.Context, ...conversions.Followed) {}

// subtitling keeps, in order, each file subtitles hears land and each
// made file it hears of with its source.
type subtitling struct{ heard []string }

func (s *subtitling) Landed(_ context.Context, _ *storage.Tx[storage.Library], f files.File) error {
	s.heard = append(s.heard, "landed "+f.ID.String())
	return nil
}

func (s *subtitling) Derived(_ context.Context, _ *storage.Tx[storage.Library], source, made files.ID) error {
	s.heard = append(s.heard, "derived "+source.String()+" "+made.String())
	return nil
}

// made is a world whose Work w1 has a WEBDL-1080p source file of its
// first clip, which fills nothing, for a Conversion to make a
// Bluray-1080p file from.
type made struct {
	*world
	tree    works.Tree
	source  files.File
	follows *following
	subs    *subtitling
}

func newMade(t *testing.T) made {
	t.Helper()
	m := made{world: newWorld(t), follows: &following{}, subs: &subtitling{}}
	m.cfg.Conversions, m.cfg.Subtitles = m.follows, m.subs
	m.downloads(t, nil)
	m.library = library.New(library.Config{Subtitles: library.NoMarks{}, History: entrytest.Ignored("library", library.Entries()), Library: m.lib, Works: m.works, Roots: m.roots, Files: m.files, Imports: m.im, Leaver: m.im})
	m.tree = m.add(t, "w1", m.root.ID)
	err := m.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		m.source, err = files.Add(ctx, tx, files.File{
			WorkID: m.tree.ID, Items: []works.ItemID{m.tree.Items[0].ID}, Cut: release.Standard, RootID: m.root.ID, Path: "Shelf w1/Clip 1.WEB.mkv",
			Size: 1 << 30, Quality: release.WEBDL1080p, Media: m.probe, Seen: &storing.Seen{Modified: time.UnixMilli(1_700_000_000_000)},
		}, nil)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// handoff stores a landing Conversion of m's source into HD and hands
// imports its output, which it wrote into the root's holding folder.
func (m made) handoff(t *testing.T) (conversions.ID, *Import) {
	t.Helper()
	id := conversions.ID{UUID: storage.NewUUID()}
	items, err := json.Marshal([]works.ItemID{m.tree.Items[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := json.Marshal(conversions.Snapshot{Work: "Shelf w1", Target: "HD", SourceName: "Clip 1.WEB.mkv"})
	if err != nil {
		t.Fatal(err)
	}
	holding, err := storing.ParseRel(files.HoldingOf(id.UUID, ".mkv"))
	if err != nil {
		t.Fatal(err)
	}
	at := filepath.Join(m.root.Path, filepath.FromSlash(holding.String()))
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte("made"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := conversions.Handoff{
		Conversion: id, Work: m.tree.ID, Items: []works.ItemID{m.tree.Items[0].ID}, Edition: targets.AnyEdition, Root: m.root.ID, Holding: &holding,
		Media: m.probe, Title: release.Title("Clip.1.1080p.BluRay.x265", release.TitleOnly{}, nil), SourceName: "Clip 1.WEB - HD.mkv",
	}
	var imp ImportID
	err = m.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO conversions_conversion (id, work_id, type, items, edition, source_file_id, target_id, version,
			lane, position, stage, root_id, not_before, snapshot, queued_at) VALUES (?, ?, ?, ?, 'any', ?, ?, 3, 'new', 'a0', 'landing', ?, 0, ?, 0)`,
			id, m.tree.ID, contenttypetest.ID, string(items), m.source.ID, m.hd, m.root.ID, string(snapshot)); err != nil {
			return err
		}
		for range 2 {
			if err := m.im.Converter().Start(ctx, tx, h); err != nil {
				return err
			}
		}
		var err error
		imp, err = idOfConversion(ctx, tx, id)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id, m.stored(t, imp)
}

// idOfConversion is the one Import of Conversion id.
func idOfConversion(ctx context.Context, r storage.Reader[storage.Library], id conversions.ID) (ImportID, error) {
	var (
		imp ImportID
		n   int
	)
	err := r.QueryRowContext(ctx, `SELECT min(id), count(*) FROM imports_import WHERE conversion_id = ?`, id).Scan(&imp, &n)
	if err != nil {
		return ImportID{}, fmt.Errorf("read the Import of the Conversion: %w", err)
	}
	if n != 1 {
		return ImportID{}, fmt.Errorf("the Conversion has %d Imports, want one", n)
	}
	return imp, nil
}

// placeAll runs the place step of each of imp's placing files.
func (m made) placeAll(t *testing.T, imp *Import) *Import {
	t.Helper()
	for _, f := range imp.Files {
		if f.Outcome != Placing {
			continue
		}
		if err := m.im.place(t.Context(), *imp, f); err != nil {
			t.Fatal(err)
		}
	}
	return m.stored(t, imp.ID)
}

// madeFile is the library file imp's first file landed as.
func (m made) madeFile(t *testing.T, imp *Import) files.File {
	t.Helper()
	var out files.File
	err := m.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		out, err = files.PlayedOf(ctx, r, imp.Files[0].FileID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// landing is one way a conversion Import goes: whether a file blocks
// its name, what happens once it waits, and how its file ends.
type landing struct {
	name    string
	blocked bool
	then    func(t *testing.T, m made, id conversions.ID, imp *Import, taken string) *Import
	want    Outcome
	code    ReasonCode
	report  func(*Import) []conversions.Report
}

func placedReport(imp *Import) []conversions.Report {
	return []conversions.Report{{Placed: true, File: imp.Files[0].FileID}}
}

func retried(t *testing.T, m made, _ conversions.ID, imp *Import, taken string) *Import {
	if err := os.Remove(filepath.Join(m.root.Path, filepath.FromSlash(taken))); err != nil {
		t.Fatal(err)
	}
	got := m.act(t, imp, 0, RetryFile{})
	if got.Files[0].Outcome != Placing {
		t.Fatalf("after Retry the file is %s, want placing again", got.Files[0].Outcome)
	}
	return m.placeAll(t, &got)
}

func declined(t *testing.T, m made, _ conversions.ID, imp *Import, _ string) *Import {
	got := m.act(t, imp, 0, DeclineFile{})
	return &got
}

func workDeleted(t *testing.T, m made, _ conversions.ID, imp *Import, _ string) *Import {
	if _, _, err := m.library.Delete(t.Context(), identity.Admin{}, []works.WorkID{m.tree.ID}, library.Deleting{}); err != nil {
		t.Fatal(err)
	}
	return m.stored(t, imp.ID)
}

func conversionCanceled(t *testing.T, m made, id conversions.ID, imp *Import, _ string) *Import {
	err := m.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return m.im.Converter().End(ctx, tx, id)
	})
	if err != nil {
		t.Fatal(err)
	}
	return m.stored(t, imp.ID)
}

func TestAConversionImportLandsItsMadeFileOrWaitsAndReportsHowItEnded(t *testing.T) {
	t.Parallel()
	base := newMade(t)
	_, landed := base.handoff(t)
	taken := base.madeFile(t, base.placeAll(t, landed)).Path
	for _, c := range []landing{
		{name: "it lands", want: Placed, report: placedReport},
		{name: "a taken name waits, then a Retry decides it again and it lands", blocked: true, then: retried, want: Placed, report: placedReport},
		{name: "the Admin declines it", blocked: true, then: declined, want: Declined, code: ByAdmin,
			report: func(*Import) []conversions.Report { return []conversions.Report{{Declined: true}} }},
		{name: "its Work's delete ends it", blocked: true, then: workDeleted, want: Declined, code: WorkDeleted,
			report: func(*Import) []conversions.Report { return []conversions.Report{{}} }},
		{name: "its Conversion's cancel ends it and hears nothing back", blocked: true, then: conversionCanceled, want: Declined, code: ConversionCanceled,
			report: func(*Import) []conversions.Report { return nil }},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			newMade(t).goes(t, c, taken)
		})
	}
}

// goes hands m an output and places it as c says, with an untracked
// file at taken when c is blocked.
func (m made) goes(t *testing.T, c landing, taken string) {
	t.Helper()
	if c.blocked {
		m.block(t, taken)
	}
	id, imp := m.handoff(t)
	if f := imp.Files[0]; imp.Source != FromConversion || f.Outcome != Placing || f.Quality != release.Bluray1080p || !slices.Equal(f.Items, []works.ItemID{m.tree.Items[0].ID}) {
		t.Fatalf("the hand-off made a %s Import whose file is %s %s for %v, want a conversion Import placing Bluray-1080p for the first clip unmatched", imp.Source, f.Outcome, f.Quality, f.Items)
	}
	got := m.placeAll(t, imp)
	if c.blocked {
		m.waitsTaken(t, got, taken)
		got = c.then(t, m, id, got, taken)
	}
	f := got.Files[0]
	if f.Outcome != c.want || c.code != "" && f.Reason.Code != c.code || got.Finished.IsZero() {
		t.Fatalf("the made file is %s %+v, finished at %v, want %s %s and the Import finished", f.Outcome, f.Reason, got.Finished, c.want, c.code)
	}
	if want := c.report(got); !slices.Equal(m.follows.reports, want) {
		t.Errorf("the Import reported %+v, want %+v", m.follows.reports, want)
	}
	if c.want == Placed {
		m.landedQuietly(t, got)
	}
}

// block puts a file SlipStream doesn't track at rel in m's root.
func (m made) block(t *testing.T, rel string) {
	t.Helper()
	at := filepath.Join(m.root.Path, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte("not tracked"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// waitsTaken checks that imp's file waits name_taken for the file at
// taken, listed in Pending imports, with nothing reported yet.
func (m made) waitsTaken(t *testing.T, imp *Import, taken string) {
	t.Helper()
	if f := imp.Files[0]; f.Outcome != Waiting || f.Reason.Code != NameTaken {
		t.Fatalf("with an untracked file at %s the made file is %s %+v, want waiting name_taken", taken, f.Outcome, f.Reason)
	}
	if !slices.ContainsFunc(m.entries(t, false), func(e Import) bool { return e.ID == imp.ID }) {
		t.Fatal("Pending imports doesn't list the waiting conversion Import")
	}
	if len(m.follows.reports) != 0 {
		t.Fatalf("the waiting Import reported %+v, want nothing until it ends", m.follows.reports)
	}
}

// landedQuietly checks a placed conversion Import: its file placed
// converted with its provenance, subtitles told after the landing,
// Imported recorded quietly and Recent leaving it out.
func (m made) landedQuietly(t *testing.T, imp *Import) {
	t.Helper()
	f := m.madeFile(t, imp)
	if want := (files.Derived{From: m.source.ID, Target: m.hd, Version: 3}); imp.Files[0].PlacedBy != Converted || f.Derived == nil || *f.Derived != want {
		t.Errorf("the file placed by %q with provenance %+v, want converted from %+v", imp.Files[0].PlacedBy, f.Derived, want)
	}
	if want := []string{"landed " + f.ID.String(), "derived " + m.source.ID.String() + " " + f.ID.String()}; !slices.Equal(m.subs.heard, want) {
		t.Errorf("subtitles heard %v, want %v", m.subs.heard, want)
	}
	if abouts := entrytest.Abouts(m.notices, importedRow); len(abouts) != 1 || !abouts[0].Quiet {
		t.Errorf("history recorded Imported %+v, want once, quiet", abouts)
	}
	var recent Recent
	err := m.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		recent, err = m.im.Recent(ctx, r, 10, time.Time{})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(recent.Arrivals) != 0 || recent.Imported != 0 || recent.Upgraded != 0 {
		t.Errorf("Recent is %+v, want the conversion Import left out", recent)
	}
}

func TestAConversionsCancelIsRefusedWhileItsFilePlaces(t *testing.T) {
	t.Parallel()
	m := newMade(t)
	id, _ := m.handoff(t)
	err := m.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return m.im.Converter().End(ctx, tx, id)
	})
	if !errors.Is(err, conversions.ErrPlacing) {
		t.Errorf("ending the Import of a placing file answered %v, want ErrPlacing", err)
	}
}
