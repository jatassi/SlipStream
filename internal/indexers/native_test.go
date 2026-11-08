package indexers_test

import (
	"context"
	"errors"
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
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexerfields"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

type codec int

func (codec) Options() []port.Option {
	return []port.Option{{Value: "1", Label: "H.264"}, {Value: "5", Label: "HEVC"}}
}

type (
	lanternSettings struct {
		port.PasskeyField
		Codecs []codec `port:"codecs" label:"Codecs" default:"1"`
		Pages  int     `port:"pages" label:"Pages" min:"1" max:"5" default:"2"`
	}
	quillSettings struct{ port.UserPass }
	speedSettings struct{ port.EmailPass }
	emberSettings struct{}
	keyed         struct{ port.APIKeyField }
)

func generator[S any](*port.Indexer[S], port.SearchCriteria) (port.Chain, error) {
	return port.Chain{}, nil
}

func parser[S any](*port.Indexer[S], *port.IndexerResponse) ([]port.TorrentInfo, error) {
	return nil, nil
}

func signIn[S any](*port.SignIn[S]) error { return nil }

func tokenIn[S any](*port.SignIn[S]) (string, error) { return "", nil }

var synced = time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)

// errValue stands for any *indexerfields.ValueError in a want.
var errValue = errors.New("a ValueError")

func fixtureEntry(id port.ID, name string, privacy port.Privacy) port.Entry {
	return port.Entry{
		ID: id, Name: name, Description: name + " is a fictional tracker for tests.", Privacy: privacy,
		IndexerURLs:  []string{"https://" + string(id) + ".test/", "https://" + string(id) + "-mirror.test/"},
		Capabilities: port.Capabilities{SearchParams: []port.Param{port.Q}, Categories: port.Categories(port.Map(1, newznab.Movies, "Movies"))},
		Upstream:     port.Upstream{Commit: "ef9c1824", Date: synced},
	}
}

// The test ports, one per family the Controller tells apart: Lantern by
// a passkey, replacing the catalog's Larkspur; Quill signing in with a
// two-factor code; Speed signing in to a token with an email; Ember by a
// pasted cookie with its User-Agent; Ghost and Fern to retire and move.
var (
	lantern = func() port.Port {
		e := fixtureEntry("lantern", "Lantern", port.Private)
		e.QueryLimit, e.RateLimit = port.PerHour(150), 4*time.Second
		e.Replaces, e.Aliases = []string{"larkspur"}, []string{"larkspur-old"}
		return port.Define(e, port.Impl[lanternSettings]{Login: port.NoLogin{}, Basic: generator[lanternSettings], Parse: parser[lanternSettings]})
	}()
	quill = port.Define(fixtureEntry("quill", "Quill", port.Private), port.Impl[quillSettings]{
		Login: port.CodeLogin[quillSettings]{DoLogin: signIn[quillSettings]}, Basic: generator[quillSettings], Parse: parser[quillSettings],
	})
	speed = port.Define(fixtureEntry("speed", "Speed", port.Private), port.Impl[speedSettings]{
		Login: port.TokenLogin[speedSettings]{DoLogin: tokenIn[speedSettings]}, Basic: generator[speedSettings], Parse: parser[speedSettings],
	})
	ember = port.Define(fixtureEntry("ember", "Ember", port.SemiPrivate), port.Impl[emberSettings]{
		Login: port.CookieLogin{UserAgent: true}, Basic: generator[emberSettings], Parse: parser[emberSettings],
	})
	ghost = port.Define(fixtureEntry("ghost", "Ghost", port.Public), port.Impl[keyed]{Login: port.NoLogin{}, Basic: generator[keyed], Parse: parser[keyed]})
	fern  = port.Define(fixtureEntry("fern", "Fern", port.SemiPrivate), port.Impl[keyed]{Login: port.NoLogin{}, Basic: generator[keyed], Parse: parser[keyed]})

	livePorts = port.NewRegistry([]port.Port{lantern, quill, speed, ember, ghost, fern}, nil, nil)
	// laterPorts is a later release's: Ghost's site died, and Fern's moved
	// to the catalog's Quarry.
	laterPorts = port.NewRegistry([]port.Port{lantern, quill, speed, ember},
		[]port.Retired{{ID: "ghost", Name: "Ghost", Reason: "the site closed"}},
		[]port.Moved{{ID: "fern", Definition: "quarry", Fields: map[string]string{"apiKey": "apikey"}}})
)

// portNode is a Node that answers each job at once with answer's answer,
// else a passing Test or no Releases, and keeps every job.
type portNode struct {
	mu     sync.Mutex
	jobs   []searching.Job
	answer func(searching.Job) searching.Answer
}

func (*portNode) ReachSolver(context.Context, searching.Solver) (int, error) {
	return 0, errors.New("the port Node has no Challenge solver")
}

func (n *portNode) Run(_ context.Context, j searching.Job) (searching.Poll, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.jobs = append(n.jobs, j)
	var a searching.Answer
	switch {
	case n.answer != nil:
		a = n.answer(j)
	default:
		if _, ok := j.Call.(searching.TestCall); ok {
			a.Report = &searching.Report{Capabilities: searching.Capabilities{Modes: map[string][]string{"search": {"q"}}, Categories: []int{2000}}}
		}
	}
	a.Started = time.Now()
	return searching.Poll{Started: a.Started, Answer: &a}, nil
}

func (n *portNode) last() searching.Job {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.jobs[len(n.jobs)-1]
}

func (n *portNode) answering(f func(searching.Job) searching.Answer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.answer = f
}

// stopping answers every job with a stop for cause.
func stopping(cause searching.StopCause, text string) func(searching.Job) searching.Answer {
	return func(searching.Job) searching.Answer {
		return searching.Answer{Outcome: &searching.Stopped{Cause: cause, Text: text}}
	}
}

type nativeWorld struct {
	t      *testing.T
	lib    *storage.DB[storage.Library]
	sealer *secret.Sealer
	fake   faketest.Served
	home   role.NodeID
	node   *portNode
	x      *indexers.Indexers
}

func newNativeWorld(t *testing.T, ports port.Catalog) *nativeWorld {
	t.Helper()
	w := &nativeWorld{
		t: t, lib: storagetest.Library(t, secret.Schema(), nodes.Schema(), works.Schema(), indexers.Schema()), sealer: secrettest.Sealer(t),
		fake: faketest.Loopback(t, catalog.Fake, scenario.Scenario{}), home: role.NodeID{UUID: storage.NewUUID()}, node: &portNode{},
	}
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, w.home, "home")
	})
	if err != nil {
		t.Fatal(err)
	}
	w.x = w.with(ports, w.node)
	return w
}

// with is Indexers over w's library reading ports and asking remote, as a
// later release of the Controller.
func (w *nativeWorld) with(ports port.Catalog, remote searching.Remote) *indexers.Indexers {
	return indexers.New(indexers.Config{
		Library: w.lib, Sealer: w.sealer, Roster: indexerstest.Alone(w.home), Ports: ports,
		Searching: func(role.NodeID) searching.Remote { return remote },
		Types:     map[works.TypeID]indexers.Facet{indexerstest.Type: indexerstest.Facet{}},
		HTTP:      w.fake.Client, Endpoints: indexers.Endpoints{Catalog: endpoint.MustParseURL(w.fake.Address.URL(catalog.Base))},
		CatalogNext: func(context.Context) (time.Time, error) { return time.Time{}, nil },
		Changed:     discard[indexers.Changed]{}, Status: discard[indexers.StatusChanged]{}, Trackers: discard[indexers.TrackerChanged]{},
		Catalogs: discard[indexers.CatalogChanged]{},
		Timings:  indexers.DefaultTimings(), Zone: time.UTC, Log: slog.New(slog.DiscardHandler),
	})
}

func (w *nativeWorld) refresh() { w.refreshWith(w.x) }

// refreshWith runs x's Indexer catalog refresh.
func (w *nativeWorld) refreshWith(x *indexers.Indexers) {
	w.t.Helper()
	if err := x.Tasks()[0].Run(w.t.Context(), tasks.Unit{}, nil); err != nil {
		w.t.Fatalf("refresh: %v", err)
	}
}

func (w *nativeWorld) add(d indexers.Draft) indexers.View {
	w.t.Helper()
	d.On, d.AutoSearch = true, true
	v, err := w.x.Create(w.t.Context(), identity.Admin{}, d, indexers.Typed{})
	if err != nil {
		w.t.Fatalf("Create(%s) = %v", d.Name, err)
	}
	return v
}

func (w *nativeWorld) view(x *indexers.Indexers, id indexers.ID) indexers.View {
	w.t.Helper()
	v, err := x.Get(w.t.Context(), identity.Admin{}, id)
	if err != nil {
		w.t.Fatal(err)
	}
	return v
}

func (w *nativeWorld) trackerOf(v indexers.View) indexers.Tracker {
	w.t.Helper()
	if len(v.Trackers) != 1 {
		w.t.Fatalf("%s reaches Trackers %v, want one", v.Name, v.Trackers)
	}
	tr, err := w.x.Tracker(w.t.Context(), identity.Admin{}, v.Trackers[0])
	if err != nil {
		w.t.Fatal(err)
	}
	return tr
}

func (w *nativeWorld) healthText(x *indexers.Indexers, id indexers.ID) (health.Item, bool) {
	w.t.Helper()
	a, err := x.HealthSource().Answer(w.t.Context())
	if err != nil {
		w.t.Fatal(err)
	}
	i := slices.IndexFunc(a.Items, func(it health.Item) bool { return it.Key == health.KeyOf("stopped", id.String()) })
	if i < 0 {
		return health.Item{}, false
	}
	return a.Items[i], true
}

// native is a draft of port p with its required settings filled.
func native(name string, p port.ID) indexers.Draft {
	d := indexers.Draft{Kind: indexers.Native, Name: name, Port: p, Values: map[string]string{}}
	e, _ := livePorts.Lookup(p)
	for _, f := range e.Fields {
		if f.Required {
			d.Values[f.Name] = "x"
		}
	}
	return d
}

func TestPortsAreListedBeforeTheCatalogAndHideOnlyTheIDsTheyReplace(t *testing.T) {
	w := newNativeWorld(t, livePorts)
	offline, err := w.x.Catalog(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range offline.Entries {
		names = append(names, e.Name+":"+string(e.Origin))
	}
	if got := strings.Join(names, " "); got != "Ember:native Fern:native Ghost:native Lantern:native Quill:native Speed:native" {
		t.Errorf("before any refresh the list is %s", got)
	}
	want := indexers.Entry{
		ID: "lantern", Name: "Lantern", Description: "Lantern is a fictional tracker for tests.", Language: "en-US", Privacy: "private",
		Categories: []newznab.ID{2000, newznab.Custom("1")}, Origin: indexers.FromPort, Version: synced,
	}
	if i := slices.IndexFunc(offline.Entries, func(e indexers.Entry) bool { return e.ID == "lantern" }); !reflect.DeepEqual(offline.Entries[i], want) {
		t.Errorf("Lantern is listed as %+v, want %+v", offline.Entries[i], want)
	}

	w.refresh()
	c, err := w.x.Catalog(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]indexers.Origin{}
	for _, e := range c.Entries {
		ids[e.ID] = e.Origin
	}
	if _, ok := ids["larkspur"]; ok {
		t.Error("the catalog's Larkspur is listed though Lantern replaces it")
	}
	if ids["quarry"] != indexers.FromCatalog || ids["cinder"] != indexers.FromCatalog {
		t.Errorf("a catalog id no port replaces is hidden: %v", ids)
	}
	if c.Entries[0].Origin != indexers.FromPort || c.Entries[len(c.Entries)-1].Origin != indexers.FromCatalog {
		t.Errorf("the ports aren't listed first: %+v", c.Entries)
	}

	_, err = w.x.AddCustom(t.Context(), identity.Admin{}, []byte(strings.Replace(string(definition(t, "quarry.yml")), "id: quarry", "id: lantern", 1)))
	var taken *indexers.PortIDTakenError
	if !errors.As(err, &taken) || taken.Name != "Lantern" || !errors.Is(err, indexers.ErrDefinitionIDTaken) {
		t.Errorf("AddCustom with a port's id = %v, want definition_id_taken naming Lantern", err)
	}
}

func TestACatalogIndexerOnAnIDAPortNowReplacesKeepsRunning(t *testing.T) {
	w := newNativeWorld(t, port.NewRegistry([]port.Port{quill}, nil, nil))
	w.refresh()
	larkspur := w.add(indexers.Draft{Kind: indexers.Catalog, Name: "Larkspur", Definition: "larkspur", Values: map[string]string{"username": "ada", "password": "pw"}})

	x := w.with(livePorts, w.node)
	if _, err := x.Test(t.Context(), identity.Admin{}, larkspur.ID, indexers.Typed{}); err != nil {
		t.Fatalf("Test of the catalog Indexer on a replaced id = %v", err)
	}
	if j := w.node.last(); j.Definition == nil || j.Port != "" {
		t.Errorf("its Test ran %+v, want its definition", j)
	}
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		lines, err := x.Plan(ctx, tx, indexerstest.Type)
		if err == nil && (len(lines) != 1 || lines[0].Skipped != nil) {
			t.Errorf("Plan = %+v, want Larkspur asked", lines)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestACatalogIndexersEditTakesANewPastedCookieAndTestsWithItFirst(t *testing.T) {
	w := newNativeWorld(t, livePorts)
	w.refresh()
	v := w.add(indexers.Draft{Kind: indexers.Catalog, Name: "Cinder", Definition: "cinder", Cookie: &indexers.PastedCookie{Cookie: "uid=1", UserAgent: "Firefox"}})
	if _, err := w.x.Update(t.Context(), identity.Admin{}, v.ID, indexers.Edit{Cookie: &indexers.PastedCookie{Cookie: "uid=2", UserAgent: "Safari"}}, indexers.Typed{}); err != nil {
		t.Fatal(err)
	}
	j := w.node.last()
	if j.Session == nil || !j.Session.Pasted || j.Session.UserAgent != "Safari" || len(j.Session.Cookies) != 1 || j.Session.Cookies[0].Value != "2" {
		t.Errorf("the edit's Test ran with session %+v, want the new cookie", j.Session)
	}
	if got := w.view(w.x, v.ID); got.Session == nil || !got.Session.Pasted {
		t.Errorf("after the edit the session is %+v, want the pasted one", got.Session)
	}
}

func TestANativeIndexerIsAddedWithItsPortsFieldsAddressAndLimit(t *testing.T) {
	w := newNativeWorld(t, livePorts)
	v := w.add(indexers.Draft{Kind: indexers.Native, Name: "Lantern", Port: "lantern", Values: map[string]string{"passkey": "pk1", "codecs": "1,5", "pages": "3"}})
	p, ok := v.Source.(indexers.Port)
	if !ok || p.ID != "lantern" || p.Address != "https://lantern.test/" || !reflect.DeepEqual(p.Plain, map[string]string{"codecs": "1,5", "pages": "3"}) {
		t.Fatalf("the Indexer's source is %+v", v.Source)
	}
	if v.Searches != (indexers.Limit{Count: 150, Per: indexers.Hour}) || v.SignsIn {
		t.Errorf("searches %+v and signs in %v, want the port's 150 an hour and stateless", v.Searches, v.SignsIn)
	}
	j := w.node.last()
	if j.Kind != searching.Native || j.Port != "lantern" || j.Address != "https://lantern.test/" ||
		!reflect.DeepEqual(j.Settings, map[string]string{"passkey": "pk1", "codecs": "1,5", "pages": "3"}) {
		t.Errorf("the Test ran %+v", j)
	}

	mirror := native("Mirror", "lantern")
	mirror.BaseAddress, mirror.Searches = "https://lantern.example/", indexers.Limit{Count: 9, Per: indexers.Day}
	typed := w.add(mirror)
	if p, _ := typed.Source.(indexers.Port); p.Address != "https://lantern.example/" || typed.Searches.Count != 9 {
		t.Errorf("a typed address and limit saved %+v, %+v", p, typed.Searches)
	}

	ember := w.add(indexers.Draft{Kind: indexers.Native, Name: "Ember", Port: "ember", Cookie: &indexers.PastedCookie{Cookie: "uid=1; pass=2", UserAgent: "Firefox"}})
	if !ember.SignsIn || ember.Session == nil || !ember.Session.Pasted {
		t.Errorf("a pasted-cookie port signs in %v with session %+v", ember.SignsIn, ember.Session)
	}
}

func TestANativeDraftThatDoesntFitItsPortIsRefused(t *testing.T) {
	w := newNativeWorld(t, livePorts)
	for _, tc := range []struct {
		name string
		d    indexers.Draft
		want error
	}{
		{"an unknown port", native("Nowhere", "nowhere"), indexers.ErrNoPort},
		{"a catalog id a port replaces", native("Lost", "larkspur"), indexers.ErrNoPort},
		{"an address that isn't http", indexers.Draft{Kind: indexers.Native, Name: "Ftp", Port: "lantern", BaseAddress: "ftp://lantern.test/"}, indexers.ErrNotAnAddress},
		{"an option the multiselect lacks", indexers.Draft{Kind: indexers.Native, Name: "Bad", Port: "lantern", Values: map[string]string{"passkey": "pk", "codecs": "1,9"}}, errValue},
		{"a number past its bound", indexers.Draft{Kind: indexers.Native, Name: "Bad", Port: "lantern", Values: map[string]string{"passkey": "pk", "pages": "6"}}, errValue},
		{"no passkey", indexers.Draft{Kind: indexers.Native, Name: "Bad", Port: "lantern"}, errValue},
		{"an empty passkey", indexers.Draft{Kind: indexers.Native, Name: "Bad", Port: "lantern", Values: map[string]string{"passkey": ""}}, errValue},
		{"an empty username", indexers.Draft{Kind: indexers.Native, Name: "Bad", Port: "quill", Values: map[string]string{"username": "", "password": "pw"}}, errValue},
		{"a cookie for a site with no cookie", indexers.Draft{Kind: indexers.Native, Name: "Bad", Port: "lantern", Cookie: &indexers.PastedCookie{Cookie: "a=b", UserAgent: "Firefox"}}, indexers.ErrNoCookieSignIn},
		{"a cookie site with none", native("Ember", "ember"), indexers.ErrCookieNeeded},
		{"a cookie without its User-Agent", indexers.Draft{Kind: indexers.Native, Name: "Ember", Port: "ember", Cookie: &indexers.PastedCookie{Cookie: "a=b"}}, indexers.ErrUserAgentNeeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := w.x.Create(t.Context(), identity.Admin{}, tc.d, indexers.Typed{})
			var value *indexerfields.ValueError
			if misfit := errors.Is(tc.want, errValue); misfit && !errors.As(err, &value) || !misfit && !errors.Is(err, tc.want) {
				t.Errorf("Create = %v, want %v", err, tc.want)
			}
		})
	}

}

func TestAPortsFormSaysWhatItsSiteTakes(t *testing.T) {
	w := newNativeWorld(t, livePorts)
	f, err := w.x.Form(t.Context(), identity.Admin{}, "lantern")
	if err != nil {
		t.Fatal(err)
	}
	var fields []string
	for _, field := range f.Fields {
		fields = append(fields, field.Name+":"+string(field.Type)+":"+map[bool]string{true: "sealed", false: "plain"}[field.Sealed])
	}
	if got := strings.Join(fields, " "); got != "passkey:text:sealed codecs:multiselect:plain pages:number:plain" {
		t.Errorf("Lantern's fields are %s", got)
	}
	want := indexers.Ported{Searches: indexers.Limit{Count: 150, Per: indexers.Hour}, Grabs: indexers.Limit{}, Recent: true, Spacing: 4 * time.Second, Synced: synced}
	if f.Port == nil || *f.Port != want || f.Session || f.Cookie || f.Code || !reflect.DeepEqual(f.Links, []string{"https://lantern.test/", "https://lantern-mirror.test/"}) {
		t.Errorf("Lantern's form is %+v with %+v", f, f.Port)
	}
	q, err := w.x.Form(t.Context(), identity.Admin{}, "quill")
	if err != nil || !q.Code || !q.Session || q.Port.Spacing != port.Floor || q.Fields[len(q.Fields)-1].Name != "authenticatorKey" {
		t.Errorf("Quill's form is %+v, %v", q, err)
	}
	e, err := w.x.Form(t.Context(), identity.Admin{}, "ember")
	if err != nil || !e.Cookie || !e.Port.UserAgent || len(e.Fields) != 0 {
		t.Errorf("Ember's form is %+v, %v", e, err)
	}
}

func TestAPortsTrackerIsKeyedByItsIDAndEveryCatalogIDItClaims(t *testing.T) {
	w := newNativeWorld(t, livePorts)
	lone := w.add(native("Quill", "quill"))
	if tr := w.trackerOf(lone); tr.Name != "Quill" || tr.Privacy != indexers.Private || !reflect.DeepEqual(tr.Keys, []string{"definition:quill"}) {
		t.Errorf("Quill's Tracker is %+v", tr)
	}

	w.refresh()
	larkspur := w.add(indexers.Draft{Kind: indexers.Catalog, Name: "Larkspur", Definition: "larkspur", Values: map[string]string{"username": "ada", "password": "pw"}})
	v := w.add(indexers.Draft{Kind: indexers.Native, Name: "Lantern", Port: "lantern", Values: map[string]string{"passkey": "pk1"}})
	tr := w.trackerOf(v)
	if tr.ID != w.trackerOf(larkspur).ID || tr.Name != "Larkspur" {
		t.Errorf("Lantern reaches %+v, want the Tracker of the catalog's Larkspur it replaces", tr)
	}
	if !reflect.DeepEqual(tr.Keys, []string{"definition:lantern", "definition:larkspur", "definition:larkspur-old"}) {
		t.Errorf("the Tracker's keys are %v", tr.Keys)
	}
}

func TestATwoFactorCodeGoesWithTheTestAndItsStopWaitsForTheAdmin(t *testing.T) {
	w := newNativeWorld(t, livePorts)
	w.node.answering(func(j searching.Job) searching.Answer {
		if c, _ := j.Call.(searching.TestCall); c.Code == "123456" {
			return searching.Answer{Report: &searching.Report{}}
		}
		return searching.Answer{Outcome: &searching.Stopped{Cause: searching.CodeNeeded}}
	})
	d := indexers.Draft{Kind: indexers.Native, Name: "Quill", Port: "quill", Values: map[string]string{"username": "ada", "password": "pw"}, On: true}
	_, err := w.x.Create(t.Context(), identity.Admin{}, d, indexers.Typed{})
	var failed *indexers.TestFailedError
	if !errors.As(err, &failed) || failed.Sentence != "Quill needs a two-factor code to sign in. Type the current code and test again." {
		t.Fatalf("Create without a code = %v", err)
	}
	v, err := w.x.Create(t.Context(), identity.Admin{}, d, indexers.Typed{Code: "123456"})
	if err != nil {
		t.Fatalf("Create with the code = %v", err)
	}
	if !v.SignsIn {
		t.Error("a two-factor port doesn't sign in")
	}

	if _, err := w.x.Test(t.Context(), identity.Admin{}, v.ID, indexers.Typed{}); !errors.As(err, &failed) {
		t.Fatalf("Test without a code = %v", err)
	}
	stopped := w.view(w.x, v.ID)
	if stopped.Status.Cause != searching.CodeNeeded || stopped.Sentence != "Stopped because Quill needs a two-factor code to sign in again." ||
		stopped.Fix != "Test the Indexer and type the current code." {
		t.Errorf("the stopped Indexer says %q, %q", stopped.Sentence, stopped.Fix)
	}
	item, ok := w.healthText(w.x, v.ID)
	if !ok || item.Text != "Quill needs a two-factor code to sign in again. Open the Indexer and type the current code." ||
		item.Action != health.LinkAction("/settings/indexers/"+v.ID.String()+"#code", "Type code") {
		t.Errorf("Health says %+v", item.Says)
	}
	if indexers.Retried(searching.CodeNeeded) || !indexers.NodeBound(searching.CodeNeeded) {
		t.Error("code_needed is retried by the health check or isn't node-bound")
	}
	if got := w.x.Line(indexerstest.Type, &indexers.Outcome{Kind: indexers.IsStopped, Cause: searching.CodeNeeded}); got != "Stopped because signing in again needs a two-factor code." {
		t.Errorf("its search line is %q", got)
	}
	if _, err := w.x.Test(t.Context(), identity.Admin{}, v.ID, indexers.Typed{Code: "123456"}); err != nil {
		t.Fatalf("Test with the code = %v", err)
	}

	w.node.answering(nil)
	lantern := w.add(native("Lantern", "lantern"))
	if _, err := w.x.Test(t.Context(), identity.Admin{}, lantern.ID, indexers.Typed{Code: "123456"}); !errors.Is(err, indexers.ErrNoCode) {
		t.Errorf("a code for a site with no two-factor sign-in = %v", err)
	}
}

func TestAPortsStopsNameItsOwnCredentials(t *testing.T) {
	for _, tc := range []struct {
		port, text   string
		cause        searching.StopCause
		form, status string
		fix, item    string
	}{
		{
			"lantern", "", searching.AccountRefused, "Lantern refused the passkey. Check the account, then test again.",
			"Stopped because Lantern refused the passkey.", "Check the account, then test the Indexer.",
			"Lantern refused the passkey. Check the account, then test the Indexer.",
		},
		{
			"lantern", "Invalid Passkey", searching.AccountRefused, "Lantern refused the passkey with 'Invalid Passkey'. Check the account, then test again.",
			"Stopped because Lantern refused the passkey with 'Invalid Passkey'.", "Check the account, then test the Indexer.",
			"Lantern refused the passkey with 'Invalid Passkey'. Check the account, then test the Indexer.",
		},
		{
			"speed", "", searching.CredentialsRejected, "Speed rejected the email or password. Check them and test again.",
			"Stopped because Speed rejected the email or password.", "Update the email or password with Edit. Saving tests them.",
			"Speed rejected the email or password. Update them on the Indexer and test it.",
		},
		{
			"ember", "", searching.CookieExpired, "Ember no longer accepts the browser cookie. Paste a new one with that browser's User-Agent and test again.",
			"Stopped because Ember no longer accepts the browser cookie.", "Paste a new browser cookie and its User-Agent with Edit.",
			"Ember no longer accepts the browser cookie. Paste a new one and that browser's User-Agent on the Indexer.",
		},
	} {
		t.Run(tc.port+" "+string(tc.cause), func(t *testing.T) {
			w := newNativeWorld(t, livePorts)
			e, _ := livePorts.Lookup(port.ID(tc.port))
			d := native(e.Name, e.ID)
			d.On = true
			if e.Auth == port.PastedCookie {
				d.Cookie = &indexers.PastedCookie{Cookie: "uid=1", UserAgent: "Firefox"}
			}
			v := w.add(d)
			w.node.answering(stopping(tc.cause, tc.text))
			_, err := w.x.Test(t.Context(), identity.Admin{}, v.ID, indexers.Typed{})
			var failed *indexers.TestFailedError
			if !errors.As(err, &failed) || failed.Sentence != tc.form {
				t.Errorf("the form says %v, want %q", err, tc.form)
			}
			got := w.view(w.x, v.ID)
			if got.Sentence != tc.status || got.Fix != tc.fix {
				t.Errorf("the Indexer page says %q, %q", got.Sentence, got.Fix)
			}
			if item, _ := w.healthText(w.x, v.ID); item.Text != tc.item {
				t.Errorf("Health says %q", item.Text)
			}
		})
	}
}

func TestATokenIsSealedWithTheSessionAndOfferedToTheNextJob(t *testing.T) {
	w := newNativeWorld(t, livePorts)
	w.node.answering(func(searching.Job) searching.Answer {
		return searching.Answer{Report: &searching.Report{}, Session: &searching.Session{Token: "tok-1", Earned: time.Now()}}
	})
	v := w.add(indexers.Draft{Kind: indexers.Native, Name: "Speed", Port: "speed", Values: map[string]string{"email": "ada@speed.test", "password": "pw"}})
	if _, err := w.x.Test(t.Context(), identity.Admin{}, v.ID, indexers.Typed{}); err != nil {
		t.Fatal(err)
	}
	if s := w.node.last().Session; s == nil || s.Token != "tok-1" {
		t.Errorf("the next job's session is %+v, want the token", s)
	}
}

func TestANativeIndexersEditTestsTheChangedSettingsFirstAndKeepsASealedOneSentEmpty(t *testing.T) {
	w := newNativeWorld(t, livePorts)
	v := w.add(indexers.Draft{Kind: indexers.Native, Name: "Lantern", Port: "lantern", Values: map[string]string{"passkey": "pk1", "pages": "3"}})
	jobs := len(w.node.jobs)
	address := "https://lantern-mirror.test/"
	if _, err := w.x.Update(t.Context(), identity.Admin{}, v.ID, indexers.Edit{BaseAddress: &address, Values: map[string]string{"passkey": "", "codecs": "5"}}, indexers.Typed{}); err != nil {
		t.Fatal(err)
	}
	j := w.node.last()
	if len(w.node.jobs) != jobs+1 || j.Address != address || !reflect.DeepEqual(j.Settings, map[string]string{"passkey": "pk1", "pages": "3", "codecs": "5"}) {
		t.Errorf("the edit's Test ran %+v", j)
	}
	if got := w.view(w.x, v.ID).Sealed; !reflect.DeepEqual(got, []string{"passkey"}) {
		t.Errorf("the sealed settings set are %v, want the kept passkey", got)
	}

	w.node.answering(stopping(searching.AccountRefused, ""))
	if _, err := w.x.Update(t.Context(), identity.Admin{}, v.ID, indexers.Edit{Values: map[string]string{"passkey": "pk2"}}, indexers.Typed{}); !errors.As(err, new(*indexers.TestFailedError)) {
		t.Fatalf("an edit whose Test fails = %v", err)
	}
	w.node.answering(nil)
	if _, err := w.x.Test(t.Context(), identity.Admin{}, v.ID, indexers.Typed{}); err != nil {
		t.Fatal(err)
	}
	if got := w.node.last().Settings["passkey"]; got != "pk1" {
		t.Errorf("a failed edit saved the passkey %q", got)
	}
	bad := "lantern.test"
	if _, err := w.x.Update(t.Context(), identity.Admin{}, v.ID, indexers.Edit{BaseAddress: &bad}, indexers.Typed{}); !errors.Is(err, indexers.ErrNotAnAddress) {
		t.Errorf("an edit to a bare host = %v", err)
	}
}

func TestARetiredPortStopsItsIndexersAndKeepsTheirTracker(t *testing.T) {
	w := newNativeWorld(t, livePorts)
	v := w.add(indexers.Draft{Kind: indexers.Native, Name: "Ghost", Port: "ghost", Values: map[string]string{"apiKey": "k"}})
	tracker := w.trackerOf(v)

	node := searching.RemoteOf(searching.New(searching.Config{
		HTTP: &http.Client{}, Timings: searching.DefaultTimings(), Log: slog.New(slog.DiscardHandler), Native: port.Driver(laterPorts),
	}))
	later := w.with(laterPorts, node)
	_, err := later.Test(t.Context(), identity.Admin{}, v.ID, indexers.Typed{})
	var failed *indexers.TestFailedError
	if !errors.As(err, &failed) || failed.Sentence != "SlipStream no longer supports Ghost, because the site closed. Delete the Indexer." {
		t.Fatalf("Test of a retired port's Indexer = %v", err)
	}
	got := w.view(later, v.ID)
	if got.Status.State != indexers.Stopped || got.Status.Cause != searching.PortRetired ||
		got.Sentence != "SlipStream no longer supports Ghost, because the site closed." || got.Fix != "Delete the Indexer." {
		t.Errorf("the retired port's Indexer stands %+v, saying %q, %q", got.Status, got.Sentence, got.Fix)
	}
	if p, _ := got.Source.(indexers.Port); p.Retired == nil || p.Site() != "Ghost" {
		t.Errorf("its port reads %+v", p)
	}
	if item, _ := w.healthText(later, v.ID); item.Text != "SlipStream no longer supports Ghost, because the site closed. Delete the Indexer." {
		t.Errorf("Health says %q", item.Text)
	}
	if indexers.Retried(searching.PortRetired) {
		t.Error("the health check retries a retired port")
	}
	if !slices.Equal(got.Trackers, []indexers.TrackerID{tracker.ID}) {
		t.Errorf("the retired port's Indexer reaches %v, want its Tracker %s", got.Trackers, tracker.ID)
	}
}

func TestAMovedPortsIndexersBecomeCatalogIndexersWithTheirTrackerAndHistory(t *testing.T) {
	w := newNativeWorld(t, livePorts)
	v := w.add(indexers.Draft{Kind: indexers.Native, Name: "Fern", Port: "fern", Values: map[string]string{"apiKey": "k1"}})
	tracker := w.trackerOf(v)

	notYet := w.with(port.NewRegistry([]port.Port{lantern}, nil, []port.Moved{{ID: "fern", Definition: "nowhere"}}), w.node)
	w.refreshWith(notYet)
	if got := w.view(notYet, v.ID); got.Kind != indexers.Native {
		t.Fatalf("Fern moved to %+v though the catalog holds no definition nowhere", got.Source)
	}

	later := w.with(laterPorts, w.node)
	ctx, cancel := context.WithCancel(t.Context())
	ran := make(chan error, 1)
	go func() { ran <- later.Run(ctx) }()
	got := w.view(later, v.ID)
	for deadline := time.Now().Add(5 * time.Second); got.Kind == indexers.Native && time.Now().Before(deadline); got = w.view(later, v.ID) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := <-ran; err != nil {
		t.Fatalf("Run = %v", err)
	}
	def, ok := got.Source.(indexers.Definition)
	if !ok || got.Kind != indexers.Catalog || def.ID != "quarry" || def.BaseAddress != "https://quarry.fixture-site.fake.localhost/" || got.Revision != v.Revision+1 {
		t.Fatalf("once the release that moves it runs, Fern is %s on %+v at revision %d", got.Kind, got.Source, got.Revision)
	}
	if !slices.Equal(got.Notes, []string{"Fern now runs from its catalog definition. It kept its settings, Tracker and history."}) {
		t.Errorf("its notes are %v", got.Notes)
	}
	moved := w.trackerOf(got)
	if moved.ID != tracker.ID || !reflect.DeepEqual(moved.Keys, []string{"definition:fern", "definition:quarry"}) {
		t.Errorf("its Tracker is %+v, want %s keyed by both ids", moved, tracker.ID)
	}
	queries, err := later.Queries(t.Context(), identity.Admin{}, v.ID, indexers.Page{Limit: 10})
	if err != nil || len(queries) != 1 {
		t.Errorf("its history is %+v, %v", queries, err)
	}
	if _, err := later.Test(t.Context(), identity.Admin{}, v.ID, indexers.Typed{}); err != nil {
		t.Fatal(err)
	}
	if j := w.node.last(); j.Definition == nil || j.Settings["apikey"] != "k1" {
		t.Errorf("its Test ran %+v, want the definition with the mapped API key", j)
	}
	w.refreshWith(later)
	if again := w.view(later, v.ID); again.Revision != got.Revision {
		t.Errorf("moving again changed it to revision %d", again.Revision)
	}
}

func TestAReleaseCarriesItsFlagsToTheController(t *testing.T) {
	p := plannedWith(t, "Alder")
	p.node.answer(p.ids["Alder"], searching.Answer{Releases: []searching.Release{{Title: "Orchard.1080p", Flags: []searching.Flag{searching.Freeleech, searching.Internal}}}})
	answers := p.search(p.plan())
	if got := answers[0].Releases[0].Flags; !slices.Equal(got, []searching.Flag{searching.Freeleech, searching.Internal}) {
		t.Errorf("the Release's flags are %v", got)
	}
}
