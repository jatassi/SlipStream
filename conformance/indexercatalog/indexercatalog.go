// Package indexercatalog is the conformance suite for the Indexer catalog:
// it reads a catalog with SlipStream's own client, against fake-catalog on
// every PR and against Prowlarr's catalog in the online job, so a fake
// that drifts from the real layout fails.
package indexercatalog

import (
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/indexers"
)

// request is how long one request to the catalog may take: package.zip
// is over a megabyte.
const request = 2 * time.Minute

type Target struct {
	// Base is the catalog's base address, to which the client adds the
	// schema.
	Base string
	// Client trusts the run CA for the fake and the system roots online.
	Client *http.Client
}

// Report is what Run read: each definition by file, and what Load made
// of each.
type Report struct {
	Files map[string][]byte
	// Classes are the login classes of the definitions that load, by id.
	Classes map[string]cardigann.LoginClass
	// Refused are why SlipStream can't run the rest, by id.
	Refused map[string]string
}

// Run reads the whole schema 12 catalog at the target, each file checked
// against its index sha, and loads every definition. Each one loads or is
// listed with the reason SlipStream refuses it. Each one that loads builds
// the first request of a search for the newest items, with nothing sent.
func Run(t *testing.T, tgt Target) Report {
	t.Helper()
	index, files, err := indexers.ReadCatalog(t.Context(), tgt.Client, tgt.Base, request)
	if err != nil {
		t.Fatalf("ReadCatalog(%s): %v", tgt.Base, err)
	}
	if len(index) == 0 {
		t.Fatalf("the catalog at %s lists no definitions", tgt.Base)
	}
	r := Report{Files: files, Classes: map[string]cardigann.LoginClass{}, Refused: map[string]string{}}
	clock := cardigann.Clock{Now: time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC), Location: time.UTC}
	for _, e := range index {
		d, err := cardigann.Load(files[e.File])
		if err != nil {
			r.Refused[e.ID] = err.Error()
			continue
		}
		s := d.Summary()
		if s.ID != e.ID {
			t.Errorf("%s: the index lists id %s, and the file says %s", e.File, e.ID, s.ID)
		}
		r.Classes[e.ID] = s.Login
		req, ok := d.Recent(cardigann.Settings{}, clock).Next()
		if !ok || req.URL == nil || req.URL.Host == "" {
			t.Errorf("%s: the search for the newest items builds no request", e.ID)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(r.Refused)) {
		t.Logf("%s: SlipStream can't run it: %s", id, r.Refused[id])
	}
	sessions := 0
	for _, c := range r.Classes {
		if c == cardigann.SessionLogin {
			sessions++
		}
	}
	t.Logf("%d definitions: %d load, %d of them holding a login session; %d refused", len(index), len(r.Classes), sessions, len(r.Refused))
	return r
}
