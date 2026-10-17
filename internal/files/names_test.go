package files

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func newTarget(t *testing.T, name string, tiers ...[]release.Quality) targets.Target {
	t.Helper()
	target, err := targets.NewTarget(targets.Spec{Name: name, Tiers: tiers, Cutoff: tiers[0][0]})
	if err != nil {
		t.Fatal(err)
	}
	target.ID = targets.TargetID{UUID: storage.NewUUID()}
	return target
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
			w := world{
				tree: works.Tree{Items: []works.Item{scarf}}, goals: targets.Goals{Targets: []targets.Target{best, web, uhd}}, fresh: ids["fresh"],
				templates: naming.Templates{Folder: "{Shelf}", File: "{Shelf} - {Thing} - {Edition Tag} {Target}"},
				tokens:    map[works.ItemID]map[string]string{scarf.ID: {"Shelf": "Attic", "Thing": "Scarf"}},
			}
			for _, s := range c.files {
				w.files = append(w.files, File{ID: ids[s.id], ItemID: scarf.ID, Edition: Standard, Quality: s.quality, Label: s.label, Extra: s.extra, Path: s.path})
			}
			fills := map[CellKey]ID{}
			for target, id := range c.fills {
				fills[CellKey{Item: scarf.ID, Edition: Standard, Target: target.ID}] = ids[id]
			}
			got, err := names(w, fills)
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

func TestTwoFilesOnOneNameIsAClash(t *testing.T) {
	web := newTarget(t, "Web", []release.Quality{release.WEBDL1080p})
	scarf := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}}
	held := File{ID: ID{UUID: storage.NewUUID()}, ItemID: scarf.ID, Edition: Standard, Quality: release.WEBDL1080p, Label: "Web", Path: "Attic/Attic - Scarf.mkv"}
	fresh := File{ID: ID{UUID: storage.NewUUID()}, ItemID: scarf.ID, Edition: Standard, Quality: release.WEBDL1080p, Path: ".slipstream-incoming/grab.mkv"}
	w := world{
		tree: works.Tree{Items: []works.Item{scarf}}, goals: targets.Goals{Targets: []targets.Target{web}}, files: []File{held, fresh}, fresh: fresh.ID,
		templates: naming.Templates{Folder: "{Shelf}", File: "{Shelf} - {Thing}"},
		tokens:    map[works.ItemID]map[string]string{scarf.ID: {"Shelf": "Attic", "Thing": "Scarf"}},
	}
	_, err := names(w, map[CellKey]ID{{Item: scarf.ID, Edition: Standard, Target: web.ID}: held.ID})
	if !errors.Is(err, ErrNameClash) {
		t.Fatalf("names of a Kept file whose template names no Target = %v, want ErrNameClash", err)
	}
}
