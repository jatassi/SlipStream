package rootsapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// file records a file of size bytes at rel in root for work, and writes
// its bytes under dir, root's folder, when dir isn't empty.
func (s *server) file(t *testing.T, work, root, dir, rel string, size int) {
	t.Helper()
	if dir != "" {
		at := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(at), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(at, make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		w := works.WorkID{UUID: must(storage.ParseUUID(work))}
		items, err := works.ItemsOf(ctx, tx, []works.WorkID{w})
		if err != nil {
			return err
		}
		_, err = files.Add(ctx, tx, files.File{
			WorkID: w, ItemID: items[w][0].ID, Cut: release.Standard, RootID: roots.ID{UUID: must(storage.ParseUUID(root))},
			Path: rel, Size: int64(size), Probe: files.Probe{Width: 1920, Height: 1080}, Quality: release.WEBDL1080p,
		}, nil)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

type listedRoot struct {
	ID        string `json:"id"`
	NodeID    string `json:"nodeId"`
	Path      string `json:"path"`
	Default   bool   `json:"default"`
	Works     int64  `json:"works"`
	FileBytes int64  `json:"fileBytes"`
	Rehome    *struct {
		Checked int64 `json:"checked"`
		Total   int64 `json:"total"`
	} `json:"rehome"`
}

func (s *server) roots(t *testing.T) []listedRoot {
	t.Helper()
	code, reply := s.Call(t, http.MethodGet, "/api/v1/roots", "")
	var list struct {
		Roots []listedRoot `json:"roots"`
	}
	if err := json.Unmarshal([]byte(reply), &list); code != http.StatusOK || err != nil {
		t.Fatalf("listRoots = %d %s", code, reply)
	}
	return list.Roots
}

// intent is root's entry in self's Storage intent: its place and whether
// self adopts it.
func (s *server) intent(t *testing.T, root string) string {
	t.Helper()
	var out string
	err := s.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		in, err := roots.StorageIntent(files.RootContents{})(ctx, rd, self)
		if err != nil {
			return err
		}
		for _, r := range in.GetRoots() {
			if must(storage.UUIDFromBytes(r.GetId())).String() == root {
				out = fmt.Sprintf("%s place=%d adopt=%t", r.GetPath(), r.GetPlace(), r.GetAdopt())
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func rehome(folder, extra string) string {
	return `{"path":` + strconv.Quote(folder) + extra + `}`
}

func TestRehomeFindsEveryFile(t *testing.T) {
	s := serve(t)
	from, to := t.TempDir(), t.TempDir()
	id := s.create(t, body("fixture", from))
	work := s.add(t, "held", "")
	for _, dir := range []string{from, to} {
		for _, f := range []string{"held/held.mkv", "held/extras/held.srt"} {
			if err := os.MkdirAll(filepath.Join(dir, "held", "extras"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(f)), make([]byte, 2048), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	s.file(t, work, id, "", "held/held.mkv", 2048)
	s.file(t, work, id, "", "held/extras/held.srt", 2048)
	if got, want := s.intent(t, id), from+" place=1 adopt=true"; got != want {
		t.Fatalf("self's intent for the root = %q, want %q", got, want)
	}
	s.changed.ids = nil

	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+id, rehome(to+string(filepath.Separator), ""), http.StatusOK,
		`{"id":"id1","contentType":"fixture","nodeId":"id2","path":`+strconv.Quote(to)+`,"default":true,"state":"checking","works":1,"fileBytes":4096}`)
	if got, want := s.intent(t, id), to+" place=2 adopt=true"; got != want {
		t.Fatalf("self's intent for the re-homed root = %q, want %q", got, want)
	}
	if len(s.changed.ids) == 0 || !slices.ContainsFunc(s.changed.ids, func(ids []string) bool { return slices.Equal(ids, []string{id}) }) {
		t.Fatalf("roots.changed published %v, want the root", s.changed.ids)
	}
	if r := s.roots(t)[0]; r.Rehome != nil {
		t.Fatalf("listRoots after the re-home shows progress %+v, want none", r.Rehome)
	}
	for _, f := range []string{"held/held.mkv", "held/extras/held.srt"} {
		if _, err := os.Stat(filepath.Join(from, filepath.FromSlash(f))); err != nil {
			t.Fatalf("the old folder lost %s: %v; a re-home never touches files", f, err)
		}
	}

	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+id, `{}`, http.StatusOK,
		`{"id":"id1","contentType":"fixture","nodeId":"id2","path":`+strconv.Quote(to)+`,"default":true,"state":"checking","works":1,"fileBytes":4096}`)
	if got, want := s.intent(t, id), to+" place=3 adopt=true"; got != want {
		t.Fatalf("self's intent after re-homing onto its own folder = %q, want %q, adopted again", got, want)
	}
}

func TestRehomeWithMissingFilesAndForget(t *testing.T) {
	s := serve(t)
	from, to := t.TempDir(), t.TempDir()
	id := s.create(t, body("fixture", from))
	work := s.add(t, "held", "")
	s.file(t, work, id, to, "held/kept.mkv", 1024)
	s.file(t, work, id, "", "held/gone.mkv", 2048)
	s.file(t, work, id, to, "held/short.mkv", 100)
	if err := os.WriteFile(filepath.Join(to, "held", "short.mkv"), make([]byte, 50), 0o600); err != nil {
		t.Fatal(err)
	}
	missing := `{"status":409,"code":"rehome_missing","title":"Recorded files aren't at the new place","missing":{"count":2,"first":[` +
		`{"path":"held/gone.mkv","reason":"missing"},{"path":"held/short.mkv","reason":"size_differs"}]}}`

	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+id, rehome(to, ""), http.StatusConflict, missing)
	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+id, rehome(to, `,"forget":1`), http.StatusConflict, missing)
	if r := s.roots(t)[0]; r.Path != from || r.FileBytes != 3172 || r.Rehome != nil {
		t.Fatalf("listRoots after refused re-homes = %+v, want it at %s with every file", r, from)
	}
	if got, want := s.intent(t, id), from+" place=1 adopt=true"; got != want {
		t.Fatalf("self's intent after refused re-homes = %q, want %q", got, want)
	}

	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+id, rehome(to, `,"forget":2`), http.StatusOK,
		`{"id":"id1","contentType":"fixture","nodeId":"id2","path":`+strconv.Quote(to)+`,"default":true,"state":"checking","works":1,"fileBytes":1024}`)
	if got, want := s.intent(t, id), to+" place=2 adopt=true"; got != want {
		t.Fatalf("self's intent = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(to, "held", "short.mkv")); err != nil {
		t.Fatalf("forgetting a file removed its bytes: %v", err)
	}
}

func TestRehomeShowsItsProgressOnTheRow(t *testing.T) {
	s := serve(t)
	from, to := t.TempDir(), t.TempDir()
	id := s.create(t, body("fixture", from))
	work := s.add(t, "held", "")
	for i := range 3 {
		s.file(t, work, id, to, fmt.Sprintf("held/part%d.mkv", i), 10)
	}
	listed := make(chan []listedRoot, 1)
	s.verifying = func() { listed <- s.roots(t) }
	if code, reply := s.Call(t, http.MethodPatch, "/api/v1/roots/"+id, rehome(to, "")); code != http.StatusOK {
		t.Fatalf("rehomeRoot = %d %s", code, reply)
	}
	during := <-listed
	if p := during[0].Rehome; p == nil || p.Checked != 0 || p.Total != 3 {
		t.Fatalf("listRoots while the re-home checks shows %+v, want 0 of 3 checked", p)
	}
	if p := s.roots(t)[0].Rehome; p != nil {
		t.Fatalf("listRoots after the re-home shows %+v, want no progress", p)
	}
}

func TestAFileRecordedWhileARehomeChecksIsCheckedToo(t *testing.T) {
	s := serve(t)
	from, to := t.TempDir(), t.TempDir()
	id := s.create(t, body("fixture", from))
	work := s.add(t, "held", "")
	s.file(t, work, id, to, "held/first.mkv", 10)
	calls := 0
	s.verifying = func() {
		calls++
		if calls == 1 {
			s.file(t, work, id, "", "held/late.mkv", 10)
		}
	}
	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+id, rehome(to, ""), http.StatusConflict,
		`{"status":409,"code":"rehome_missing","title":"Recorded files aren't at the new place","missing":{"count":1,"first":[{"path":"held/late.mkv","reason":"missing"}]}}`)
	if calls != 2 {
		t.Fatalf("the re-home checked in %d calls, want the late file checked in a second", calls)
	}
}

// moving records a move of the file at rel whose bytes the disk queue
// last recorded at sits, and answers the file's holding path.
func (s *server) moving(t *testing.T, rel, sits string) string {
	t.Helper()
	var holding string
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var id storage.UUID
		if err := tx.QueryRowContext(ctx, `SELECT id FROM files_file WHERE path = ?`, rel).Scan(&id); err != nil {
			return fmt.Errorf("file at %s: %w", rel, err)
		}
		holding = files.HoldingOf(id, path.Ext(rel))
		if sits == "" {
			sits = holding
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO files_move (file_id, sits) VALUES (?, ?)`, id, sits)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return holding
}

func TestRehomeFindsFilesTheDiskQueueIsMoving(t *testing.T) {
	s := serve(t)
	from, to := t.TempDir(), t.TempDir()
	id := s.create(t, body("fixture", from))
	work := s.add(t, "held", "")
	s.file(t, work, id, to, "held/named.mkv", 10)
	s.moving(t, "held/named.mkv", "")
	s.file(t, work, id, "", "held/renamed.mkv", 20)
	holding := s.moving(t, "held/renamed.mkv", "held/old.mkv")
	if err := os.MkdirAll(filepath.Join(to, filepath.Dir(filepath.FromSlash(holding))), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(to, filepath.FromSlash(holding)), make([]byte, 20), 0o600); err != nil {
		t.Fatal(err)
	}

	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+id, rehome(to, ""), http.StatusOK,
		`{"id":"id1","contentType":"fixture","nodeId":"id2","path":`+strconv.Quote(to)+`,"default":true,"state":"checking","works":1,"fileBytes":30}`)
}

func TestRehomeRefusals(t *testing.T) {
	s := serve(t)
	dir := t.TempDir()
	id := s.create(t, body("fixture", dir))
	other := t.TempDir()
	s.create(t, body("other", other))
	unusable := func(reason, extra string) string {
		return `{"status":422,"code":"root_unusable","title":"The folder can't be a root","refusal":{"reason":"` + reason + `"` + extra + `}}`
	}
	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+storage.NewUUID().String(), rehome(dir, ""), http.StatusNotFound,
		`{"status":404,"code":"root_not_found","title":"No root has that id"}`)
	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+id, rehome(filepath.Join(dir, "gone"), ""), http.StatusUnprocessableEntity, unusable("not_found", ""))
	code, reply := s.Call(t, http.MethodPatch, "/api/v1/roots/"+id, rehome(other, ""))
	if code != http.StatusUnprocessableEntity || !strings.HasPrefix(reply, strings.TrimSuffix(unusable("inside_root", ""), `"}}`)) {
		t.Errorf("rehomeRoot onto another root = %d %s, want inside_root", code, reply)
	}
	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+id, `{"nodeId":"`+away.String()+`"}`, http.StatusUnprocessableEntity, unusable("node_offline", ""))
	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+id, `{"nodeId":"`+bare.String()+`"}`, http.StatusConflict,
		`{"status":409,"code":"node_not_storage","title":"The Node doesn't hold Storage"}`)
	if r := s.roots(t)[0]; r.Path != dir || r.NodeID != self.String() {
		t.Fatalf("listRoots after refused re-homes = %+v, want it where it was", r)
	}
}

// grab records a Grab of work at stage, imported into root at rel when
// stage is imported.
func (s *server) grab(t *testing.T, work, stage, root, rel string) {
	t.Helper()
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		home := storage.NewUUID()
		if _, err := tx.ExecContext(ctx, `INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), ?, 0)`, home, home.String()); err != nil {
			return err
		}
		indexer := storage.NewUUID()
		if _, err := tx.ExecContext(ctx, `INSERT INTO indexers_indexer (id, name, protocol, url, api_key_sealed, home_node_id, caps, created_at)
			VALUES (?, ?, 'torznab', 'https://site.test/api', NULL, ?, '{}', 0)`, indexer, indexer.String(), home); err != nil {
			return err
		}
		var rootID, path any
		if stage == "imported" {
			rootID, path = must(storage.ParseUUID(root)), rel
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id,
			release_title, release_size, release_link, stage, info_hash, metainfo, file_index, probe, root_id, path, created_at, updated_at)
			SELECT ?, work_id, id, 'any', 'search', ?, ?, 'The.Held.1080p.WEB', 4096, 'https://site.test/dl/1', ?,
				randomblob(20), x'00', 0, '{}', ?, ?, 0, 0
			FROM works_item WHERE work_id = ? LIMIT 1`,
			storage.NewUUID(), indexer, home, stage, rootID, path, must(storage.ParseUUID(work)))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteRefusesFilesAndGrabs(t *testing.T) {
	s := serve(t)
	dir := t.TempDir()
	id := s.create(t, body("fixture", dir))
	held := s.add(t, "held", "")
	s.file(t, held, id, dir, "held/held.mkv", 2048)
	s.grab(t, held, "imported", id, "held/held.mkv")
	s.grab(t, s.add(t, "fetching", ""), "downloading", "", "")
	inUse := func(files, grabs int) string {
		return fmt.Sprintf(`{"status":409,"code":"root_in_use","title":"The root holds files or has a Grab in flight","use":{"files":%d,"grabs":%d}}`, files, grabs)
	}
	s.Expect(t, http.MethodDelete, "/api/v1/roots/"+id, "", http.StatusConflict, inUse(1, 1))

	empty := t.TempDir()
	s.Expect(t, http.MethodPatch, "/api/v1/roots/"+id, rehome(empty, `,"forget":1`), http.StatusOK,
		`{"id":"id1","contentType":"fixture","nodeId":"id2","path":`+strconv.Quote(empty)+`,"default":true,"state":"checking","works":2,"fileBytes":0}`)
	s.Expect(t, http.MethodDelete, "/api/v1/roots/"+id, "", http.StatusConflict, inUse(0, 1))
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `UPDATE grabs_grab SET stage = 'failed', failed_stage = 'downloading', failure_code = 'download_failed',
			failure_message = 'gone' WHERE stage = 'downloading'`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	s.changed.ids = nil
	if code, reply := s.Call(t, http.MethodDelete, "/api/v1/roots/"+id, ""); code != http.StatusNoContent {
		t.Fatalf("deleteRoot of a root whose imported Grab's file was forgotten = %d %s, want 204", code, reply)
	}
	if got := fmt.Sprint(s.changed.ids); got != "[["+id+"]]" {
		t.Fatalf("roots.changed published %s, want the deleted root", got)
	}
	var released int
	err = s.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		return rd.QueryRowContext(ctx, `SELECT count(*) FROM grabs_grab WHERE stage = 'imported' AND root_id IS NULL AND path IS NOT NULL`).Scan(&released)
	})
	if err != nil || released != 1 {
		t.Fatalf("the imported Grab kept its history without its root %d times (%v), want once", released, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "held", "held.mkv")); err != nil {
		t.Fatalf("deleting the root removed a file from its old folder: %v", err)
	}
}

func TestDeleteMovesWorksToAnotherRootOfTheirType(t *testing.T) {
	s := serve(t)
	first := s.create(t, body("fixture", t.TempDir()))
	second := s.create(t, body("fixture", t.TempDir()))
	other := s.create(t, body("other", t.TempDir()))
	s.add(t, "w1", `,"rootId":"`+second+`"`)
	s.add(t, "w2", `,"rootId":"`+second+`"`)

	s.Expect(t, http.MethodDelete, "/api/v1/roots/"+first, "", http.StatusConflict,
		`{"status":409,"code":"root_is_default","title":"The root is its Content Type's default while another root exists"}`)
	s.Expect(t, http.MethodDelete, "/api/v1/roots/"+second, "", http.StatusUnprocessableEntity,
		`{"status":422,"code":"works_to_required","title":"The root's Works need another root of their Content Type to go to"}`)
	s.Expect(t, http.MethodDelete, "/api/v1/roots/"+second+"?worksTo="+second, "", http.StatusUnprocessableEntity,
		`{"status":422,"code":"works_to_required","title":"The root's Works need another root of their Content Type to go to"}`)
	s.Expect(t, http.MethodDelete, "/api/v1/roots/"+second+"?worksTo="+other, "", http.StatusUnprocessableEntity,
		`{"status":422,"code":"works_to_wrong_type","title":"The root's Works can go only to a root of their Content Type"}`)
	s.Expect(t, http.MethodDelete, "/api/v1/roots/"+second+"?worksTo="+storage.NewUUID().String(), "", http.StatusNotFound,
		`{"status":404,"code":"root_not_found","title":"No root has that id"}`)
	if got, want := s.works(t), first+" default=true works=0\n"+second+" default=false works=2\n"+other+" default=true works=0\n"; got != want {
		t.Fatalf("listRoots after refused deletes =\n%swant\n%s", got, want)
	}

	s.changed.ids = nil
	if code, reply := s.Call(t, http.MethodDelete, "/api/v1/roots/"+second+"?worksTo="+first, ""); code != http.StatusNoContent || reply != "" {
		t.Fatalf("deleteRoot with worksTo = %d %s, want 204", code, reply)
	}
	if got, want := s.works(t), first+" default=true works=2\n"+other+" default=true works=0\n"; got != want {
		t.Fatalf("listRoots after the delete =\n%swant\n%s", got, want)
	}
	if got := fmt.Sprint(s.changed.ids); got != "[["+second+" "+first+"]]" {
		t.Fatalf("roots.changed published %s, want the deleted root and the one its Works went to", got)
	}
	s.Expect(t, http.MethodDelete, "/api/v1/roots/"+second, "", http.StatusNotFound,
		`{"status":404,"code":"root_not_found","title":"No root has that id"}`)
}

func TestDeletingTheLastRootLeavesItsWorksRootless(t *testing.T) {
	s := serve(t)
	dir := t.TempDir()
	only := s.create(t, body("fixture", dir))
	s.add(t, "w1", "")
	s.add(t, "w2", "")
	if code, reply := s.Call(t, http.MethodDelete, "/api/v1/roots/"+only, ""); code != http.StatusNoContent {
		t.Fatalf("deleteRoot of the type's last root = %d %s, want 204", code, reply)
	}
	if got := s.works(t); got != "" {
		t.Fatalf("listRoots after deleting the last root =\n%swant none", got)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("deleting the root removed its folder: %v", err)
	}
	next := s.create(t, body("fixture", t.TempDir()))
	if got, want := s.works(t), next+" default=true works=2\n"; got != want {
		t.Fatalf("listRoots after a new root =\n%swant\n%s, the rootless Works homed", got, want)
	}
}
