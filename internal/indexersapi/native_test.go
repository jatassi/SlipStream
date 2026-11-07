package indexersapi_test

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
)

type codec int

func (codec) Options() []port.Option {
	return []port.Option{{Value: "1", Label: "H.264"}, {Value: "5", Label: "HEVC"}}
}

type (
	lanternSettings struct {
		port.PasskeyField
		Codecs []codec `port:"codecs" label:"Codecs"`
		Pages  int     `port:"pages" label:"Pages" min:"1" max:"5" default:"2"`
	}
	quillSettings struct{ port.UserPass }
)

func generator[S any](*port.Indexer[S], port.SearchCriteria) (port.Chain, error) {
	return port.Chain{}, nil
}

func parser[S any](*port.Indexer[S], *port.IndexerResponse) ([]port.TorrentInfo, error) {
	return nil, nil
}

func fixtureEntry(id port.ID, name string) port.Entry {
	return port.Entry{
		ID: id, Name: name, Description: name + " is a fictional tracker for tests.", Privacy: port.Private,
		IndexerURLs:  []string{"https://" + string(id) + ".test/", "https://" + string(id) + "-mirror.test/"},
		Capabilities: port.Capabilities{SearchParams: []port.Param{port.Q}, Categories: port.Categories(port.Map(1, newznab.Movies, ""))},
		Upstream:     port.Upstream{Commit: "ef9c1824", Date: time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)},
	}
}

// ports are Lantern, reached by a passkey, and Quill, which signs in with
// a two-factor code.
var ports = port.NewRegistry([]port.Port{
	func() port.Port {
		e := fixtureEntry("lantern", "Lantern")
		e.QueryLimit, e.RateLimit = port.PerHour(150), 4100*time.Millisecond
		return port.Define(e, port.Impl[lanternSettings]{Login: port.NoLogin{}, Basic: generator[lanternSettings], Parse: parser[lanternSettings]})
	}(),
	port.Define(fixtureEntry("quill", "Quill"), port.Impl[quillSettings]{
		Login: port.CodeLogin[quillSettings]{DoLogin: func(*port.SignIn[quillSettings]) error { return nil }},
		Basic: generator[quillSettings], Parse: parser[quillSettings],
	}),
}, nil, nil)

// nativeNode answers native jobs: a Test passes unless it needs a code and
// carries none.
type nativeNode struct {
	mu   sync.Mutex
	jobs []searching.Job
}

func (*nativeNode) ReachSolver(context.Context, searching.Solver) (int, error) { return 0, nil }

func (n *nativeNode) Run(_ context.Context, j searching.Job) (searching.Poll, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.jobs = append(n.jobs, j)
	a := searching.Answer{Started: time.Now(), Report: &searching.Report{Capabilities: searching.Capabilities{Modes: map[string][]string{"search": {"q"}}}}}
	if call, ok := j.Call.(searching.TestCall); ok && j.Port == "quill" && call.Code == "" {
		a = searching.Answer{Started: a.Started, Outcome: &searching.Stopped{Cause: searching.CodeNeeded}}
	}
	return searching.Poll{Started: a.Started, Answer: &a}, nil
}

func (n *nativeNode) last() searching.Job {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.jobs[len(n.jobs)-1]
}

// split sends native jobs to native and every other job to other.
type split struct {
	native *nativeNode
	other  searching.Remote
}

func (s split) ReachSolver(ctx context.Context, sv searching.Solver) (int, error) {
	return s.other.ReachSolver(ctx, sv)
}

func (s split) Run(ctx context.Context, j searching.Job) (searching.Poll, error) {
	if j.Kind == searching.Native {
		return s.native.Run(ctx, j)
	}
	return s.other.Run(ctx, j)
}

const invalid = `{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept",`

func TestAPortsFormAndCatalogEntry(t *testing.T) {
	s := serve(t, indexerstest.Alone(self), role.NodeID{})
	s.owner.Expect(t, http.MethodGet, "/api/v1/indexer-catalog/lantern", "", http.StatusOK,
		`{"entry":{"id":"lantern","name":"Lantern","description":"Lantern is a fictional tracker for tests.","language":"en-US","privacy":"private","categories":[2000],"origin":"native","reason":null,"replacedByCustom":false,"version":"T"},`+
			`"links":["https://lantern.test/","https://lantern-mirror.test/"],"fields":[`+
			`{"name":"passkey","label":"Passkey","type":"text","default":null,"options":[],"min":null,"max":null,"help":null,"sealed":true},`+
			`{"name":"codecs","label":"Codecs","type":"multiselect","default":null,"options":[{"value":"1","label":"H.264"},{"value":"5","label":"HEVC"}],"min":null,"max":null,"help":null,"sealed":false},`+
			`{"name":"pages","label":"Pages","type":"number","default":"2","options":[],"min":1,"max":5,"help":null,"sealed":false}],`+
			`"session":false,"captcha":false,"solver":false,"cookie":false,"code":false,`+
			`"port":{"userAgent":false,"searches":{"count":150,"per":"hour"},"grabs":{"count":0,"per":"hour"},"recent":true,"spacingSeconds":4.1,"synced":"T"}}`)
	_, quill := s.owner.Call(t, http.MethodGet, "/api/v1/indexer-catalog/quill", "")
	if !strings.Contains(quill, `"session":true,"captcha":false,"solver":false,"cookie":false,"code":true,`) || !strings.Contains(quill, `"name":"authenticatorKey"`) {
		t.Errorf("Quill's form is %s", quill)
	}
	_, list := s.owner.Call(t, http.MethodGet, "/api/v1/indexer-catalog", "")
	if !strings.Contains(list, `"entries":[{"id":"lantern",`) || !strings.Contains(list, `{"id":"quill","name":"Quill",`) {
		t.Errorf("before any refresh the catalog lists %s, want the ports", list)
	}
}

func TestANativeIndexerIsAddedFromItsPortAndShownWithIt(t *testing.T) {
	s := serve(t, indexerstest.Alone(self), role.NodeID{})
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", `{"kind":"native","name":"Lantern","port":"lantern","values":{"passkey":"pk1","codecs":"1,5"}}`, http.StatusCreated,
		`{"id":"id1","name":"Lantern","kind":"native","homeNodeId":"id2","on":true,"automaticSearch":true,"rss":true,"allowFailover":false,"priority":25,"categories":null,"media":null,"allowNoSize":false,`+
			`"searches":{"count":150,"per":"hour","used":0,"freeAt":null},"grabs":{"count":0,"per":"hour","used":0,"freeAt":null},`+
			`"status":{"state":"ok","cause":null,"text":null,"until":null,"since":null,"nodeId":null,"solverTried":false,"rung":0,"lastFailure":null,"successAt":"T"},`+
			`"sentence":"Lantern is working.","fix":null,"signsIn":false,"trackerIds":["id3"],"waits":null,"session":null,"feedHost":null,"definition":null,`+
			`"port":{"id":"lantern","name":"Lantern","synced":"T","baseAddress":"https://lantern.test/","values":{"codecs":"1,5"}},`+
			`"searchModes":[{"mode":"search","params":["q"]}],"notes":[],"revision":1,"createdAt":"T"}`)
	if j := s.native.last(); j.Port != "lantern" || j.Settings["passkey"] != "pk1" {
		t.Errorf("the Test ran %+v", j)
	}

	for _, tc := range []struct{ body, want string }{
		{`{"kind":"native","name":"Nowhere","port":"nowhere"}`, invalid + `"detail":"Choose one of the sites SlipStream includes.","errors":[{"location":"body.port","message":"Choose one of the sites SlipStream includes."}]}`},
		{`{"kind":"native","name":"Bare","port":"lantern","baseAddress":"lantern.test"}`, invalid + `"detail":"Choose one of the site's addresses, or type one that starts with http:// or https://.","errors":[{"location":"body.baseAddress","message":"Choose one of the site's addresses, or type one that starts with http:// or https://."}]}`},
		{`{"kind":"native","name":"Bad","port":"lantern","values":{"codecs":"1,9"}}`, invalid + `"detail":"Choose from the options for Codecs.","errors":[{"location":"body.values.codecs","message":"Choose from the options for Codecs."}]}`},
		{`{"kind":"native","name":"Bad","port":"lantern","values":{"pages":"9"}}`, invalid + `"detail":"Pages is a whole number from 1 to 5.","errors":[{"location":"body.values.pages","message":"Pages is a whole number from 1 to 5."}]}`},
		{`{"kind":"native","name":"Bad","port":"lantern","cookie":{"cookie":"a=b"}}`, invalid + `"detail":"Only a site that signs in with a browser cookie takes one.","errors":[{"location":"body.cookie","message":"Only a site that signs in with a browser cookie takes one."}]}`},
		{`{"kind":"native","name":"Ported","address":"https://lantern.test/"}`, invalid + `"errors":[{"location":"body.port","message":"Choose the site."},{"location":"body.address","message":"A ported site takes its address as baseAddress."}]}`},
	} {
		s.owner.Expect(t, http.MethodPost, "/api/v1/indexers", tc.body, http.StatusUnprocessableEntity, tc.want)
	}

	_, list := s.owner.Call(t, http.MethodGet, "/api/v1/indexers", "")
	id := ownertest.FirstID(list)
	_, edited := s.owner.Call(t, http.MethodPatch, "/api/v1/indexers/"+id, `{"baseAddress":"https://lantern.example/","values":{"passkey":"","pages":"4"}}`)
	if !strings.Contains(edited, `"baseAddress":"https://lantern.example/","values":{"codecs":"1,5","pages":"4"}`) || !strings.Contains(edited, `"revision":2`) {
		t.Errorf("the edit answered %s", edited)
	}
	if j := s.native.last(); j.Address != "https://lantern.example/" || j.Settings["passkey"] != "pk1" || j.Settings["pages"] != "4" {
		t.Errorf("the edit's Test ran %+v, want the saved passkey kept", j)
	}
}

func TestATwoFactorCodeGoesWithTheTestsAndTheEdit(t *testing.T) {
	s := serve(t, indexerstest.Alone(self), role.NodeID{})
	body := `{"kind":"native","name":"Quill","port":"quill","values":{"username":"ada","password":"pw"}`
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers/test", body+`}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"indexer_test_failed","title":"The Indexer's Test failed","detail":"Quill needs a two-factor code to sign in again. Type the current code and test again."}`)
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers/test", body+`,"code":"123456"}`, http.StatusOK, `{"searchModes":[{"mode":"search","params":["q"]}],"items":0,"sources":[]}`)
	code, reply := s.owner.Call(t, http.MethodPost, "/api/v1/indexers", body+`,"code":"123456"}`)
	if code != http.StatusCreated {
		t.Fatalf("createIndexer with the code = %d %s", code, reply)
	}
	id := ownertest.FirstID(reply)
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers/"+id+"/test", `{"code":"654321"}`, http.StatusOK, `{"searchModes":[{"mode":"search","params":["q"]}],"items":0,"sources":[]}`)
	if c, _ := s.native.last().Call.(searching.TestCall); c.Code != "654321" {
		t.Errorf("the Test carried %+v", c)
	}
	if code, reply := s.owner.Call(t, http.MethodPatch, "/api/v1/indexers/"+id, `{"values":{"password":"new"},"code":"111111"}`); code != http.StatusOK {
		t.Fatalf("an edit with the code = %d %s", code, reply)
	}
	if j := s.native.last(); j.Call != searching.Call(searching.TestCall{Code: "111111"}) || j.Settings["password"] != "new" {
		t.Errorf("the edit's Test ran %+v", j)
	}

	torznab := s.create(t)
	refused := invalid + `"detail":"Only a site that signs in with a two-factor code takes one.","errors":[{"location":"body.code","message":"Only a site that signs in with a two-factor code takes one."}]}`
	s.owner.Expect(t, http.MethodPost, "/api/v1/indexers/"+torznab+"/test", `{"code":"123456"}`, http.StatusUnprocessableEntity, refused)
	s.owner.Expect(t, http.MethodPatch, "/api/v1/indexers/"+torznab, `{"apiKey":"`+s.key+`","code":"123456"}`, http.StatusUnprocessableEntity, refused)
}
