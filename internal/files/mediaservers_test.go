package files_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// tellings is a library's files.MediaServers. It records each change in
// the Write that tells it, so a Write that fails records none.
type tellings struct{}

func (s *tellings) record(t testing.TB, lib *storage.DB[storage.Library]) {
	t.Helper()
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `CREATE TABLE test_told (
			seq INTEGER PRIMARY KEY, kind INTEGER NOT NULL, file_id BLOB NOT NULL, root_id BLOB NOT NULL,
			path TEXT NOT NULL, from_path TEXT NOT NULL, replaced_by BLOB, upgrade INTEGER NOT NULL)`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (s *tellings) Changed(ctx context.Context, tx *storage.Tx[storage.Library], cs []files.PathChange) error {
	for _, c := range cs {
		_, err := tx.ExecContext(ctx, `INSERT INTO test_told (kind, file_id, root_id, path, from_path, replaced_by, upgrade) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			c.Kind, c.File, c.Root, c.Path.String(), c.From.String(), c.ReplacedBy, c.Upgrade)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *tellings) Prepare(context.Context, []files.Rename) {}

// told is every change l's files owners told, in order.
func (l *library) told(t *testing.T) []files.PathChange {
	t.Helper()
	rows, err := l.lib.QueryContext(t.Context(), `SELECT kind, file_id, root_id, path, from_path, replaced_by, upgrade FROM test_told ORDER BY seq`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []files.PathChange
	for rows.Next() {
		var c files.PathChange
		var at, from string
		if err := rows.Scan(&c.Kind, &c.File, &c.Root, &at, &from, &c.ReplacedBy, &c.Upgrade); err != nil {
			t.Fatal(err)
		}
		c.Path = rel(t, at)
		if from != "" {
			c.From = rel(t, from)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// forget drops every change told so far.
func (l *library) forget(t *testing.T) {
	t.Helper()
	l.exec(t, `DELETE FROM test_told`)
}

func rel(t *testing.T, p string) storing.Rel {
	t.Helper()
	r, err := storing.ParseRel(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

var kinds = map[files.PathChangeKind]string{
	files.PathAdded: "added", files.PathReplaced: "replaced", files.PathRemoved: "removed", files.PathMoved: "moved", files.PathUnmoved: "unmoved",
}

// said is each of cs as a line: its kind and paths, and its pairing.
func said(cs []files.PathChange) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		line := kinds[c.Kind] + " " + c.Path.String()
		if c.Kind == files.PathMoved {
			line = kinds[c.Kind] + " " + c.From.String() + " -> " + c.Path.String()
		}
		if !c.ReplacedBy.IsZero() {
			line += " replaced"
		}
		if c.Upgrade {
			line += " upgrade"
		}
		out[i] = line
	}
	return out
}

// listing is where a server that took every change so far lists each file
// of l's root: a live or doomed file outside holding, at its move's origin
// while one is pending, else at its path.
func (l *library) listing(t *testing.T) map[files.ID]string {
	t.Helper()
	rows, err := l.lib.QueryContext(t.Context(), `
		SELECT f.id, coalesce(m.origin, f.path) FROM files_file f LEFT JOIN files_move m ON m.file_id = f.id WHERE f.gone_at IS NULL
		UNION ALL SELECT file_id, coalesce(origin, path) FROM files_doomed`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	out := map[files.ID]string{}
	for rows.Next() {
		var id files.ID
		var at string
		if err := rows.Scan(&id, &at); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(at, storing.HoldingDir+"/") {
			out[id] = at
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// served is what a server listing base lists after cs. It fails t on a
// change that server can't take: a file added twice, or moved, replaced
// or removed at a path it doesn't list the file at. A second change of
// one transition is such a change, so every file's changes add up to
// exactly one net change.
func served(t *testing.T, base map[files.ID]string, cs []files.PathChange) map[files.ID]string {
	t.Helper()
	out := maps.Clone(base)
	for i, c := range cs {
		at, ok := out[c.File]
		switch c.Kind {
		case files.PathAdded:
			if ok {
				t.Fatalf("change %d, %s: the server already lists the file at %s; changes %v", i, said(cs[i:i+1]), at, said(cs))
			}
			out[c.File] = c.Path.String()
		case files.PathMoved:
			if at != c.From.String() {
				t.Fatalf("change %d, %s: the server lists the file at %q; changes %v", i, said(cs[i:i+1]), at, said(cs))
			}
			out[c.File] = c.Path.String()
		case files.PathRemoved, files.PathReplaced, files.PathUnmoved:
			if at != c.Path.String() {
				t.Fatalf("change %d, %s: the server lists the file at %q; changes %v", i, said(cs[i:i+1]), at, said(cs))
			}
			if c.Kind == files.PathRemoved {
				delete(out, c.File)
			}
		}
	}
	return out
}

// converges checks that a server that listed base and took every change
// told since lists each file where the library holds it now.
func (l *library) converges(t *testing.T, base map[files.ID]string) {
	t.Helper()
	if got, want := served(t, base, l.told(t)), l.listing(t); !maps.Equal(got, want) {
		t.Fatalf("after the changes %v the server lists\n%v\nwant\n%v", said(l.told(t)), got, want)
	}
}

func TestAPlacedImportIsAddedAndASameNameUpgradesRemovalNamesItsReplacer(t *testing.T) {
	l := open(t)
	best := l.target(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Best", best)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	web := l.download(t, svc, tree, 0, release.WEBDL1080p, "webdl")
	if got := said(l.told(t)); len(got) != 0 {
		t.Fatalf("the import's Write told %v, want nothing before its bytes reach their name", got)
	}
	l.settle(t, svc)
	if got, want := said(l.told(t)), []string{"added " + web.Path}; !slices.Equal(got, want) {
		t.Fatalf("placing the import told %v, want %v", got, want)
	}
	base := l.listing(t)
	l.forget(t)

	bluray := l.download(t, svc, tree, 0, release.Bluray1080p, "bluray")
	l.settle(t, svc)
	told := l.told(t)
	if got, want := said(told), []string{"removed " + web.Path + " replaced", "added " + web.Path}; !slices.Equal(got, want) {
		t.Fatalf("the upgrade told %v, want %v", got, want)
	}
	if told[0].File != web.ID || told[0].ReplacedBy != bluray.ID || told[1].File != bluray.ID {
		t.Fatalf("the upgrade told %+v, want the WEB-DL removed naming the Bluray, then the Bluray added", told)
	}
	l.converges(t, base)
}

func TestAnImportWhoseBytesSitAtItsNameIsAddedInItsWrite(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	svc := l.files(files.NoGrabs{})
	name := l.judge(t, svc, tree.ID, holdingOf(tree, 0, files.HoldingOf(storage.NewUUID(), ".mkv"), 5, media.Record{}, release.WEBDL1080p)).Path
	l.put(t, name, "found")
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := l.land(ctx, tx, svc, tree.ID, holdingOf(tree, 0, name, 5, media.Record{Video: media.Video{Width: 1920, Height: 1080}}, release.WEBDL1080p))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := said(l.told(t)), []string{"added " + name}; !slices.Equal(got, want) {
		t.Fatalf("the in-place import told %v, want %v", got, want)
	}
	l.settle(t, svc)
	if got := said(l.told(t)); len(got) != 1 {
		t.Fatalf("after the queue ran the library told %v, want only the added", got)
	}
}

func TestADoomMidRenameRemovesTheFileWhereServersListIt(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	l.stock(t, tree, "w1/a.mkv", "w1/b.mkv")
	base := l.listing(t)
	l.renames(t, map[string]string{"w1/a.mkv": "w1/b.mkv", "w1/b.mkv": "w1/a.mkv"})
	disk := &recording{Disk: l.held}
	disk.before = func(from, _ storing.Rel) error {
		if strings.HasPrefix(from.String(), storing.HoldingDir+"/") {
			return errors.New("the Node refused the move")
		}
		return nil
	}
	l.runUntil(t, l.on(files.NoGrabs{}, disk), "both files reach holding", func() bool { return l.gone("w1/a.mkv", "w1/b.mkv") })
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return files.DoomIn(ctx, tx, []works.WorkID{tree.ID})
	})
	if err != nil {
		t.Fatal(err)
	}
	l.settle(t, l.files(files.NoGrabs{}))

	got := said(l.told(t))
	slices.Sort(got)
	if want := []string{"removed w1/a.mkv", "removed w1/b.mkv"}; !slices.Equal(got, want) {
		t.Fatalf("the doomed renames told %v, want each removed at its origin", got)
	}
	l.converges(t, base)
}

func TestAScanTellsEachTransitionItRecords(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "HD", hd)
	tree := l.work(t, "w1", true, &p)
	scarf, mittens := tree.Items[0].ID, tree.Items[1].ID
	svc := l.files(files.NoGrabs{})
	l.place(t, scarf, tree.ID, "Shelf w1/Scarf.mkv", release.WEBDL1080p, hd.ID)
	l.place(t, mittens, tree.ID, "Shelf w1/Mittens.mkv", release.WEBDL1080p, hd.ID)
	base := l.listing(t)
	scan := func(found []files.Found, probes map[storing.Rel]files.Probe, entries ...storing.Walked) []string {
		t.Helper()
		l.forget(t)
		l.scan(t, svc, files.Scan{Work: tree.ID, Root: l.root.ID, Share: l.compare(t, entries...)[tree.ID], Probes: probes, Found: found})
		return said(l.told(t))
	}
	scarfAt, mittensAt, extraAt := walked(t, "Shelf w1/Scarf.mkv", 1024, 1), walked(t, "Shelf w1/Mittens.mkv", 1024, 2), walked(t, "Shelf w1/Extra.mkv", 10, 3)
	var all []files.PathChange
	steps := []struct {
		what    string
		found   []files.Found
		probes  map[storing.Rel]files.Probe
		entries []storing.Walked
		want    []string
	}{
		{"a found file", []files.Found{found(scarf, release.WEBDL1080p, extraAt)}, nil, []storing.Walked{scarfAt, mittensAt, extraAt}, []string{"added Shelf w1/Extra.mkv"}},
		{"a resize, a move and a gone file", nil, map[storing.Rel]files.Probe{mittensAt.Path: {Media: read1080}}, []storing.Walked{
			walked(t, "Shelf w1/Moved/Scarf.mkv", 1024, 1), walked(t, "Shelf w1/Mittens.mkv", 2048, 2),
		}, []string{"replaced Shelf w1/Mittens.mkv", "moved Shelf w1/Scarf.mkv -> Shelf w1/Moved/Scarf.mkv", "removed Shelf w1/Extra.mkv"}},
		{"a gone file back at its path", nil, nil, []storing.Walked{walked(t, "Shelf w1/Moved/Scarf.mkv", 1024, 1), walked(t, "Shelf w1/Mittens.mkv", 2048, 2), extraAt}, []string{"added Shelf w1/Extra.mkv"}},
		{"a gone file", nil, nil, []storing.Walked{walked(t, "Shelf w1/Moved/Scarf.mkv", 1024, 1), extraAt}, []string{"removed Shelf w1/Mittens.mkv"}},
		{"a gone file back at another path", nil, nil, []storing.Walked{walked(t, "Shelf w1/Moved/Scarf.mkv", 1024, 1), extraAt, walked(t, "Shelf w1/Mittens (back).mkv", 2048, 2)}, []string{"added Shelf w1/Mittens (back).mkv"}},
	}
	for _, s := range steps {
		if got := scan(s.found, s.probes, s.entries...); !slices.Equal(got, s.want) {
			t.Fatalf("%s told %v, want %v", s.what, got, s.want)
		}
		all = append(all, l.told(t)...)
	}
	if got, want := served(t, base, all), l.listing(t); !maps.Equal(got, want) {
		t.Fatalf("after the scans the server lists %v, want %v", got, want)
	}
}

func TestIDsAtFindsTheFileWhoseBytesSitAtEachPath(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	ids := l.stock(t, tree, "w1/a.mkv", "w1/b.mkv", "w1/c.mkv")
	l.renames(t, map[string]string{"w1/a.mkv": "w1/z.mkv"})
	svc := l.files(files.NoGrabs{})
	l.markGone(t, svc, tree.ID, "w1/b.mkv")
	var got map[storing.Rel]files.ID
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = svc.IDsAt(ctx, r, l.root.ID, []storing.Rel{rel(t, "w1/a.mkv"), rel(t, "w1/b.mkv"), rel(t, "w1/z.mkv"), rel(t, "w1/none.mkv")})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[storing.Rel]files.ID{rel(t, "w1/a.mkv"): ids["w1/a.mkv"], rel(t, "w1/b.mkv"): ids["w1/b.mkv"]}
	if !maps.Equal(got, want) {
		t.Fatalf("IDsAt = %v, want the renaming file where its bytes still sit and the gone file at its path", got)
	}
}

func TestAnInPlaceUpgradeIsAddedAsOneThatARemovalFollows(t *testing.T) {
	l := open(t)
	best := l.target(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Best", best)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	web := l.download(t, svc, tree, 0, release.WEBDL1080p, "webdl")
	l.settle(t, svc)
	base := l.listing(t)
	l.forget(t)

	name := strings.TrimSuffix(web.Path, ".mkv") + ".mp4"
	l.put(t, name, "bluray")
	var landed files.Landed
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		landed, err = l.land(ctx, tx, svc, tree.ID, holdingOf(tree, 0, name, 6, media.Record{Video: media.Video{Width: 1920, Height: 1080}}, release.Bluray1080p))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := said(l.told(t)), []string{"added " + name + " upgrade"}; landed.File.Path != name || !slices.Equal(got, want) {
		t.Fatalf("the in-place upgrade at %s told %v, want %v", landed.File.Path, got, want)
	}
	l.settle(t, svc)
	if got, want := said(l.told(t)), []string{"added " + name + " upgrade", "removed " + web.Path + " replaced"}; !slices.Equal(got, want) {
		t.Fatalf("after the queue ran the upgrade told %v, want %v", got, want)
	}
	l.converges(t, base)
}
