package storing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1/storagev1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

var node = role.NodeID{UUID: storage.NewUUID()}

func remoteOf(s *Store) Remote {
	return func(role.NodeID) storagev1connect.StorageServiceHandler { return s }
}

func foldersOf(s *Store) Folders {
	return func(role.NodeID) storagev1connect.FolderServiceHandler { return s }
}

func folderErr(t *testing.T, err error) *FolderError {
	t.Helper()
	fe, ok := errors.AsType[*FolderError](err)
	if !ok {
		t.Fatalf("error = %v, want a *FolderError", err)
	}
	return fe
}

func notReady(t *testing.T, err error) *NotReady {
	t.Helper()
	nr, ok := errors.AsType[*NotReady](err)
	if !ok {
		t.Fatalf("error = %v, want a *NotReady", err)
	}
	return nr
}

func TestBrowseTypesEveryError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDisk(t)
		remote := foldersOf(newStore(d))
		base := t.TempDir()
		denied, hung, broken := dir(t, base, "denied"), dir(t, base, "hung"), dir(t, base, "broken")
		d.flag(d.denied, denied)
		d.flag(d.broken, broken)
		release := d.hang(hung)
		defer release()
		for _, c := range []struct {
			path    string
			kind    FolderErrorKind
			text    string
			account string
		}{
			{filepath.Join(base, "gone"), FolderNotFound, "gone", ""},
			{write(t, filepath.Join(base, "notes.txt"), "x"), FolderNotAFolder, "not a folder", ""},
			{denied, FolderDenied, "permission denied", "media"},
			{hung, FolderUnresponsive, "hasn't answered within 10s", ""},
			{broken, FolderIO, "input/output error", ""},
		} {
			_, err := remote.Browse(t.Context(), node, c.path, false)
			fe := folderErr(t, err)
			if fe.Kind != c.kind || !strings.Contains(fe.Text, c.text) || fe.Account != c.account {
				t.Errorf("Browse %s = %+v, want %v with %q and account %q", c.path, fe, c.kind, c.text, c.account)
			}
		}
	})
}

func TestBrowseShowsLinksAndUnreadableFolders(t *testing.T) {
	d := newTestDisk(t)
	base := t.TempDir()
	d.volumes[base] = Volume{ID: "fs-1", Name: "Media", Mount: base, Kind: Local, Total: 100, Free: 40}
	for _, name := range []string{"Season 10", "Season 2", "season 1", ".cache", "@eaDir", "Locked"} {
		dir(t, base, name)
	}
	write(t, filepath.Join(base, "notes.txt"), "x")
	d.flag(d.unreadable, filepath.Join(base, "Locked"))
	var want []Folder
	wantHidden := 2
	if nativeStyle == Windows {
		want, wantHidden = append(want, Folder{".cache", filepath.Join(base, ".cache"), true}), 1
	}
	want = append(want, []Folder{
		{"Locked", filepath.Join(base, "Locked"), false},
		{"season 1", filepath.Join(base, "season 1"), true},
		{"Season 2", filepath.Join(base, "Season 2"), true},
		{"Season 10", filepath.Join(base, "Season 10"), true},
	}...)
	if err := os.Symlink(dir(t, t.TempDir(), "Elsewhere"), filepath.Join(base, "Shortcut")); err == nil {
		want = append(want, Folder{"Shortcut", filepath.Join(base, "Shortcut"), true})
	} else {
		t.Logf("no symlink on this runner: %v", err)
	}
	remote := foldersOf(newStore(d))

	l, err := remote.Browse(t.Context(), node, base, false)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(l.Folders, want) {
		t.Errorf("folders = %v, want %v", l.Folders, want)
	}
	if l.Hidden != wantHidden {
		t.Errorf("hidden = %d, want %d", l.Hidden, wantHidden)
	}
	if l.Path != base || l.Volume == nil || l.Volume.ID != "fs-1" || l.Volume.Kind != Local {
		t.Errorf("listing = path %q volume %+v, want %q on fs-1", l.Path, l.Volume, base)
	}
	if parent, _ := styleHere.Parent(base); l.Parent != parent {
		t.Errorf("parent = %q, want %q", l.Parent, parent)
	}
	if segs := l.Segments; len(segs) < 2 || segs[len(segs)-1].Path != base || segs[len(segs)-1].Name != filepath.Base(base) {
		t.Errorf("segments = %v, want them to end at %s", segs, base)
	}

	shown, err := remote.Browse(t.Context(), node, base, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(shown.Folders) != len(want)+wantHidden || shown.Hidden != 0 {
		t.Errorf("with hidden shown: %d folders and %d hidden, want %d and 0", len(shown.Folders), shown.Hidden, len(want)+wantHidden)
	}
}

func TestBrowsePlacesListTheRootsFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDisk(t)
		d.host.Service = true
		d.places = []Place{{Name: "C:", Path: `C:\`, Kind: Local, Free: 5, Total: 9}, {Name: `\\nas\media`, Path: `\\nas\media\`, Kind: Network}}
		id := storage.NewUUID()
		folder := marked(t, id)
		d.volumes[folder] = Volume{ID: "fs-1", Kind: Removable, Total: 100, Free: 40}
		s := newStore(d)
		running(t, s, intentOf(rootAt(id, folder, false))).next(t)

		l, err := foldersOf(s).Browse(t.Context(), node, "", false)
		if err != nil {
			t.Fatal(err)
		}
		want := append([]Place{{Name: folder, Path: folder, Kind: Removable, Free: 40, Total: 100, Root: id}}, d.places...)
		if !slices.Equal(l.Places, want) || !l.Service {
			t.Errorf("places = %+v service %v, want %+v on a service", l.Places, l.Service, want)
		}
	})
}

// listing is a Disk that only lists, for path rules this machine's
// folders can't play.
type listing struct {
	Disk
	style   PathStyle
	folders map[string][]string
}

type folderInfo struct{}

func (folderInfo) Dir() bool      { return true }
func (folderInfo) Size() int64    { return 0 }
func (folderInfo) Same(Info) bool { return false }

func (l listing) Host() Host { return Host{Style: l.style} }

func (l listing) Stat(_ context.Context, p string) (Info, error) {
	if _, ok := l.folders[p]; !ok {
		return nil, &fs.PathError{Op: "stat", Path: p, Err: fs.ErrNotExist}
	}
	return folderInfo{}, nil
}

func (l listing) ReadDir(_ context.Context, p string) ([]Entry, error) {
	var es []Entry
	for _, name := range l.folders[p] {
		es = append(es, Entry{Name: name, Dir: true, Hidden: hidden(l.style, name, 0), Readable: true})
	}
	return es, nil
}

func TestSuggestKeepsTheTypedSeparators(t *testing.T) {
	many := make([]string, 0, 30)
	for i := range 30 {
		many = append(many, fmt.Sprintf("Show %d", i))
	}
	windows := foldersOf(NewStore(Config{Disk: listing{style: Windows, folders: map[string][]string{
		`D:\`:       {"Media", "meetings", "Maps", "$RECYCLE.BIN"},
		`D:\Media`:  {"Movies", "Mixes", "TV"},
		`D:\Shows`:  many,
		`\\nas\tv\`: {"Anime", "Archive"},
	}}, Timings: Timings{Stat: time.Second}}))
	unix := foldersOf(NewStore(Config{Disk: listing{style: Unix, folders: map[string][]string{
		"/srv": {"media", "Media2", ".snapshots"},
	}}, Timings: Timings{Stat: time.Second}}))
	for _, c := range []struct {
		remote Folders
		typed  string
		want   []string
	}{
		{windows, `D:/Me`, []string{"D:/Media", "D:/meetings"}},
		{windows, `d:\me`, []string{`d:\Media`, `d:\meetings`}},
		{windows, `D:\Media/m`, []string{`D:\Media/Mixes`, `D:\Media/Movies`}},
		{windows, `\\nas\tv\A`, []string{`\\nas\tv\Anime`, `\\nas\tv\Archive`}},
		{windows, `D:\Gone\x`, nil},
		{windows, `D:`, nil},
		{windows, `D:/Shows/Show `, append(append([]string{}, prefixed("D:/Shows/", many[:2])...), prefixed("D:/Shows/", natural(many)[2:25])...)},
		{unix, "/srv/me", []string{"/srv/media", "/srv/Media2"}},
		{unix, "/srv/.s", []string{"/srv/.snapshots"}},
	} {
		got, err := c.remote.Suggest(t.Context(), node, c.typed)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("Suggest(%q) = %q, want %q", c.typed, got, c.want)
		}
	}
}

func natural(names []string) []string {
	out := slices.Clone(names)
	slices.SortFunc(out, naturalOrder)
	return out
}

func prefixed(prefix string, names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, prefix+n)
	}
	return out
}

func TestMakeFolderMakesOneNamedFolder(t *testing.T) {
	d := newTestDisk(t)
	remote := foldersOf(newStore(d))
	parent := t.TempDir()

	got, err := remote.MakeFolder(t.Context(), node, parent, "Movies")
	if want := filepath.Join(parent, "Movies"); err != nil || got != want {
		t.Fatalf("MakeFolder = %q, %v; want %q", got, err, want)
	}
	if fi, err := os.Stat(got); err != nil || !fi.IsDir() {
		t.Fatalf("no folder at %s: %v", got, err)
	}
	if _, err := remote.MakeFolder(t.Context(), node, parent, "Movies"); !errors.Is(err, fs.ErrExist) {
		t.Errorf("making it again = %v, want it exists", err)
	}
	for _, name := range []string{"", "..", "a/b"} {
		if _, err := remote.MakeFolder(t.Context(), node, parent, name); !errors.Is(err, ErrInvalid) {
			t.Errorf("MakeFolder(%q) = %v, want invalid", name, err)
		}
	}
	if _, err := remote.Browse(t.Context(), node, "Movies", false); !errors.Is(err, ErrInvalid) {
		t.Errorf("Browse of a relative path = %v, want invalid", err)
	}
	_, err = remote.MakeFolder(t.Context(), node, filepath.Join(parent, "gone"), "Movies")
	if fe := folderErr(t, err); fe.Kind != FolderNotFound {
		t.Errorf("MakeFolder in a missing parent = %v, want not found", fe.Kind)
	}
}

func TestCheckReasons(t *testing.T) {
	d := newTestDisk(t)
	downloads := dir(t, t.TempDir(), "downloads")
	d.host.Downloads = dir(t, downloads, "incoming")
	base := t.TempDir()
	d.volumes[base] = Volume{ID: "fs-1", Kind: Local, Total: 100, Free: 40}
	movies, tv := storage.NewUUID(), storage.NewUUID()
	moviesAt := dir(t, base, "movies")
	readOnly := dir(t, base, "read-only")
	d.flag(d.readOnly, readOnly)
	others := []Other{{movies, moviesAt}}
	remote := foldersOf(newStore(d))
	for _, c := range []struct {
		name   string
		path   string
		reason Reason
		root   storage.UUID
	}{
		{"relative", "media/tv", ReasonNotAbsolute, storage.UUID{}},
		{"missing", filepath.Join(base, "gone"), ReasonNotFound, storage.UUID{}},
		{"a file", write(t, filepath.Join(base, "notes.txt"), "x"), ReasonNotAFolder, storage.UUID{}},
		{"inside a root", dir(t, moviesAt, "new"), ReasonInsideRoot, movies},
		{"a root itself", moviesAt, ReasonInsideRoot, movies},
		{"containing a root", base, ReasonContainsRoot, movies},
		{"inside data", dir(t, d.host.Data, "x"), ReasonInsideData, storage.UUID{}},
		{"containing downloads", downloads, ReasonContainsDownloads, storage.UUID{}},
		{"read-only", readOnly, ReasonUnwritable, storage.UUID{}},
	} {
		got, err := remote.Check(t.Context(), node, c.path, PurposeRoot, others)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.Refusal == nil || got.Refusal.Reason != c.reason || got.Refusal.Root != c.root {
			t.Errorf("%s: refusal = %+v, want %v naming %v", c.name, got.Refusal, c.reason, c.root)
		}
	}

	tvAt := dir(t, base, "tv")
	write(t, filepath.Join(tvAt, MarkerName), tv.String())
	got, err := remote.Check(t.Context(), node, tvAt+string(filepath.Separator), PurposeRoot, others)
	if err != nil {
		t.Fatal(err)
	}
	if got.Refusal != nil || got.Path != tvAt || got.Marker != tv || got.Volume == nil || got.Volume.ID != "fs-1" {
		t.Errorf("Check of a usable folder = %+v, want usable at %s, marker %s, on fs-1", got, tvAt, tv)
	}
	if es, _ := os.ReadDir(tvAt); len(es) != 1 {
		t.Errorf("Check left %d entries in the folder, want only the marker", len(es))
	}
}

func TestCheckForDownloads(t *testing.T) {
	d := newTestDisk(t)
	base := t.TempDir()
	d.volumes[base] = Volume{ID: "fs-1", Kind: Local}
	share := dir(t, base, "share")
	d.volumes[share] = Volume{ID: "nas", Kind: Network}
	held := storage.NewUUID()
	heldAt := marked(t, held)
	d.volumes[heldAt] = Volume{ID: "fs-2", Kind: Local}
	readOnly := dir(t, base, "read-only")
	d.flag(d.readOnly, readOnly)
	s := newStore(d)
	running(t, s, intentOf(rootAt(held, heldAt, false))).next(t)
	sent := storage.NewUUID()
	others := []Other{{sent, dir(t, base, "sent")}}
	folders := foldersOf(s)
	for _, c := range []struct {
		name   string
		path   string
		reason Reason
		root   storage.UUID
	}{
		{"inside a root the Node holds", dir(t, heldAt, "incoming"), ReasonInsideRoot, held},
		{"containing a root the Node holds", filepath.Dir(heldAt), ReasonContainsRoot, held},
		{"on a network share", dir(t, share, "torrents"), ReasonNetworkShare, storage.UUID{}},
		{"read-only", readOnly, ReasonUnwritable, storage.UUID{}},
	} {
		got, err := folders.Check(t.Context(), node, c.path, PurposeDownloads, others)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.Refusal == nil || got.Refusal.Reason != c.reason || got.Refusal.Root != c.root {
			t.Errorf("%s: refusal = %+v, want %v naming %v", c.name, got.Refusal, c.reason, c.root)
		}
	}
	for name, path := range map[string]string{
		"inside the data folder":      dir(t, d.host.Data, "downloads"),
		"the downloads folder":        d.host.Downloads,
		"a root the Controller named": others[0].Path,
	} {
		got, err := folders.Check(t.Context(), node, path, PurposeDownloads, others)
		if err != nil || got.Refusal != nil {
			t.Errorf("%s: Check = %+v, %v, want usable", name, got, err)
		}
	}
}

func TestVerifyFindsMissingFilesInBatches(t *testing.T) {
	d := newTestDisk(t)
	folder := t.TempDir()
	files := make([]Recorded, 0, 1201)
	for i := range 1201 {
		rel := fmt.Sprintf("Show/S01E%04d.mkv", i)
		files = append(files, Recorded{Path: mustRel(t, rel), Size: 4})
		if i != 7 && i != 1100 {
			write(t, filepath.Join(folder, filepath.FromSlash(rel)), "film")
		}
	}
	write(t, filepath.Join(folder, "Show", "S01E0600.mkv"), "a longer film")
	s := newStore(d)
	var progress []int
	missing, err := remoteOf(s).Verify(t.Context(), node, folder, files, func(n int) { progress = append(progress, n) })
	if err != nil {
		t.Fatal(err)
	}
	want := []MissingFile{{7, FileMissing}, {600, SizeDiffers}, {1100, FileMissing}}
	if !slices.Equal(missing, want) {
		t.Errorf("missing = %v, want %v", missing, want)
	}
	if !slices.Equal(progress, []int{500, 1000, 1201}) {
		t.Errorf("progress = %v, want [500 1000 1201]", progress)
	}
	req := &storagev1.VerifyRequest{Path: folder}
	for range 501 {
		req.Files = append(req.Files, &storagev1.Recorded{Path: "a.mkv"})
	}
	if _, err := s.Verify(t.Context(), req); rpc.CodeOf(err) != rpc.CodeInvalidArgument {
		t.Errorf("a batch of 501 = %v, want invalid argument", err)
	}
}

type placing struct {
	disk   *testDisk
	store  *Store
	holder *holder
	remote Remote
	at     At
	folder string
	hash   bencode.InfoHash
}

func newPlacing(t *testing.T, draining bool) placing {
	t.Helper()
	d := newTestDisk(t)
	id := storage.NewUUID()
	p := placing{disk: d, at: At{Node: node, Root: id}, folder: marked(t, id), hash: bencode.InfoHash{1}}
	p.store = NewStore(Config{Disk: d, Payloads: payloads{p.hash: write(t, filepath.Join(t.TempDir(), "heat.mkv"), "film")}, Timings: Timings{Stat: 10 * time.Second}})
	p.holder = running(t, p.store, intentOf(rootAt(id, p.folder, false)))
	p.holder.next(t)
	if draining {
		p.holder.send(role.Snapshot[*storagev1.Intent]{Intent: intentOf(rootAt(id, p.folder, false)), Standing: role.Live, Draining: true})
		p.holder.next(t)
	}
	p.remote = remoteOf(p.store)
	return p
}

func (p placing) payload() Payload { return Payload{InfoHash: p.hash} }

func TestPlaceConvergesOnRepeat(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		for _, links := range []bool{true, false} {
			p := newPlacing(t, false)
			p.disk.noLinks = !links
			to := mustRel(t, HoldingDir+"/0190.mkv")
			for range 2 {
				size, err := p.remote.Place(t.Context(), p.at, p.payload(), to)
				if err != nil || size != 4 {
					t.Fatalf("links %v: Place = %d, %v; want 4 bytes", links, size, err)
				}
			}
			placed := filepath.Join(p.folder, HoldingDir, "0190.mkv")
			if b, err := os.ReadFile(placed); err != nil || string(b) != "film" {
				t.Errorf("links %v: the placed file holds %q, %v", links, b, err)
			}
			if _, err := os.Stat(placed + ".part"); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("links %v: a .part file stayed: %v", links, err)
			}
		}

		p := newPlacing(t, false)
		write(t, filepath.Join(p.folder, HoldingDir, "taken.mkv"), "another film")
		if _, err := p.remote.Place(t.Context(), p.at, p.payload(), mustRel(t, HoldingDir+"/taken.mkv")); !errors.Is(err, ErrOccupied) {
			t.Errorf("Place onto another file = %v, want ErrOccupied", err)
		}
	})
}

func TestMoveRemoveAndListActOnTheRoot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPlacing(t, false)
		holding := mustRel(t, HoldingDir+"/0190.mkv")
		if _, err := p.remote.Place(t.Context(), p.at, p.payload(), holding); err != nil {
			t.Fatal(err)
		}
		names, err := p.remote.List(t.Context(), p.at, Holding)
		if err != nil || !slices.Equal(names, []string{"0190.mkv"}) {
			t.Errorf("List holding = %v, %v; want [0190.mkv]", names, err)
		}
		if names, err := p.remote.List(t.Context(), p.at, Staging); err != nil || len(names) != 0 {
			t.Errorf("List of a staging folder never made = %v, %v; want none", names, err)
		}
		final := mustRel(t, "Heat (1995)/Heat (1995).mkv")
		for range 2 {
			if size, err := p.remote.Move(t.Context(), p.at, holding, final); err != nil || size != 4 {
				t.Fatalf("Move = %d, %v; want 4 bytes", size, err)
			}
		}
		if err := p.remote.Remove(t.Context(), p.at, final); err != nil {
			t.Fatal(err)
		}
		if err := p.remote.Remove(t.Context(), p.at, final); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("removing it again = %v, want fs.ErrNotExist", err)
		}
	})
}

func TestAPlacementIntoAFolderThatStoppedTakingWritesIsNotReady(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPlacing(t, false)
		p.disk.noLinks = true
		holding := dir(t, p.folder, HoldingDir)
		p.disk.mu.Lock()
		p.disk.readOnly[p.folder], p.disk.readOnly[holding] = true, true
		p.disk.mu.Unlock()
		_, err := p.remote.Place(t.Context(), p.at, p.payload(), mustRel(t, HoldingDir+"/0190.mkv"))
		if nr := notReady(t, err); nr.State != Unwritable {
			t.Errorf("Place into a read-only root = %+v, want unwritable", nr)
		}
	})
}

func TestPlaceRefusesWhileDraining(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPlacing(t, true)
		_, err := p.remote.Place(t.Context(), p.at, p.payload(), mustRel(t, HoldingDir+"/0190.mkv"))
		if nr := notReady(t, err); !nr.Draining {
			t.Errorf("Place while draining = %+v, want draining", nr)
		}
		if _, err := p.store.Local(t.Context(), p.at.Root, mustRel(t, StagingDir+"/0190.part")); err == nil {
			t.Error("staging for a Transfer while draining was allowed")
		}
		write(t, filepath.Join(p.folder, HoldingDir, "0180.mkv"), "film")
		if _, err := p.remote.Move(t.Context(), p.at, mustRel(t, HoldingDir+"/0180.mkv"), mustRel(t, "Heat/Heat.mkv")); err != nil {
			t.Errorf("Move while draining = %v, want the existing file moved", err)
		}
	})
}

func TestAWriteReReadsTheMarker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPlacing(t, false)
		if err := os.Remove(filepath.Join(p.folder, MarkerName)); err != nil {
			t.Fatal(err)
		}
		_, err := p.remote.Place(t.Context(), p.at, p.payload(), mustRel(t, HoldingDir+"/0190.mkv"))
		if nr := notReady(t, err); nr.State != Unmounted {
			t.Errorf("Place into a folder that lost its marker = %+v, want unmounted", nr)
		}
		if _, err := os.Stat(filepath.Join(p.folder, HoldingDir)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("bytes landed in a folder without its marker: %v", err)
		}
		checkStates(t, p.holder.next(t), map[storage.UUID]RootState{p.at.Root: Unmounted})

		_, err = p.remote.Place(t.Context(), At{Node: node, Root: storage.NewUUID()}, p.payload(), mustRel(t, HoldingDir+"/0190.mkv"))
		if nr := notReady(t, err); !nr.Unknown {
			t.Errorf("Place into a root not in the intent = %+v, want unknown", nr)
		}
	})
}
