package historyapi_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/historyapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notificationstest"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

// noted is an owner's facts about whatever On says.
type noted struct {
	Name string      `json:"name"`
	On   entry.About `json:"-"`
}

func (f noted) About() entry.About { return f.On }

func notedAs(kind entry.Kind, label string, group entry.Group, tally works.Noun, tone entry.Tone) entry.Row[noted] {
	return entry.Row[noted]{
		Spec: entry.Spec{Kind: kind, Label: label, Group: group, BellByDefault: true, Tally: tally},
		Render: func(f noted, _ entry.Context) entry.Message {
			return entry.Message{
				Title: label + " " + f.Name, Tone: tone, Lines: []string{"It happened."}, Link: entry.Link{Path: "/noted"},
				Detail: entry.Detail{
					Files:  []entry.DetailFile{{Items: []string{"Part 1"}, Quality: "WEBDL-1080p", PlacedBy: "link", Replaced: "HDTV-720p", ReplacedLabel: "The.Attic.720p"}},
					Stages: []entry.StageAt{{Stage: "fetching", At: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)}},
				},
			}
		},
	}
}

var (
	grabbedRow  = notedAs("grabbed", "Grabbed", entry.Grabs, works.Noun{One: "Grab", Many: "Grabs"}, entry.Info)
	failedRow   = notedAs("grab_failed", "Grab failed", entry.Grabs, works.Noun{}, entry.Fail)
	importedRow = notedAs("imported", "Imported", entry.Imports, works.Noun{One: "Import", Many: "Imports"}, entry.Good)
	deletedRow  = notedAs("works_deleted", "Works deleted", entry.Library, works.Noun{}, entry.Info)
)

type node struct {
	h  *history.Service
	w  *works.Works
	hl *health.Health
	m  artwork.Minter
}

func (n node) History() (*history.Service, bool) { return n.h, true }
func (n node) Works() (*works.Works, bool)       { return n.w, true }
func (n node) Health() (*health.Health, bool)    { return n.hl, true }
func (n node) Minter() (artwork.Minter, bool)    { return n.m, true }

type world struct {
	*notificationstest.World
	t     *testing.T
	s     *ownertest.Owner
	works *works.Works
	rec   entry.Recorder
	attic works.Tree
	home  role.NodeID
	hd    targets.TargetID
	// faults is the Library category's items now.
	faults atomic.Pointer[[]health.Item]
	health *health.Health
}

func quiet(id health.ID, label string) health.Category {
	return health.Category{
		About:  health.About{ID: id, Label: label, Phrase: label},
		Source: health.Source{Answer: func(context.Context) (health.Answer, error) { return health.Answer{}, nil }},
	}
}

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

// within runs f over a Controller holding The Attic, a Work of one Item
// that wants HD, with History served on a fake clock.
func within(t *testing.T, f func(t *testing.T, w *world)) {
	over(t, notificationstest.Owner{Name: "noter", Rows: []entry.Entry{grabbedRow, failedRow, importedRow, deletedRow}}, f)
}

// over is within with owner's rows the ones recorded.
func over(t *testing.T, owner notificationstest.Owner, f func(t *testing.T, w *world)) {
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()
		lib := storagetest.Library(t, secret.Schema(), identity.Schema(), mail.Schema(), nodes.Schema(), works.Schema(), targets.Schema(),
			roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), conversions.Schema(), imports.Schema(),
			health.Schema(), history.Schema(), notifications.Schema())
		nw := notificationstest.Over(t, lib, filepath.Join(t.TempDir(), secret.KeyFile), notifications.DefaultTimings(),
			owner)
		w := &world{World: nw, t: t, rec: nw.Recorder(owner.Name)}
		w.faults.Store(&[]health.Item{})
		fx := contenttypetest.New()
		w.works = must[*works.Works](t)(works.Open(ctx, works.Config{
			Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
			Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Removed: discard[works.Removed]{},
		}))
		goals := must[*targets.Targets](t)(targets.Open(ctx, targets.Config{
			Library: lib, Works: w.works, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{},
		}))
		w.hd = must[targets.Target](t)(goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{
			Scores: targets.Scores{Increment: 1}, Name: "HD", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p,
		})).ID
		fx.Put("attic", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{{Key: "b1", Label: "Box 1", Children: []works.Entry{{Key: "p1", Label: "Part 1"}, {Key: "p2", Label: "Part 2"}}}}}})
		attic, _, err := w.works.Add(ctx, identity.Admin{}, contenttypetest.ID, "attic", works.Adding{Monitor: true})
		if err != nil {
			t.Fatal(err)
		}
		w.attic = attic
		w.home = role.NodeID{UUID: storage.NewUUID()}
		w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return nodestest.Insert(ctx, tx, w.home, "home")
		})
		hub := live.NewHub(nil)
		t.Cleanup(hub.Close)
		library := health.Category{
			About: health.About{ID: health.LibraryID, Label: "Library", Phrase: "the library"},
			Source: health.Source{Answer: func(context.Context) (health.Answer, error) {
				return health.Answer{Items: *w.faults.Load()}, nil
			}},
		}
		w.health = must[*health.Health](t)(health.Open(ctx, health.Config{
			Library: lib, Hub: hub, Changed: discard[health.Changed]{},
			Categories: []health.Category{
				library, quiet(health.NodesID, "Nodes"), quiet(health.DownloadClientsID, "Download Clients"), quiet(health.MediaServersID, "Media Servers"), quiet(health.RemoteAccessID, "Remote access"),
				quiet(health.SeedingID, "Seeding"), quiet(health.IndexersID, "Indexers"), quiet(health.SubtitlesID, "Subtitles"), quiet(health.ScheduledTasksID, "Scheduled tasks"), quiet(health.BackupsID, "Backups"), quiet(health.NotificationsID, "Notifications"),
			},
			Transitions: func(ctx context.Context, tx *storage.Tx[storage.Library], ts []health.Transition) error {
				return w.History.Transitions(ctx, tx, ts)
			},
			Zone: time.UTC, Timings: health.DefaultTimings(), Log: slog.New(slog.DiscardHandler),
		}))
		run, stop := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = w.health.Run(run)
		}()
		t.Cleanup(func() {
			stop()
			<-done
		})
		synctest.Wait()
		n := node{h: w.History, w: w.works, hl: w.health, m: artwork.NewMinter([]byte("history test key 32 bytes long!!"))}
		w.s = ownertest.ServeOver(t, lib, nw.Identity, func(r *api.Router) { historyapi.Routes(r, n) })
		f(t, w)
	})
}

func (w *world) write(f func(ctx context.Context, tx *storage.Tx[storage.Library]) error) {
	w.t.Helper()
	if err := w.Lib.Write(w.t.Context(), f); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) record(row entry.Row[noted], name string, about entry.About) {
	w.t.Helper()
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return row.Record(ctx, tx, w.rec, noted{Name: name, On: about})
	})
	synctest.Wait()
}

func (w *world) about(grab, imp storage.UUID) entry.About {
	return entry.About{Type: contenttypetest.ID, Works: []works.WorkID{w.attic.ID}, Grab: grab, Import: imp}
}

// grab seeds a Grab of The Attic's Part at place at HD, failed with code
// when code isn't "", and answers its id.
func (w *world) grab(code grabs.Code, place int) storage.UUID {
	w.t.Helper()
	id := storage.NewUUID()
	stage, failedStage, failure := "fetching", any(nil), any(nil)
	if code != "" {
		stage, failedStage, failure = "failed", "fetching", string(code)
	}
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO grabs_grab (id, work_id, item_id, edition, node_id, release_title, release_size,
			release_magnet, stage, failed_stage, failure_code, failure_message, created_at, updated_at)
			VALUES (?, ?, ?, 'standard', ?, 'The.Attic.1080p.WEB', 4096, 'magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567', ?, ?, ?, ?, 0, 0)`,
			id, w.attic.ID, w.attic.Items[place].ID, w.home, stage, failedStage, failure, failure); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO grabs_target (grab_id, position, item_id, target_id) VALUES (?, 0, ?, ?)`, id, w.attic.Items[place].ID, w.hd)
		return err
	})
	return id
}

// imported seeds a finished folder Import of The Attic and answers its id.
func (w *world) imported() storage.UUID {
	w.t.Helper()
	id := storage.NewUUID()
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO imports_import (id, source, node_id, folder, content_type, hint, work_id, created_at, finished_at)
			VALUES (?, 'folder', ?, '/downloads/The Attic', ?, 'The Attic', ?, 0, 0)`, id, w.home, contenttypetest.ID, w.attic.ID)
		return err
	})
	return id
}

func (w *world) deleteRow(table string, id storage.UUID) {
	w.t.Helper()
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE id = ?", id)
		return err
	})
}

// fault puts items on the Library category and has Health evaluate it.
func (w *world) fault(items ...health.Item) {
	w.t.Helper()
	w.faults.Store(&items)
	if _, err := w.health.Evaluate(w.t.Context(), health.LibraryID); err != nil {
		w.t.Fatal(err)
	}
	synctest.Wait()
}

type wireWork struct {
	ID          string  `json:"id"`
	ContentType string  `json:"contentType"`
	Label       string  `json:"label"`
	PosterURL   *string `json:"posterUrl"`
}

type wireEntry struct {
	ID    int64     `json:"id"`
	Kind  string    `json:"kind"`
	Group string    `json:"group"`
	At    time.Time `json:"at"`
	Words string    `json:"words"`
	Tone  string    `json:"tone"`
	Work  *wireWork `json:"work"`
	Links struct {
		Grab   *string `json:"grab"`
		Import *string `json:"import"`
		Health *string `json:"health"`
	} `json:"links"`
}

type wirePage struct {
	Entries []wireEntry `json:"entries"`
	Next    *int64      `json:"next"`
}

func (w *world) get(path string, v any) {
	w.t.Helper()
	status, body := w.s.Call(w.t, http.MethodGet, path, "")
	if status != http.StatusOK {
		w.t.Fatalf("GET %s = %d %s", path, status, body)
	}
	if err := json.Unmarshal([]byte(body), v); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) list(query string) wirePage {
	w.t.Helper()
	var p wirePage
	w.get("/api/v1/history"+query, &p)
	return p
}

func words(p wirePage) []string {
	out := make([]string, len(p.Entries))
	for i, e := range p.Entries {
		out[i] = e.Words
	}
	return out
}

func TestEveryKindReadsBackWithItsWordsAndLinks(t *testing.T) {
	within(t, func(t *testing.T, w *world) {
		grab, imp := w.grab("", 0), w.imported()
		w.record(grabbedRow, "The Attic", w.about(grab, storage.UUID{}))
		failed := w.about(w.grab(grabs.NameTooLong, 1), storage.UUID{})
		failed.Quiet = true
		w.record(failedRow, "quietly", failed)
		w.record(importedRow, "The Attic", w.about(grab, imp))
		w.record(deletedRow, "12 Works", entry.About{Type: contenttypetest.ID, Works: []works.WorkID{w.attic.ID, {UUID: storage.NewUUID()}}})
		w.fault(health.Item{Key: "disk", Level: health.Fail, Says: health.Says{Subject: "Disk full", Text: "The disk is full."}})

		p := w.list("")
		want := []string{"New Health item for Disk full", "Works deleted 12 Works", "Imported The Attic", "Grab failed quietly", "Grabbed The Attic"}
		if !slices.Equal(words(p), want) || p.Next != nil {
			t.Fatalf("History lists %q (next %v), want %q on one page", words(p), p.Next, want)
		}
		disk, gone, landed, quiet, grabbed := p.Entries[0], p.Entries[1], p.Entries[2], p.Entries[3], p.Entries[4]
		if disk.Kind != "new_health_item" || disk.Group != "health" || disk.Tone != "fail" || disk.Work != nil || disk.Links.Health == nil || *disk.Links.Health != "library.disk" {
			t.Errorf("the Health entry is %+v, want a failing New Health item linked to library.disk", disk)
		}
		if gone.Work != nil {
			t.Errorf("an entry about two Works reads Work %+v, want none", gone.Work)
		}
		if landed.Kind != "imported" || landed.Group != "imports" || landed.Tone != "good" || landed.Work == nil || landed.Work.Label != "The Attic" ||
			landed.Links.Grab == nil || *landed.Links.Grab != grab.String() || landed.Links.Import == nil || *landed.Links.Import != imp.String() {
			t.Errorf("the Imported entry is %+v, want it about The Attic with its Grab and Import", landed)
		}
		if quiet.Kind != "grab_failed" || quiet.Tone != "fail" {
			t.Errorf("the quiet Grab failed entry is %+v, want it listed", quiet)
		}
		if grabbed.Links.Import != nil || grabbed.Links.Health != nil {
			t.Errorf("the Grabbed entry links %+v, want its Grab alone", grabbed.Links)
		}
	})
}

func TestAnEntryReadsTheSameAfterItsWorkGrabImportAndHealthItemAreGone(t *testing.T) {
	within(t, func(t *testing.T, w *world) {
		address := must[identity.Address](t)(identity.ParseAddress("https://media.example.com"))
		if err := w.Identity.SetAddress(t.Context(), identity.Admin{}, address); err != nil {
			t.Fatal(err)
		}
		sam := w.Admin("sam@example.test")
		grab, imp := w.grab("", 0), w.imported()
		w.record(importedRow, "The Attic", w.about(grab, imp))
		w.fault(health.Item{Key: "disk", Level: health.Warn, Says: health.Says{Subject: "Disk low", Text: "The disk is low."}})
		before := w.list("")
		if link := bellLink(w.Feed(sam), "Imported The Attic"); link != "https://media.example.com/noted" {
			t.Fatalf("the bell links the Imported entry to %q, want its path on the Address", link)
		}

		w.deleteRow("imports_import", imp)
		w.deleteRow("grabs_grab", grab)
		must[[]works.WorkID](t)(w.works.Delete(t.Context(), identity.Admin{}, []works.WorkID{w.attic.ID}, func(context.Context, *storage.Tx[storage.Library], []works.WorkID) ([]works.WorkID, error) {
			return []works.WorkID{w.attic.ID}, nil
		}))
		w.fault()

		after := w.list("")
		if got := words(after); !slices.Equal(got, append([]string{"Health item for Disk low cleared"}, words(before)...)) {
			t.Fatalf("History lists %q, want %q under the clear's entry", got, words(before))
		}
		landed := after.Entries[len(after.Entries)-1]
		if landed.Work != nil || landed.Links.Grab != nil || landed.Links.Import != nil {
			t.Errorf("the Imported entry reads %+v, want no Work, Grab or Import once they are deleted", landed)
		}
		if link := bellLink(w.Feed(sam), "Imported The Attic"); link != "" {
			t.Errorf("the bell links the Imported entry to %q once its Work is deleted, want no link as on History", link)
		}
		raised := after.Entries[1]
		if raised.Words != "New Health item for Disk low" || raised.Links.Health != nil {
			t.Errorf("the New Health item reads %+v, want no Health link once the item cleared", raised)
		}
	})
}

func bellLink(f notifications.Feed, title string) string {
	for _, r := range f.Rows {
		if r.Title == title {
			return r.Link
		}
	}
	return "not on the bell"
}

func TestAOneWorkEntryIsOnItsWorksHistoryAndPagesByBefore(t *testing.T) {
	within(t, func(t *testing.T, w *world) {
		other := works.WorkID{UUID: storage.NewUUID()}
		for _, name := range []string{"one", "two", "three"} {
			w.record(grabbedRow, name, w.about(storage.UUID{}, storage.UUID{}))
			w.record(grabbedRow, "elsewhere", entry.About{Type: contenttypetest.ID, Works: []works.WorkID{other}})
		}
		first := w.list("?work=" + w.attic.ID.String() + "&limit=2")
		if !slices.Equal(words(first), []string{"Grabbed three", "Grabbed two"}) || first.Next == nil {
			t.Fatalf("the Work's first page is %q (next %v), want its two newest", words(first), first.Next)
		}
		rest := w.list("?work=" + w.attic.ID.String() + "&limit=2&before=" + strconv.FormatInt(*first.Next, 10))
		if !slices.Equal(words(rest), []string{"Grabbed one"}) || rest.Next != nil {
			t.Fatalf("the Work's next page is %q (next %v), want its oldest alone", words(rest), rest.Next)
		}
		if got := words(w.list("?groups=imports")); len(got) != 0 {
			t.Errorf("Imports alone lists %q, want nothing", got)
		}
	})
}

type problem struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Errors []struct {
		Location string `json:"location"`
		Message  string `json:"message"`
	} `json:"errors"`
}

func (w *world) refused(path string, status int) problem {
	w.t.Helper()
	code, body := w.s.Call(w.t, http.MethodGet, path, "")
	var p problem
	if err := json.Unmarshal([]byte(body), &p); err != nil || code != status {
		w.t.Fatalf("GET %s = %d %s, want %d", path, code, body, status)
	}
	return p
}

func TestTheRefusalsSayWhatToChange(t *testing.T) {
	within(t, func(t *testing.T, w *world) {
		for _, c := range []struct{ query, at, words string }{
			{"?kinds=nothing", "query.kinds", "There's no History entry kind called nothing."},
			{"?groups=imports&kinds=grab_failed", "query.kinds", "Grab failed is in Grabs, which isn't chosen. Choose Grabs too, or leave out Grab failed."},
		} {
			p := w.refused("/api/v1/history"+c.query, http.StatusUnprocessableEntity)
			if len(p.Errors) != 1 || p.Errors[0].Location != c.at || p.Errors[0].Message != c.words {
				t.Errorf("%s refused %+v, want %q at %s", c.query, p.Errors, c.words, c.at)
			}
		}
		w.record(grabbedRow, "The Attic", entry.About{})
		id := w.list("").Entries[0].ID
		p := w.refused("/api/v1/history/"+strconv.FormatInt(id, 10)+"?zone=Mars/Olympus", http.StatusUnprocessableEntity)
		if len(p.Errors) != 1 || p.Errors[0].Location != "query.zone" || p.Errors[0].Message != "There's no time zone called Mars/Olympus. Use an IANA name, such as Europe/London." {
			t.Errorf("an unknown zone refused %+v", p.Errors)
		}
		gone := w.refused("/api/v1/history/"+strconv.FormatInt(id+1, 10)+"?zone=UTC", http.StatusNotFound)
		if gone.Code != "history_entry_gone" || gone.Title != "This History entry was deleted" || gone.Detail != "It was cleared, or it was older than Keep history for." {
			t.Errorf("an entry never recorded answered %+v", gone)
		}
	})
}

type wireOne struct {
	Entry wireEntry `json:"entry"`
	Lines []string  `json:"lines"`
	Files []struct {
		Items         []string `json:"items"`
		Quality       string   `json:"quality"`
		PlacedBy      string   `json:"placedBy"`
		Replaced      string   `json:"replaced"`
		ReplacedLabel string   `json:"replacedLabel"`
	} `json:"files"`
	Stages []struct {
		Stage string    `json:"stage"`
		At    time.Time `json:"at"`
	} `json:"stages"`
	Health []struct {
		Subject string `json:"subject"`
		Text    string `json:"text"`
		Level   string `json:"level"`
	} `json:"health"`
}

func TestTheLongFormHasTheLinesFilesStagesAndHealthItemsInTheZoneAsked(t *testing.T) {
	within(t, func(t *testing.T, w *world) {
		w.record(importedRow, "The Attic", w.about(storage.UUID{}, storage.UUID{}))
		w.fault(health.Item{Key: "disk", Level: health.Fail, Says: health.Says{Subject: "Disk full", Text: "The disk is full."}})
		p := w.list("")
		var landed, raised wireOne
		w.get("/api/v1/history/"+strconv.FormatInt(p.Entries[1].ID, 10)+"?zone=Europe/London", &landed)
		if len(landed.Files) != 1 || landed.Files[0].ReplacedLabel != "The.Attic.720p" || landed.Files[0].Quality != "WEBDL-1080p" ||
			len(landed.Stages) != 1 || landed.Stages[0].Stage != "fetching" || !slices.Equal(landed.Lines, []string{"It happened."}) || landed.Entry.Words != "Imported The Attic" {
			t.Errorf("the Imported entry's long form is %+v", landed)
		}
		w.get("/api/v1/history/"+strconv.FormatInt(p.Entries[0].ID, 10)+"?zone=Asia/Tokyo", &raised)
		since := time.Now().In(must[*time.Location](t)(time.LoadLocation("Asia/Tokyo"))).Format("15:04")
		if len(raised.Health) != 1 || raised.Health[0].Subject != "Disk full" || raised.Health[0].Level != "fail" ||
			!slices.Equal(raised.Lines, []string{"The disk is full.", "It has been on Health since " + since + "."}) {
			t.Errorf("the New Health item's long form is %+v, want its item and its clock in Tokyo, %s", raised, since)
		}
	})
}

func TestAOneItemWorksImportedEntryListsEachFileWithNoItems(t *testing.T) {
	over(t, notificationstest.Owner{Name: "imports", Rows: imports.Entries()}, func(t *testing.T, w *world) {
		var spec entry.Spec
		for _, e := range imports.Entries() {
			if e.Declared().Kind == "imported" {
				spec = e.Declared()
			}
		}
		single := entry.Work{ID: w.attic.ID, Label: "The Attic", Type: contenttypetest.ID, Nouns: entry.Nouns{
			Work: works.Noun{One: "Attic", Many: "Attics"}, Item: works.Noun{One: "Attic", Many: "Attics"}, Single: true,
		}}
		w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return entry.Row[imports.ImportedFacts]{Spec: spec}.Record(ctx, tx, w.rec, imports.ImportedFacts{
				Work: single, Root: "Attics library root on home", Files: []imports.PlacedFile{{Quality: release.WEBDL1080p, Size: 1 << 30, PlacedBy: imports.ByLink}},
			})
		})
		synctest.Wait()
		p := w.list("")
		_, body := w.s.Call(t, http.MethodGet, "/api/v1/history/"+strconv.FormatInt(p.Entries[0].ID, 10)+"?zone=UTC", "")
		if !strings.Contains(body, `"files":[{"items":[],"quality":"WEBDL-1080p"`) {
			t.Errorf("the Imported entry of a one-Item Work answers %s, want its file with items []", body)
		}
	})
}

func TestTheKindsAreTheCatalogByGroup(t *testing.T) {
	within(t, func(t *testing.T, w *world) {
		var got struct {
			Groups []struct {
				Group string `json:"group"`
				Label string `json:"label"`
				Kinds []struct {
					Kind  string `json:"kind"`
					Label string `json:"label"`
				} `json:"kinds"`
			} `json:"groups"`
		}
		w.get("/api/v1/history/kinds", &got)
		var lines []string
		for _, g := range got.Groups {
			kinds := make([]string, len(g.Kinds))
			for i, k := range g.Kinds {
				kinds[i] = k.Label
			}
			lines = append(lines, g.Group+" "+g.Label+": "+strings.Join(kinds, ", "))
		}
		want := []string{
			"grabs Grabs: Ping, Grabbed, Grab failed",
			"imports Imports: Imported",
			"library Library: Pong, Works deleted",
			"health Health: New Health item, Health item cleared",
		}
		if !slices.Equal(lines, want) {
			t.Fatalf("the kinds are %q, want %q", lines, want)
		}
	})
}

type wireSentence struct {
	Words    string `json:"words"`
	NeedsYou int    `json:"needsYou"`
}

func (w *world) summary() *wireSentence {
	w.t.Helper()
	var s *wireSentence
	w.get("/api/v1/history/summary?since="+url.QueryEscape(time.Now().Add(-7*24*time.Hour).Format(time.RFC3339)), &s)
	return s
}

func TestTheSummaryCountsTheWeekAndLeadsWithWhatNeedsTheAdminEvenAfterAClear(t *testing.T) {
	within(t, func(t *testing.T, w *world) {
		if s := w.summary(); s != nil {
			t.Fatalf("an empty History sums up as %+v, want null", s)
		}
		w.record(failedRow, "long ago", entry.About{})
		time.Sleep(8 * 24 * time.Hour)
		if s := w.summary(); s == nil || s.Words != "There were no Grabs or Imports in the last 7 days." {
			t.Fatalf("a quiet week sums up as %+v", s)
		}
		w.record(importedRow, "one", entry.About{})
		w.record(importedRow, "two", entry.About{})
		if s := w.summary(); s == nil || s.Words != "There were 2 Imports in the last 7 days." {
			t.Fatalf("two Imports sum up as %+v", s)
		}
		w.record(grabbedRow, "one", entry.About{})
		if s := w.summary(); s == nil || s.Words != "There was 1 Grab and 2 Imports in the last 7 days." {
			t.Fatalf("a Grab and two Imports sum up as %+v", s)
		}
		w.grab(grabs.NameTooLong, 0)
		w.grab(grabs.Canceled, 1)
		w.grab(grabs.NameTooLong, 1)
		if status, body := w.s.Call(t, http.MethodDelete, "/api/v1/history", ""); status != http.StatusAccepted {
			t.Fatalf("DELETE /history = %d %s", status, body)
		}
		synctest.Wait()
		if got := w.list(""); len(got.Entries) != 0 {
			t.Fatalf("after a Clear History lists %q", words(got))
		}
		if s := w.summary(); s == nil || s.Words != "2 Grabs failed and need attention." || s.NeedsYou != 2 {
			t.Fatalf("two failed Grabs after a Clear sum up as %+v", s)
		}
	})
}
