package indexers_test

import (
	"context"
	"crypto/sha1" //nolint:gosec // G505: the catalog's sha is SHA-1
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/catalog"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site/definitions"
	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// recorder keeps the path of every request the catalog client sends.
type recorder struct {
	next  http.RoundTripper
	mu    sync.Mutex
	paths []string
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.paths = append(r.paths, strings.TrimPrefix(req.URL.Path, catalog.Base+"/12/"))
	r.mu.Unlock()
	res, err := r.next.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("send %s: %w", req.URL, err)
	}
	return res, nil
}

// take is the paths sent since the last take.
func (r *recorder) take() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.paths
	r.paths = nil
	return out
}

type catalogWorld struct {
	fake   faketest.Served
	sent   *recorder
	x      *indexers.Indexers
	lib    *storage.DB[storage.Library]
	sealer *secret.Sealer
	home   role.NodeID
}

var nextRefresh = time.Date(2026, 10, 28, 3, 0, 0, 0, time.UTC)

func newCatalogWorld(t *testing.T) *catalogWorld {
	t.Helper()
	f := faketest.Loopback(t, catalog.Fake, scenario.Scenario{})
	sent := &recorder{next: f.Client.Transport}
	w := &catalogWorld{
		fake: f, sent: sent, sealer: secrettest.Sealer(t), home: role.NodeID{UUID: storage.NewUUID()},
		lib: storagetest.Library(t, secret.Schema(), nodes.Schema(), works.Schema(), indexers.Schema()),
	}
	w.x = indexers.New(indexers.Config{
		Library: w.lib, Sealer: w.sealer, Roster: indexerstest.Alone(w.home),
		HTTP:        &http.Client{Transport: sent},
		Endpoints:   indexers.Endpoints{Catalog: endpoint.MustParseURL(f.Address.URL(catalog.Base))},
		CatalogNext: func(context.Context) (time.Time, error) { return nextRefresh, nil },
		Changed:     discard[indexers.Changed]{}, Status: discard[indexers.StatusChanged]{}, Trackers: discard[indexers.TrackerChanged]{}, Catalogs: discard[indexers.CatalogChanged]{},
		Timings: indexers.DefaultTimings(), Zone: time.UTC, Log: slog.New(slog.DiscardHandler),
	})
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, w.home, "home")
	})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// refresh runs the Indexer catalog refresh's one unit.
func (w *catalogWorld) refresh(t *testing.T) error {
	t.Helper()
	task := w.x.Tasks()[0]
	if task.ID != indexers.CatalogRefreshTaskID {
		t.Fatalf("the Indexers' first task is %s", task.ID)
	}
	return task.Run(t.Context(), tasks.Unit{}, nil)
}

func (w *catalogWorld) mustRefresh(t *testing.T) {
	t.Helper()
	if err := w.refresh(t); err != nil {
		t.Fatalf("refresh: %v", err)
	}
}

func (w *catalogWorld) catalog(t *testing.T) indexers.CatalogView {
	t.Helper()
	c, err := w.x.Catalog(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// shas are the catalog's entries by id, each with its version's sha read
// from the fake's files.
func (w *catalogWorld) entry(t *testing.T, id string) (indexers.Entry, bool) {
	t.Helper()
	for _, e := range w.catalog(t).Entries {
		if e.ID == id && e.Origin == indexers.FromCatalog {
			return e, true
		}
	}
	return indexers.Entry{}, false
}

func (w *catalogWorld) cue(t *testing.T, spec string, args any) { w.fake.Cue(t, spec, args) }

// pin saves a catalog Indexer on the catalog's version of definition.
func (w *catalogWorld) pin(t *testing.T, name, definition string, plain, sealed map[string]string) indexers.ID {
	t.Helper()
	id := indexers.ID{UUID: storage.NewUUID()}
	sha := w.pinned(t, definition)
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return indexerstest.Catalog(ctx, tx, w.sealer, id, name, w.home, definition, sha, plain, sealed)
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// pinned is the sha of the catalog's version of definition, from the
// fake's own index.
func (w *catalogWorld) pinned(t *testing.T, definition string) string {
	t.Helper()
	index, _, err := indexers.ReadCatalog(t.Context(), w.fake.Client, w.fake.Address.URL(catalog.Base), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range index {
		if e.ID == definition {
			return e.SHA
		}
	}
	t.Fatalf("the catalog lists no %s", definition)
	return ""
}

func (w *catalogWorld) indexer(t *testing.T, id indexers.ID) (indexers.Indexer, indexers.Definition) {
	t.Helper()
	idx, err := indexers.Get(t.Context(), w.lib, id)
	if err != nil {
		t.Fatal(err)
	}
	def, ok := idx.Source.(indexers.Definition)
	if !ok {
		t.Fatalf("Indexer %s has no definition", idx.Name)
	}
	return idx, def
}

func (w *catalogWorld) health(t *testing.T) []health.Item {
	t.Helper()
	a, err := w.x.HealthSource().Answer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return a.Items
}

func TestBeforeAnyRefreshTheCatalogSaysWhenItTriesAgain(t *testing.T) {
	w := newCatalogWorld(t)
	c := w.catalog(t)
	want := "SlipStream hasn't downloaded the Indexer catalog yet. SlipStream retries at 03:00."
	if c.Notice != want || len(c.Entries) != 0 || !c.Refreshed.IsZero() || !c.Next.Equal(nextRefresh) || !c.IsDefault {
		t.Errorf("Catalog = %+v, want no entries, the default address and the notice %q", c, want)
	}
	if _, err := w.x.Form(t.Context(), identity.Admin{}, "larkspur"); !errors.Is(err, indexers.ErrCatalogUnavailable) {
		t.Errorf("Form(larkspur) = %v, want ErrCatalogUnavailable", err)
	}
}

func TestTheFirstRefreshFetchesPackageZipAndListsEveryDefinition(t *testing.T) {
	w := newCatalogWorld(t)
	w.mustRefresh(t)
	if got, want := w.sent.take(), []string{"", "package.zip"}; !slices.Equal(got, want) {
		t.Errorf("the first refresh sent %q, want %q", got, want)
	}
	c := w.catalog(t)
	var names []string
	for _, e := range c.Entries {
		names = append(names, e.Name)
	}
	if !slices.Equal(names, []string{"Broken", "Cinder", "Larkspur", "Quarry"}) || c.Notice != "" || c.Refreshed.IsZero() {
		t.Errorf("Catalog lists %q with notice %q, refreshed %v; want the four by name", names, c.Notice, c.Refreshed)
	}
	broken, _ := w.entry(t, "broken")
	if broken.Reason != "it has no links" || len(broken.Categories) != 0 {
		t.Errorf("broken = %+v, want it refused with its reason", broken)
	}
	quarry, _ := w.entry(t, "quarry")
	if quarry.Reason != "" || quarry.Privacy != cardigann.SemiPrivate || quarry.Language != "en-US" || !slices.Contains(quarry.Categories, 2000) {
		t.Errorf("quarry = %+v, want it running, semi-private, in English and in Movies", quarry)
	}
}

func TestAnUnchangedCatalogAnswers304AndFetchesNoFile(t *testing.T) {
	w := newCatalogWorld(t)
	w.mustRefresh(t)
	w.sent.take()
	before, _ := w.entry(t, "larkspur")
	w.mustRefresh(t)
	if got := w.sent.take(); !slices.Equal(got, []string{""}) {
		t.Errorf("a refresh of an unchanged catalog sent %q, want only the index", got)
	}
	if after, _ := w.entry(t, "larkspur"); !reflect.DeepEqual(after, before) {
		t.Errorf("larkspur = %+v after a 304, want %+v", after, before)
	}
}

func TestAnUpdateFetchesOnlyTheChangedFile(t *testing.T) {
	w := newCatalogWorld(t)
	w.mustRefresh(t)
	w.sent.take()
	w.cue(t, catalog.Update.Name, nil)
	w.mustRefresh(t)
	if got, want := w.sent.take(), []string{"", "larkspur"}; !slices.Equal(got, want) {
		t.Errorf("a refresh after one update sent %q, want %q", got, want)
	}
	if e, _ := w.entry(t, "larkspur"); !strings.Contains(e.Description, "no longer sorts by seeders") {
		t.Errorf("larkspur = %+v, want its second version", e)
	}
}

func TestAZipEntryAtAnotherShaIsFetchedAlone(t *testing.T) {
	w := newCatalogWorld(t)
	w.cue(t, catalog.StaleZip.Name, nil)
	w.cue(t, catalog.Update.Name, nil)
	w.mustRefresh(t)
	if got, want := w.sent.take(), []string{"", "package.zip", "larkspur"}; !slices.Equal(got, want) {
		t.Errorf("a refresh with a stale zip sent %q, want %q", got, want)
	}
	if e, _ := w.entry(t, "larkspur"); !strings.Contains(e.Description, "no longer sorts by seeders") {
		t.Errorf("larkspur = %+v, want the version the index lists", e)
	}
}

func TestACorruptZipFetchesEveryFileAlone(t *testing.T) {
	w := newCatalogWorld(t)
	w.cue(t, catalog.CorruptZip.Name, nil)
	w.mustRefresh(t)
	if got, want := w.sent.take(), []string{"", "package.zip", "broken", "cinder", "larkspur", "quarry"}; !slices.Equal(got, want) {
		t.Errorf("a refresh with a corrupt zip sent %q, want %q", got, want)
	}
	if len(w.catalog(t).Entries) != 4 {
		t.Errorf("the catalog lists %d definitions, want 4", len(w.catalog(t).Entries))
	}
}

func TestAFailedRefreshKeepsTheLastGoodCatalog(t *testing.T) {
	for _, c := range []struct {
		name string
		fail func(w *catalogWorld)
		want string
	}{
		{"an unknown schema", func(w *catalogWorld) { w.cue(t, catalog.Schema.Name, map[string]int{"schema": 13}) }, "has no schema 12 index"},
		{"an outage", func(w *catalogWorld) {
			if _, err := w.fake.Faults.Set(fault.Rule{ID: "outage", Target: fault.Target(catalog.Fake.Name), Fault: fault.Status{Code: http.StatusServiceUnavailable}}); err != nil {
				t.Fatal(err)
			}
		}, "503 Service Unavailable"},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newCatalogWorld(t)
			w.mustRefresh(t)
			before := w.catalog(t)
			w.cue(t, catalog.Update.Name, nil)
			c.fail(w)
			if err := w.refresh(t); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("refresh = %v, want an error naming %q", err, c.want)
			}
			if after := w.catalog(t); !reflect.DeepEqual(after, before) {
				t.Errorf("the catalog after a failed refresh = %+v, want %+v", after, before)
			}
		})
	}
}

func TestAPinMovesToANewVersionThatLoadsWithNotes(t *testing.T) {
	w := newCatalogWorld(t)
	w.mustRefresh(t)
	id := w.pin(t, "Larkspur", "larkspur", map[string]string{"username": "ada", "sort": "seeders"}, map[string]string{"password": "hunter2"})
	w.cue(t, catalog.Update.Name, nil)
	w.mustRefresh(t)
	idx, def := w.indexer(t, id)
	wantNotes := []string{`The definition's new version no longer offers "seeders" for Sort requested from site. The Indexer now uses its default, "created".`}
	if def.SHA != w.pinned(t, "larkspur") || !reflect.DeepEqual(def.Plain, map[string]string{"username": "ada"}) || !slices.Equal(idx.Notes, wantNotes) {
		t.Errorf("the Indexer pins %s with %v and notes %q, want the new version with the username and the notes %q", def.SHA, def.Plain, idx.Notes, wantNotes)
	}
	if items := w.health(t); len(items) != 0 {
		t.Errorf("Health = %+v, want nothing once the pin moved", items)
	}
}

func TestABadUpdateLeavesThePinWithAWarning(t *testing.T) {
	w := newCatalogWorld(t)
	w.mustRefresh(t)
	before := w.pinned(t, "larkspur")
	id := w.pin(t, "Larkspur", "larkspur", map[string]string{"username": "ada"}, nil)
	w.cue(t, catalog.BadUpdate.Name, nil)
	w.mustRefresh(t)
	if _, def := w.indexer(t, id); def.SHA != before {
		t.Errorf("the Indexer pins %s, want %s", def.SHA, before)
	}
	if e, _ := w.entry(t, "larkspur"); e.Reason != `it logs in with "twofactor", which SlipStream doesn't run` {
		t.Errorf("the catalog's larkspur = %+v, want refused", e)
	}
	wantHealth(t, w.health(t), id, "The catalog's new version of Larkspur doesn't load. The Indexer keeps running the version from "+today()+".")
}

func TestASiteThatLeftTheCatalogKeepsRunningWithAWarning(t *testing.T) {
	w := newCatalogWorld(t)
	w.mustRefresh(t)
	sha := w.pinned(t, "cinder")
	id := w.pin(t, "Cinder", "cinder", nil, map[string]string{"cookie": "c=1"})
	w.cue(t, catalog.Remove.Name, map[string]string{"id": "cinder"})
	w.mustRefresh(t)
	if _, ok := w.entry(t, "cinder"); ok {
		t.Error("the catalog still lists cinder")
	}
	if _, def := w.indexer(t, id); def.SHA != sha {
		t.Errorf("the Indexer pins %s, want %s", def.SHA, sha)
	}
	wantHealth(t, w.health(t), id, "Cinder is no longer in the Indexer catalog. The Indexer keeps running the version from "+today()+".")
}

func today() string { return time.Now().UTC().Format("2 Jan") }

func wantHealth(t *testing.T, items []health.Item, id indexers.ID, text string) {
	t.Helper()
	if len(items) != 1 || items[0].Level != health.Warn || items[0].Text != text ||
		items[0].Action.Href() != "/settings/indexers/"+id.String() {
		t.Errorf("Health = %+v, want one Warn %q linking to the Indexer", items, text)
	}
}

func definition(t *testing.T, file string) []byte {
	t.Helper()
	src, err := fs.ReadFile(definitions.FS, file)
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func TestACustomDefinitionMustLoadAndKeepItsOwnID(t *testing.T) {
	w := newCatalogWorld(t)
	larkspur := definition(t, "larkspur.yml")
	custom, err := w.x.AddCustom(t.Context(), identity.Admin{}, larkspur)
	if err != nil || custom.Origin != indexers.FromCustom || custom.Name != "Larkspur" {
		t.Fatalf("AddCustom(larkspur) before the catalog lists it = %+v, %v", custom, err)
	}
	w.mustRefresh(t)
	if e, _ := w.entry(t, "larkspur"); !e.ReplacedBy {
		t.Errorf("the catalog's larkspur = %+v, want it replaced by the custom definition", e)
	}
	if _, err := w.x.AddCustom(t.Context(), identity.Admin{}, definition(t, "quarry.yml")); !errors.Is(err, indexers.ErrDefinitionIDTaken) {
		t.Errorf("AddCustom(quarry) = %v, want ErrDefinitionIDTaken", err)
	}
	var unrunnable *cardigann.Unrunnable
	if _, err := w.x.AddCustom(t.Context(), identity.Admin{}, definition(t, "larkspur-v3.yml")); !errors.As(err, &unrunnable) {
		t.Errorf("AddCustom(larkspur-v3) = %v, want *cardigann.Unrunnable", err)
	}
	form, err := w.x.Form(t.Context(), identity.Admin{}, "larkspur")
	if err != nil || form.Origin != indexers.FromCustom {
		t.Errorf("Form(larkspur) = %+v, %v; want the custom definition's", form, err)
	}
}

func TestReplacingACustomDefinitionMovesItsIndexers(t *testing.T) {
	w := newCatalogWorld(t)
	first, err := w.x.AddCustom(t.Context(), identity.Admin{}, definition(t, "larkspur.yml"))
	if err != nil {
		t.Fatal(err)
	}
	id := indexers.ID{UUID: storage.NewUUID()}
	err = w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return indexerstest.Catalog(ctx, tx, w.sealer, id, "Larkspur", w.home, "larkspur", shaOf(definition(t, "larkspur.yml")), map[string]string{"sort": "size"}, nil)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.x.DeleteCustom(t.Context(), identity.Admin{}, "larkspur"); !errors.Is(err, indexers.ErrDefinitionInUse) {
		t.Errorf("DeleteCustom of a definition an Indexer runs = %v, want ErrDefinitionInUse", err)
	}
	second, err := w.x.AddCustom(t.Context(), identity.Admin{}, definition(t, "larkspur-v2.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if _, def := w.indexer(t, id); def.SHA != shaOf(definition(t, "larkspur-v2.yml")) || def.Plain["sort"] != "size" {
		t.Errorf("the Indexer pins %s with %v, want the replacement with sort kept", def.SHA, def.Plain)
	}
	var custom []indexers.Entry
	for _, e := range w.catalog(t).Entries {
		if e.Origin == indexers.FromCustom {
			custom = append(custom, e)
		}
	}
	if len(custom) != 1 || custom[0].Description != second.Description || second.Description == first.Description {
		t.Errorf("the custom definitions are %+v, want only the replacement", custom)
	}
}

func TestDeletingACustomDefinitionNoIndexerRuns(t *testing.T) {
	w := newCatalogWorld(t)
	if _, err := w.x.AddCustom(t.Context(), identity.Admin{}, definition(t, "quarry.yml")); err != nil {
		t.Fatal(err)
	}
	if err := w.x.DeleteCustom(t.Context(), identity.Admin{}, "quarry"); err != nil {
		t.Fatal(err)
	}
	if err := w.x.DeleteCustom(t.Context(), identity.Admin{}, "quarry"); !errors.Is(err, indexers.ErrNoDefinition) {
		t.Errorf("a second DeleteCustom = %v, want ErrNoDefinition", err)
	}
	if n := len(w.catalog(t).Entries); n != 0 {
		t.Errorf("the catalog lists %d definitions, want none", n)
	}
}

func TestTheFormSealsEveryTextFieldButTheUsername(t *testing.T) {
	w := newCatalogWorld(t)
	w.mustRefresh(t)
	form, err := w.x.Form(t.Context(), identity.Admin{}, "larkspur")
	if err != nil {
		t.Fatal(err)
	}
	sealed := map[string]bool{}
	for _, f := range form.Fields {
		sealed[f.Name] = f.Sealed
	}
	want := map[string]bool{"username": false, "password": true, "sort": false, "cardigannCaptcha": false}
	if !reflect.DeepEqual(sealed, want) || !form.Session || !form.Captcha || form.Solver || len(form.Links) != 1 {
		t.Errorf("Form(larkspur) = %+v, want fields sealed as %v, a session, a captcha and one link", form, want)
	}
	if cinder, err := w.x.Form(t.Context(), identity.Admin{}, "cinder"); err != nil || !cinder.Solver {
		t.Errorf("Form(cinder) = %+v, %v; want it to ask for a Challenge solver", cinder, err)
	}
	if broken, err := w.x.Form(t.Context(), identity.Admin{}, "broken"); err != nil || broken.Reason == "" || len(broken.Fields) != 0 {
		t.Errorf("Form(broken) = %+v, %v; want its reason and no fields", broken, err)
	}
	if _, err := w.x.Form(t.Context(), identity.Admin{}, "nowhere"); !errors.Is(err, indexers.ErrNoDefinition) {
		t.Errorf("Form(nowhere) = %v, want ErrNoDefinition", err)
	}
}

func TestTheCatalogAddressIsSavedOnlyWhenItServesAnIndex(t *testing.T) {
	w := newCatalogWorld(t)
	var unreachable *indexers.CatalogUnreachableError
	for _, addr := range []string{"ftp://catalog.test", w.fake.Address.URL("/elsewhere")} {
		if err := w.x.SetCatalogAddress(t.Context(), identity.Admin{}, &addr); !errors.As(err, &unreachable) {
			t.Errorf("SetCatalogAddress(%s) = %v, want *CatalogUnreachableError", addr, err)
		}
	}
	addr := w.fake.Address.URL(catalog.Base + "/")
	if err := w.x.SetCatalogAddress(t.Context(), identity.Admin{}, &addr); err != nil {
		t.Fatal(err)
	}
	if c := w.catalog(t); c.IsDefault || c.Address != strings.TrimSuffix(addr, "/") {
		t.Errorf("the catalog address = %q (default %t), want %q", c.Address, c.IsDefault, addr)
	}
	if err := w.x.SetCatalogAddress(t.Context(), identity.Admin{}, nil); err != nil {
		t.Fatal(err)
	}
	if c := w.catalog(t); !c.IsDefault {
		t.Errorf("the catalog address = %q, want the default", c.Address)
	}
}

// shaOf is src's git blob id, the sha a catalog lists.
func shaOf(src []byte) string {
	h := sha1.New() //nolint:gosec // G401: the catalog's sha is SHA-1
	_, _ = fmt.Fprintf(h, "blob %d\x00", len(src))
	_, _ = h.Write(src)
	return hex.EncodeToString(h.Sum(nil))
}
