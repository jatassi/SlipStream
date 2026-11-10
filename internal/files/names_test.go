package files

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func newTarget(t *testing.T, name string, tiers ...[]release.Quality) targets.Target {
	t.Helper()
	target, err := targets.NewTarget(targets.Spec{Scores: targets.Scores{Increment: 1}, Name: name, Tiers: tiers, Cutoff: tiers[0][0]})
	if err != nil {
		t.Fatal(err)
	}
	target.ID = targets.TargetID{UUID: storage.NewUUID()}
	return target
}

// shelf names a test Work's files in a folder per Shelf, by its template.
type shelf string

func (file shelf) Naming() naming.Declaration {
	return naming.Declaration{
		Tokens: []naming.Token{{Name: "Shelf", Kind: naming.KindTitle, Scope: naming.WorkScope}, {Name: "Thing", Kind: naming.KindTitle, Scope: naming.ItemScope}},
		Folder: "{Shelf}",
		Files:  []naming.Variant{{Key: "standard", Label: "File", File: string(file), Anchor: naming.TokenAnchor("Thing")}},
	}
}

func (shelf) NamingValues(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[works.WorkID]naming.Values, error) {
	return nil, nil
}

// shelved is the world of a Work on the shelf Attic whose items are each
// a Scarf, named by file under goals.
func shelved(t *testing.T, file string, goals targets.Goals, items ...works.Item) world {
	t.Helper()
	scheme, err := naming.DefaultScheme(shelf(file).Naming())
	if err != nil {
		t.Fatal(err)
	}
	values := naming.Values{Work: naming.Set{"Shelf": naming.Text("Attic")}, Items: map[works.ItemID]naming.ItemValues{}}
	for _, it := range items {
		values.Items[it.ID] = naming.ItemValues{Set: naming.Set{"Thing": naming.Text("Scarf")}}
	}
	return world{
		tree: works.Bare{Items: items}, goals: goals, scoring: newScoring(nil, goals, release.Fact[release.Language]{}),
		scheme: scheme, values: values, budgets: map[roots.ID]int{{}: 250},
	}
}

func TestNames(t *testing.T) {
	best := newTarget(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	web := newTarget(t, "Web", []release.Quality{release.WEBDL1080p})
	uhd := newTarget(t, "UHD", []release.Quality{release.WEBDL2160p})
	scarf := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}}
	ids := map[string]ID{}
	for _, n := range []string{"a", "b", "c", "d", "fresh"} {
		ids[n] = ID{UUID: storage.NewUUID()}
	}
	type seed struct {
		id      string
		quality release.Quality
		label   string
		extra   int64
		path    string
	}
	for _, c := range []struct {
		name  string
		files []seed
		fills map[*targets.Target]string
		want  map[string]name
	}{
		{
			name:  "a file is labeled after the first Target in Profile order it fills, and a new label keeps its folder",
			files: []seed{{"a", release.WEBDL1080p, "Web", 0, "Old Attic/Scarf, as it was named.mkv"}},
			fills: map[*targets.Target]string{&web: "a", &best: "a"},
			want:  map[string]name{"a": {"Best", 0, "Old Attic/Attic - Scarf - Best.mkv"}},
		},
		{
			name:  "an unchanged label keeps the path the file has",
			files: []seed{{"a", release.WEBDL1080p, "Best", 0, "Old Attic/Scarf, as it was named.mkv"}},
			fills: map[*targets.Target]string{&best: "a", &web: "a"},
			want:  map[string]name{"a": {"Best", 0, "Old Attic/Scarf, as it was named.mkv"}},
		},
		{
			name: "a Kept file takes the first Target it meets and the lowest N free in its folder",
			files: []seed{
				{"a", release.Bluray1080p, "Best", 0, "Attic/Attic - Scarf - Best.mkv"},
				{"b", release.WEBDL1080p, "Best", 0, "Attic/Attic - Scarf - Best (old).mkv"},
				{"c", release.WEBDL2160p, "UHD", 2, "Attic/Attic - Scarf - UHD (extra 2).mkv"},
			},
			fills: map[*targets.Target]string{&best: "a"},
			want: map[string]name{
				"a": {"Best", 0, "Attic/Attic - Scarf - Best.mkv"},
				"b": {"Best", 1, "Attic/Attic - Scarf - Best (extra 1).mkv"},
				"c": {"UHD", 2, "Attic/Attic - Scarf - UHD (extra 2).mkv"},
			},
		},
		{
			name: "a Kept file keeps its N while it is free, and one whose N another holds takes the lowest free",
			files: []seed{
				{"a", release.WEBDL1080p, "Web", 1, "Attic/Attic - Scarf - Web (extra 1).mkv"},
				{"b", release.WEBDL1080p, "Web", 1, "Attic/Copy - Web (extra 1).mkv"},
				{"c", release.WEBDL1080p, "Web", 3, "Attic/Attic - Scarf - Web (extra 3).mkv"},
				{"d", release.WEBDL1080p, "Web", 1, "Den/Attic - Scarf - Web (extra 1).mkv"},
			},
			want: map[string]name{
				"a": {"Best", 1, "Attic/Attic - Scarf - Best (extra 1).mkv"},
				"b": {"Best", 2, "Attic/Attic - Scarf - Best (extra 2).mkv"},
				"c": {"Best", 3, "Attic/Attic - Scarf - Best (extra 3).mkv"},
				"d": {"Best", 1, "Den/Attic - Scarf - Best (extra 1).mkv"},
			},
		},
		{
			name:  "a Kept file that meets no Target keeps its label",
			files: []seed{{"a", release.HDTV720p, "Gone", 0, "Attic/Attic - Scarf - Gone.mkv"}},
			want:  map[string]name{"a": {"Gone", 1, "Attic/Attic - Scarf - Gone (extra 1).mkv"}},
		},
		{
			name:  "a Kept file that fills again takes its name back",
			files: []seed{{"a", release.WEBDL2160p, "UHD", 1, "Attic/Attic - Scarf - UHD (extra 1).mkv"}},
			fills: map[*targets.Target]string{&uhd: "a"},
			want:  map[string]name{"a": {"UHD", 0, "Attic/Attic - Scarf - UHD.mkv"}},
		},
		{
			name:  "the file being placed takes the template's folder too",
			files: []seed{{"fresh", release.WEBDL1080p, "", 0, ".slipstream-incoming/grab.mkv"}},
			fills: map[*targets.Target]string{&web: "fresh"},
			want:  map[string]name{"fresh": {"Web", 0, "Attic/Attic - Scarf - Web.mkv"}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := shelved(t, "{Shelf} - {Thing} - {Edition Tag} {Target}", targets.Goals{Targets: []targets.Target{best, web, uhd}}, scarf)
			w.fresh = ids["fresh"]
			for _, s := range c.files {
				w.files = append(w.files, File{ID: ids[s.id], ItemID: scarf.ID, Cut: release.Standard, Quality: s.quality, Label: s.label, Extra: s.extra, Path: s.path})
			}
			fills := map[CellKey]Fill{}
			for target, id := range c.fills {
				fills[CellKey{Item: scarf.ID, Edition: targets.AnyEdition, Target: target.ID}] = Fill{File: ids[id]}
			}
			got, _, err := names(w, fills)
			if err != nil {
				t.Fatal(err)
			}
			want := map[ID]name{}
			for id, nm := range c.want {
				want[ids[id]] = nm
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("names =\n%v\nwant\n%v", got, want)
			}
		})
	}
}

func TestAFileBeingPlacedOnAnotherFilesNameFindsItTaken(t *testing.T) {
	web := newTarget(t, "Web", []release.Quality{release.WEBDL1080p})
	scarf := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}}
	held := File{ID: ID{UUID: storage.NewUUID()}, ItemID: scarf.ID, Cut: release.Standard, Quality: release.WEBDL1080p, Label: "Web", Path: "Attic/Attic - Scarf.mkv"}
	fresh := File{ID: ID{UUID: storage.NewUUID()}, ItemID: scarf.ID, Cut: release.Standard, Quality: release.WEBDL1080p, Path: ".slipstream-incoming/grab.mkv"}
	w := shelved(t, "{Shelf} - {Thing}", targets.Goals{Targets: []targets.Target{web}}, scarf)
	w.files, w.fresh = []File{held, fresh}, fresh.ID
	_, _, err := names(w, map[CellKey]Fill{{Item: scarf.ID, Edition: targets.AnyEdition, Target: web.ID}: {File: held.ID}})
	if taken, ok := errors.AsType[*NameTakenError](err); !ok || *taken != (NameTakenError{Path: "Attic/Attic - Scarf.mkv"}) {
		t.Fatalf("names of a Kept file whose template names no Target = %v, want its name taken at Attic/Attic - Scarf.mkv", err)
	}
}

// keeping is the world of shelved whose type's renaming is off, holding
// a file filling Web at held and, being placed, a Kept file downloaded
// as source.
func keeping(t *testing.T, held, source string) (w world, fills map[CellKey]Fill, fresh ID) {
	t.Helper()
	web := newTarget(t, "Web", []release.Quality{release.WEBDL1080p})
	scarf := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}}
	w = shelved(t, "{Shelf} - {Thing} - {Target}", targets.Goals{Targets: []targets.Target{web}}, scarf)
	off := false
	scheme, err := naming.SchemeOf(shelf("{Shelf} - {Thing} - {Target}").Naming(), naming.Settings{Rename: &off})
	if err != nil {
		t.Fatal(err)
	}
	w.scheme = scheme
	filling := File{ID: ID{UUID: storage.NewUUID()}, ItemID: scarf.ID, Cut: release.Standard, Quality: release.WEBDL1080p, Label: "Web", Path: held}
	placing := File{ID: ID{UUID: storage.NewUUID()}, ItemID: scarf.ID, Cut: release.Standard, Quality: release.WEBDL1080p, Path: ".slipstream-incoming/grab.mkv", SourceName: source}
	w.files, w.beside, w.fresh = []File{filling, placing}, []File{filling}, placing.ID
	return w, map[CellKey]Fill{{Item: scarf.ID, Edition: targets.AnyEdition, Target: web.ID}: {File: filling.ID}}, placing.ID
}

func TestWithRenamingOffAFileBeingPlacedKeepsItsDownloadedNameInTheWorksFolder(t *testing.T) {
	w, fills, fresh := keeping(t, "Attic, as found/scarf.mkv", "Scarf.2020.1080p.WEB-DL.mkv")
	got, _, err := names(w, fills)
	if err != nil {
		t.Fatal(err)
	}
	want := map[ID]name{w.files[0].ID: {"Web", 0, "Attic, as found/scarf.mkv"}, fresh: {"Web", 1, "Attic, as found/Scarf.2020.1080p.WEB-DL.mkv"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("names =\n%v\nwant the file beside it kept and the new one under its downloaded name in that folder, labeled\n%v", got, want)
	}
}

func TestWithRenamingOffADownloadedNameAnotherFileHasIsTaken(t *testing.T) {
	w, fills, _ := keeping(t, "Attic/scarf.2020.1080p.web-dl.mkv", "Scarf.2020.1080p.WEB-DL.mkv")
	_, _, err := names(w, fills)
	if taken, ok := errors.AsType[*NameTakenError](err); !ok || *taken != (NameTakenError{Path: "Attic/Scarf.2020.1080p.WEB-DL.mkv"}) {
		t.Fatalf("names = %v, want the downloaded name taken in Attic, whatever its case", err)
	}
}

func TestAGoneFileKeepsItsNameAndHoldsItsN(t *testing.T) {
	best := newTarget(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	scarf := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}}
	gone := File{
		ID: ID{UUID: storage.NewUUID()}, ItemID: scarf.ID, Cut: release.Standard, Quality: release.WEBDL1080p, Label: "Web", Extra: 1,
		Path: "Attic/Attic - Scarf - Web (extra 1).mkv", GoneAt: storage.Now(),
	}
	kept := File{ID: ID{UUID: storage.NewUUID()}, ItemID: scarf.ID, Cut: release.Standard, Quality: release.HDTV720p, Label: "Web", Path: "Attic/Scarf.mkv"}
	w := shelved(t, "{Shelf} - {Thing} - {Edition Tag} {Target}", targets.Goals{Targets: []targets.Target{best}}, scarf)
	w.files, w.retagged = []File{gone, kept}, map[release.CutKey]bool{release.Standard: true}
	got, _, err := names(w, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[ID]name{
		gone.ID: {"Web", 1, "Attic/Attic - Scarf - Web (extra 1).mkv"},
		kept.ID: {"Web", 2, "Attic/Attic - Scarf - Web (extra 2).mkv"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("names =\n%v\nwant the gone file unchanged though it meets Best and its cut retags, and the Kept file past its N\n%v", got, want)
	}
}
