// Package ports is the conformance suite for the public ported sites: it
// runs each port through the Node's searching, as the Controller sends a
// job, against fixture-site's replay of the port's fixture on every PR
// and against the site itself in the online job, so a fixture that drifts
// from the live site fails.
package ports

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/conformance/internal/shape"
	"github.com/jatassi/slipstream/internal/ports/all"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/searching"
)

//go:embed shape.json
var shapeJSON []byte

type Target struct {
	Port port.ID
	// Address is the Indexer's address: the port's fixture host against
	// fixture-site, the site's own online.
	Address string
	// Client reaches fixture-site through the run CA, or the network
	// online.
	Client  *http.Client
	Timings searching.Timings
	// Search is a search the target has Releases for.
	Search searching.Query
	// Cue, when set, readies the target for the call Run sends next,
	// "search" or "recent": fixture-site's case of that name.
	Cue func(call string)
	// Record, when set, takes every body in place of the check against
	// shape.json; the online suite's -update sets it to rewrite the file.
	Record func(key string, body []byte)
}

// Run sends the port its search and, when the site lists new uploads,
// Recent, and asserts what holds on both sides: each answers Releases,
// at least one, each with a title, a size, a date and a link or magnet
// that parse, and no negative peer counts. It then checks every body the
// site sent against shape.json: keys and types, no values.
func Run(t *testing.T, tgt Target) {
	t.Helper()
	e, ok := entry(tgt.Port)
	if !ok {
		t.Fatalf("the registry has no port %s", tgt.Port)
	}
	rec := shape.NewRecorder(tgt.Client.Transport, func(path string) string { return string(tgt.Port) + " " + path })
	client := *tgt.Client
	client.Transport = rec
	reg := all.Registry()
	node := searching.RemoteOf(searching.New(searching.Config{
		HTTP: &client, Timings: tgt.Timings, Log: slog.New(slog.DiscardHandler), Native: port.Driver(reg),
	}))
	calls := []string{"search"}
	if !e.Capabilities.NoRss {
		calls = append(calls, "recent")
	}
	for _, name := range calls {
		if tgt.Cue != nil {
			tgt.Cue(name)
		}
		var call searching.Call = searching.SearchCall{Query: tgt.Search}
		if name == "recent" {
			call = searching.RecentCall{}
		}
		job := searching.Job{
			ID: string(tgt.Port) + "-" + name, Indexer: "conformance", Revision: 1, Kind: searching.Native,
			Port: string(tgt.Port), Address: tgt.Address, Call: call,
		}
		a := answer(t, node, job)
		if a.Outcome != nil {
			t.Errorf("%s %s: %v", tgt.Port, name, a.Outcome)
			continue
		}
		if len(a.Releases) == 0 {
			t.Errorf("%s %s found no Releases", tgt.Port, name)
		}
		for _, r := range a.Releases {
			if problem := check(r); problem != "" {
				t.Errorf("%s %s: Release %q %s", tgt.Port, name, r.Title, problem)
			}
		}
	}
	shape.Verify(t, entries(tgt.Port, jsonBodies(rec.Bodies())), shapeJSON, tgt.Record)
}

// keyedByTitle are the ports whose site answers one object keyed by each
// Release's title, as SubsPlease does.
var keyedByTitle = []port.ID{"subsplease"}

// entries splits a title-keyed answer into one body per entry, under the
// key "{}", so the shape holds an entry's fields and not the week's titles.
func entries(id port.ID, bodies []shape.Body) []shape.Body {
	if !slices.Contains(keyedByTitle, id) {
		return bodies
	}
	var split []shape.Body
	for _, b := range bodies {
		var byTitle map[string]json.RawMessage
		if json.Unmarshal(b.JSON, &byTitle) != nil || len(byTitle) == 0 {
			split = append(split, b)
			continue
		}
		for _, entry := range byTitle {
			split = append(split, shape.Body{Key: b.Key, JSON: []byte(`{"{}":` + string(entry) + `}`)})
		}
	}
	return split
}

// jsonBodies are the bodies a JSON API sent; an HTML site such as
// Anidex has no shape to keep.
func jsonBodies(bodies []shape.Body) []shape.Body {
	return slices.DeleteFunc(bodies, func(b shape.Body) bool {
		t := bytes.TrimSpace(b.JSON)
		return len(t) == 0 || (t[0] != '{' && t[0] != '[')
	})
}

func entry(id port.ID) (port.Entry, bool) {
	for _, e := range all.Registry().Entries() {
		if e.ID == id {
			return e, true
		}
	}
	return port.Entry{}, false
}

// answer runs job until the Node answers it.
func answer(t *testing.T, node searching.Remote, job searching.Job) searching.Answer {
	t.Helper()
	for {
		poll, err := node.Run(t.Context(), job)
		if err != nil {
			t.Fatalf("Run %s = %v", job.ID, err)
		}
		if poll.Answer != nil {
			return *poll.Answer
		}
	}
}

// check is what is wrong with r, or "".
func check(r searching.Release) string {
	switch {
	case r.Title == "":
		return "has no title"
	case r.Size <= 0:
		return "has no size"
	case r.Published.IsZero():
		return "has no date"
	case r.Seeders < 0 || r.Leechers < 0:
		return "has a negative peer count"
	case r.Link.Reveal() == "" && r.Magnet.Reveal() == "":
		return "has neither a link nor a magnet"
	}
	for _, link := range []string{r.Link.Reveal(), r.Magnet.Reveal(), r.Details} {
		if u, err := url.Parse(link); link != "" && (err != nil || u.Scheme == "") {
			return "has a link that isn't a URL: " + link
		}
	}
	return ""
}
