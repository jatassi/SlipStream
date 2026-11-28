package files_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// converting is a library's files.Conversions: cells answers Cells, cost
// answers Cost, and it records what files asks and tells.
type converting struct {
	cells   map[files.CellKey]files.ConversionCell
	cost    files.MadeCost
	asks    []files.Ask
	costed  [][]files.MadePair
	leaving []files.ID
}

func (c *converting) Cells(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[files.CellKey]files.ConversionCell, error) {
	return c.cells, nil
}

func (*converting) Holding(context.Context, storage.Reader[storage.Library]) (map[roots.ID]map[string]bool, error) {
	return nil, nil
}

func (c *converting) Queue(_ context.Context, _ *storage.Tx[storage.Library], asks ...files.Ask) error {
	c.asks = append(c.asks, asks...)
	return nil
}

func (c *converting) Leaving(_ context.Context, _ *storage.Tx[storage.Library], d files.File, _ files.ID) error {
	c.leaving = append(c.leaving, d.ID)
	return nil
}

func (*converting) Changed(context.Context, *storage.Tx[storage.Library], []files.PathChange) error {
	return nil
}

func (c *converting) Cost(_ context.Context, _ *storage.Tx[storage.Library], made []files.MadePair) (files.MadeCost, error) {
	c.costed = append(c.costed, made)
	return c.cost, nil
}

func (*converting) Remakes(context.Context, *storage.Tx[storage.Library], targets.TargetID, int, []works.WorkID, bool) (files.RemakeImpact, error) {
	return files.RemakeImpact{}, nil
}

// made is a library whose Profiles download UHD and make Mobile, a
// Bluray or WEB-DL 1080p Target with Conversion settings, from it.
type made struct {
	*library
	conv        *converting
	uhd, mobile targets.Target
}

func newMade(t *testing.T) made {
	t.Helper()
	l := open(t)
	conv := &converting{}
	l.conversions = conv
	s := encode.Defaults()
	s.Kbps = 6000
	mobile, err := l.targets.CreateTarget(t.Context(), admin, targets.Spec{
		Name: "Mobile", Tiers: [][]release.Quality{{release.Bluray1080p}, {release.WEBDL1080p}}, Cutoff: release.Bluray1080p, Scores: targets.Scores{Increment: 1},
		Sizes:      map[release.Quality]targets.Size{release.Bluray1080p: {}, release.WEBDL1080p: {}},
		Conversion: &targets.Conversion{Settings: s},
	})
	if err != nil {
		t.Fatal(err)
	}
	mobile = must[targets.Target](t)(l.targets.GetTarget(t.Context(), admin, mobile.ID))
	return made{library: l, conv: conv, uhd: l.target(t, "UHD", []release.Quality{release.WEBDL2160p}), mobile: mobile}
}

// entries is UHD downloaded and Mobile sourced as sourcing from it.
func (m made) entries(sourcing targets.Sourcing) []targets.Entry {
	e := targets.Entry{Target: m.mobile.ID, Sourcing: sourcing, From: &m.uhd.ID}
	if sourcing == targets.DownloadThenDerive {
		e.Grace = targets.DefaultGrace
	}
	return []targets.Entry{{Target: m.uhd.ID, Sourcing: targets.Download}, e}
}

func (m made) madeProfile(t *testing.T, name string, sourcing targets.Sourcing) targets.ProfileID {
	t.Helper()
	p, err := m.targets.CreateProfile(t.Context(), admin, targets.ProfileSpec{Name: name, ContentType: contenttypetest.ID, Entries: m.entries(sourcing)})
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

// add records a file of item at q filling fills, made from derived when
// set.
func (m made) add(t *testing.T, tree works.Tree, item int, p string, q release.Quality, derived *files.Derived, fills ...targets.TargetID) files.File {
	t.Helper()
	var out files.File
	err := m.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		out, err = files.Add(ctx, tx, files.File{
			WorkID: tree.ID, Items: []works.ItemID{tree.Items[item].ID}, Cut: release.Standard, RootID: m.root.ID, Path: p, Size: 1024, Quality: q, Derived: derived,
		}, fills)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (m made) version() int { return m.mobile.Conversion.Version }

func TestAMadeFileLandsInEveryTargetItMeetsUnderItsKeptNameWithItsProvenance(t *testing.T) {
	m := newMade(t)
	hd := m.target(t, "HD", []release.Quality{release.Bluray1080p})
	p, err := m.targets.CreateProfile(t.Context(), admin, targets.ProfileSpec{Name: "Movies", ContentType: contenttypetest.ID, Entries: append(m.entries(targets.Derive), targets.Entry{Target: hd.ID, Sourcing: targets.Download})})
	if err != nil {
		t.Fatal(err)
	}
	err = naming.New(naming.Config{Library: m.lib, Facets: map[works.TypeID]naming.Facet{contenttypetest.ID: m.fx}, Describe: m.works.Describe, Budget: roots.SmallestBudget, Changed: discard[naming.Changed]{}}).
		Panel(contenttypetest.ID).Apply(t.Context(), setting.Patch{string(contenttypetest.ID) + ".naming.renameFiles": json.RawMessage(`false`)})
	if err != nil {
		t.Fatal(err)
	}
	tree := m.work(t, "w1", true, &p.ID)
	svc := m.files(files.NoGrabs{})
	source := m.download(t, svc, tree, 0, release.WEBDL2160p, "uhd")
	m.conv.asks = nil

	conversion := storage.NewUUID()
	holding := files.HoldingOf(conversion, ".mkv")
	m.put(t, holding, "made")
	kept := "Heat.2160p.WEB - Mobile.mkv"
	provenance := files.Derived{From: source.ID, Target: m.mobile.ID, Version: m.version()}
	var landed files.Landed
	err = m.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		c := holdingOf(tree, 0, holding, 4, media.Record{Video: media.Video{Width: 1920, Height: 800}}, release.Bluray1080p)
		c.SourceName = kept
		landed, err = svc.Import(ctx, tx, files.Landing{
			Batch: files.Batch{Work: tree.ID, Root: m.root.ID, Import: conversion, Candidates: []files.Candidate{c}}, PlacedBy: "converted", Arrival: files.ArrivalMade, Derived: &provenance,
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := targetsOf(landed.Fills); !slices.Equal(got, []targets.TargetID{m.mobile.ID, hd.ID}) {
		t.Fatalf("the made file fills %v, want Mobile and HD, every Target it meets, in Profile order", got)
	}
	if base := path.Base(landed.File.Path); base != kept {
		t.Fatalf("with renaming off the made file is named %q, want its kept name %q", base, kept)
	}
	if len(m.conv.asks) != 0 {
		t.Fatalf("landing a made file asked %+v, want no Conversion converged by its own output", m.conv.asks)
	}
	_, cells, _, err := svc.WorkCells(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cells {
		if c.Item.ID != tree.Items[0].ID || c.Target.ID == m.uhd.ID {
			continue
		}
		if c.File == nil || c.File.Derived == nil || *c.File.Derived != provenance || !c.File.Facts().Derived {
			t.Fatalf("%s's cell holds %+v, want the made file with provenance %+v", c.Target.Name, c.File, provenance)
		}
	}
	listed, err := svc.WorkFiles(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	if i := slices.IndexFunc(listed.Files, func(f files.Listed) bool { return f.ID == landed.File.ID }); i < 0 || listed.Files[i].PlacedBy != "converted" {
		t.Fatalf("the Work's files list %+v, want the made file placed converted", listed.Files)
	}
	var tallies []files.RecordTally
	err = m.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		tallies, err = files.TalliesSince(ctx, r, time.Time{})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tallies) != 0 {
		t.Fatalf("TalliesSince = %+v, want the conversion Import left out and no other Import recorded", tallies)
	}
}

func TestADownloadMeetingADownloadedOrMadeCellDoomsTheMadeFile(t *testing.T) {
	for _, c := range []struct {
		sourcing targets.Sourcing
		doomed   bool
	}{
		{targets.DownloadThenDerive, true},
		{targets.Derive, false},
	} {
		t.Run(string(c.sourcing), func(t *testing.T) {
			m := newMade(t)
			p := m.madeProfile(t, "Movies", c.sourcing)
			tree := m.work(t, "w1", true, &p)
			svc := m.files(files.NoGrabs{})
			source := m.add(t, tree, 0, "Shelf w1/uhd.mkv", release.WEBDL2160p, nil, m.uhd.ID)
			mine := m.add(t, tree, 0, "Shelf w1/made.mkv", release.Bluray1080p, &files.Derived{From: source.ID, Target: m.mobile.ID, Version: m.version()}, m.mobile.ID)

			holding := files.HoldingOf(storage.NewUUID(), ".mkv")
			m.put(t, holding, "webdl")
			var landed files.Landed
			err := m.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				var err error
				landed, err = m.land(ctx, tx, svc, tree.ID, holdingOf(tree, 0, holding, 5, media.Record{Video: media.Video{Width: 1920, Height: 1080}}, release.WEBDL1080p))
				return err
			})
			var refused *files.Refused
			if c.doomed && err != nil || !c.doomed && !errors.As(err, &refused) {
				t.Fatalf("landing a WEB-DL below the made Bluray answered %v", err)
			}
			want := mine.ID
			if c.doomed {
				want = landed.File.ID
			}
			cells := m.cellsOf(t, svc, tree.ID)
			i := slices.IndexFunc(cells, func(cell files.Cell) bool { return cell.Item.ID == tree.Items[0].ID && cell.Target.ID == m.mobile.ID })
			if cells[i].File == nil || cells[i].File.ID != want {
				t.Fatalf("Mobile's cell holds %+v, want %s", cells[i].File, want)
			}
			if doomed := slices.Contains(m.conv.leaving, mine.ID); doomed != c.doomed {
				t.Fatalf("Conversions heard the made file leave: %t, want %t", doomed, c.doomed)
			}
		})
	}
}

func TestAProfileSaveCountsTheFilesItMakesAndRefusesMoreThanConfirmed(t *testing.T) {
	m := newMade(t)
	m.conv.cost = files.MadeCost{Works: 2, Seconds: 3600, Bytes: 9 << 30}
	p := m.profile(t, "Movies", m.uhd)
	full, half, bare := m.work(t, "w1", true, &p), m.work(t, "w2", true, &p), m.work(t, "w3", true, &p)
	sources := []files.File{
		m.add(t, full, 0, "Shelf w1/a.mkv", release.WEBDL2160p, nil, m.uhd.ID),
		m.add(t, full, 1, "Shelf w1/b.mkv", release.WEBDL2160p, nil, m.uhd.ID),
		m.add(t, half, 0, "Shelf w2/a.mkv", release.WEBDL2160p, nil, m.uhd.ID),
	}
	svc := m.files(files.NoGrabs{})
	add := targets.UpdateProfile{ID: p, Spec: targets.ProfileSpec{Name: "Movies", ContentType: contenttypetest.ID, Entries: m.entries(targets.Derive)}}

	preview, err := svc.Change(t.Context(), admin, add, files.Mode{Unwanted: files.Delete, Preview: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := preview.Made; got.Files != 3 || got.Works != 2 || got.Seconds != 3600 || got.Bytes != 9<<30 {
		t.Fatalf("the preview's Made = %+v, want 3 files, one per source file, with Cost's estimate", got)
	}
	var pairs []files.MadePair
	for _, s := range sources {
		pairs = append(pairs, files.MadePair{Work: s.WorkID, Source: s.ID, Target: m.mobile.ID})
	}
	if got := m.conv.costed[0]; len(got) != 3 || slices.ContainsFunc(pairs, func(p files.MadePair) bool { return !slices.Contains(got, p) }) {
		t.Fatalf("Cost was asked of %+v, want %+v: Shelf w3 has no file to make one from", got, pairs)
	}

	_, err = svc.Change(t.Context(), admin, add, files.Mode{Unwanted: files.Delete, Confirmed: new(0), ConfirmedMade: new(2)})
	if e, ok := errors.AsType[*files.ImpactChangedError](err); !ok || !e.Made || e.Files || e.ConfirmedMade != 2 || e.Impact.Made.Files != 3 {
		t.Fatalf("a save making 3 files after 2 were confirmed answered %v, want impact_changed naming the made files", err)
	}
	m.conv.asks = nil
	if _, err := svc.Change(t.Context(), admin, add, files.Mode{Unwanted: files.Delete, Confirmed: new(0), ConfirmedMade: new(3)}); err != nil {
		t.Fatalf("a save confirming its 3 made files: %v", err)
	}
	var asked []works.WorkID
	for _, a := range m.conv.asks {
		if a.Why != files.ArrivalProfile || a.Backfill {
			t.Fatalf("the save asked %+v, want a Profile Ask that isn't a backfill", a)
		}
		asked = append(asked, a.Work)
	}
	slices.SortFunc(asked, func(a, b works.WorkID) int { return a.Compare(b.UUID) })
	want := []works.WorkID{full.ID, half.ID, bare.ID}
	slices.SortFunc(want, func(a, b works.WorkID) int { return a.Compare(b.UUID) })
	if !slices.Equal(asked, want) {
		t.Fatalf("the save asked for the Conversions of %v, want every Work of the Profile %v", asked, want)
	}
}

// tallying is a library's files.Subtitles whose Languages records how
// many Works want from its Profile then.
type tallying struct {
	*hearing
	works []int
}

func (c *tallying) Languages(ctx context.Context, tx *storage.Tx[storage.Library], p targets.ProfileID, was, now []targets.WantedLanguage, preview bool) (files.SubtitlesImpact, error) {
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM targets_wants WHERE profile_id = ?`, p).Scan(&n); err != nil {
		return files.SubtitlesImpact{}, fmt.Errorf("count the Profile's Works: %w", err)
	}
	c.works = append(c.works, n)
	return c.hearing.Languages(ctx, tx, p, was, now, preview)
}

func TestASplitProfileMakesOnlyTheConvertedWorksFilesAndTellsSubtitlesOnceTheRestLeft(t *testing.T) {
	m := newMade(t)
	h := &tallying{hearing: m.hear(t)}
	m.subtitles = h
	p := m.profile(t, "Movies", m.uhd)
	convert, rest := m.work(t, "w1", true, &p), m.work(t, "w2", true, &p)
	m.add(t, convert, 0, "Shelf w1/a.mkv", release.WEBDL2160p, nil, m.uhd.ID)
	m.add(t, rest, 0, "Shelf w2/a.mkv", release.WEBDL2160p, nil, m.uhd.ID)
	svc := m.files(files.NoGrabs{})
	english := targets.WantedLanguage{Language: lang(t, "en"), HI: targets.Allow, Audio: targets.NoAudio}
	split := targets.SplitProfile{
		Profile: p, Rest: []works.WorkID{rest.ID}, To: targets.ProfileTarget{NewName: "Movies (without Mobile)"},
		Spec: targets.ProfileSpec{Name: "Movies", ContentType: contenttypetest.ID, Entries: m.entries(targets.Derive), Languages: []targets.WantedLanguage{english}},
	}
	h.impact = files.SubtitlesImpact{Gained: []files.GainedLanguage{{Language: english.Language, Files: 1}}}

	impact, err := svc.Change(t.Context(), admin, split, files.Mode{Unwanted: files.Delete, Confirmed: new(0), ConfirmedMade: new(1)})
	if err != nil {
		t.Fatal(err)
	}
	if impact.Made.Files != 1 || impact.Made.Matching != 1 || len(impact.Subtitles.Gained) != 1 {
		t.Fatalf("the split's Impact made %d files, moved %d Works and gained %+v, want 1, 1 and English", impact.Made.Files, impact.Made.Matching, impact.Subtitles.Gained)
	}
	if a := m.conv.asks; len(a) != 1 || a[0].Work != convert.ID || a[0].Why != files.ArrivalProfile || !a[0].Backfill {
		t.Fatalf("the split asked %+v, want a backfill Ask of the converted Work alone", m.conv.asks)
	}
	if !slices.Equal(h.works, []int{1}) {
		t.Fatalf("subtitles heard the languages change with %v Works on the Profile, want once, with the converted Work alone", h.works)
	}
}

func TestAMadeFromCellIsSourcedThroughTheFileItsSourceCellsMadeFileWasMadeFrom(t *testing.T) {
	m := newMade(t)
	s := encode.Defaults()
	s.Kbps = 3000
	tiny := must[targets.Target](t)(m.targets.CreateTarget(t.Context(), admin, targets.Spec{
		Name: "Tiny", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p, Scores: targets.Scores{Increment: 1},
		Sizes: map[release.Quality]targets.Size{release.WEBDL1080p: {}}, Conversion: &targets.Conversion{Settings: s},
	}))
	p := must[targets.Profile](t)(m.targets.CreateProfile(t.Context(), admin, targets.ProfileSpec{
		Name: "Movies", ContentType: contenttypetest.ID,
		Entries: append(m.entries(targets.DownloadThenDerive), targets.Entry{Target: tiny.ID, Sourcing: targets.Derive, From: &m.mobile.ID}),
	}))
	tree := m.work(t, "w1", true, &p.ID)
	source := m.add(t, tree, 0, "Shelf w1/uhd.mkv", release.WEBDL2160p, nil, m.uhd.ID)
	madeFile := m.add(t, tree, 0, "Shelf w1/mobile.mkv", release.Bluray1080p, &files.Derived{From: source.ID, Target: m.mobile.ID, Version: m.version()}, m.mobile.ID)
	svc := m.files(files.NoGrabs{})

	var got map[works.WorkID][]files.Sourced
	var located files.Located
	err := m.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if got, err = svc.Sourced(ctx, r, works.Only([]works.WorkID{tree.ID})); err != nil {
			return err
		}
		located, err = svc.Located(ctx, r, source.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		item     works.ItemID
		target   targets.TargetID
		file     files.ID
		source   files.ID
		download bool
	}
	var rows []row
	for _, c := range got[tree.ID] {
		r := row{item: c.Items[0], target: c.Target, download: c.FilledByDownload()}
		if c.File != nil {
			r.file = c.File.ID
		}
		if src, ok := c.Source(); ok {
			r.source = src.ID
		}
		rows = append(rows, r)
	}
	scarf, mittens := tree.Items[0].ID, tree.Items[1].ID
	want := []row{
		{item: scarf, target: m.mobile.ID, file: madeFile.ID, source: source.ID},
		{item: scarf, target: tiny.ID, source: source.ID},
		{item: mittens, target: m.mobile.ID},
		{item: mittens, target: tiny.ID},
	}
	if !slices.Equal(rows, want) {
		t.Fatalf("Sourced = %+v, want %+v: Tiny is made from the UHD file Mobile's made file came from", rows, want)
	}
	if located.Root != m.root.ID || located.Node != m.root.Node || located.Rel.String() != source.Path || located.Name != "uhd.mkv" {
		t.Fatalf("Located = %+v, want the UHD file in its root on its Node", located)
	}
}
