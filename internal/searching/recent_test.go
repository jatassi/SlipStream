package searching_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/internal/searching"
)

// recorder is a site that answers body and records each request's query.
type recorder struct {
	mu   sync.Mutex
	sent []url.Values
}

func (rec *recorder) serve(t *testing.T, body func(url.Values) string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		rec.sent = append(rec.sent, r.URL.Query())
		rec.mu.Unlock()
		_, _ = w.Write([]byte(body(r.URL.Query())))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (rec *recorder) requests() []url.Values {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return slices.Clone(rec.sent)
}

func guids(rs []searching.Release) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.GUID)
	}
	return out
}

func TestTorznabRecentPagesByOffsetAndCarriesEachGUID(t *testing.T) {
	var rec recorder
	srv := rec.serve(t, func(q url.Values) string {
		if q.Get("t") == "caps" {
			return caps
		}
		return `<rss><channel>` +
			`<item><title>Orchard.2021</title><guid>orchard-guid</guid><link>http://x.test/1</link></item>` +
			`<item><title>Meadow.2020</title><link>http://x.test/2</link></item></channel></rss>`
	})
	n := node(t, &bytes.Buffer{})
	a := run(t, n, srv.URL, "k", searching.RecentCall{Offset: 200})
	if outcome(a) != "ok" || !slices.Equal(guids(a.Releases), []string{"orchard-guid", "http://x.test/2"}) {
		t.Fatalf("the page at offset 200 = %s with GUIDs %v, want ok with the guid, else the link", outcome(a), guids(a.Releases))
	}
	sent := rec.requests()
	if len(sent) != 1 || sent[0].Get("t") != "search" || sent[0].Get("offset") != "200" || sent[0].Get("limit") != "100" {
		t.Fatalf("requests = %v, want one t=search at offset 200, limit 100", sent)
	}
	if a := run(t, n, srv.URL, "k", searching.TestCall{}); a.Report == nil || !a.Report.Capabilities.RecentPaging {
		t.Fatalf("the Test's report = %+v, want recent paging", a.Report)
	}
}

func TestAnRSSFeedListsOnlyItsNewest(t *testing.T) {
	var rec recorder
	srv := rec.serve(t, func(url.Values) string { return brookFeed })
	n := feedNode(t, srv)
	if a := feedJob(n, t, nil, searching.RecentCall{}); !slices.Equal(guids(a.Releases), []string{"1", "2"}) {
		t.Fatalf("the feed's GUIDs = %v, want 1 and 2", guids(a.Releases))
	}
	a := feedJob(n, t, nil, searching.RecentCall{Offset: 100})
	if f, ok := errors.AsType[*searching.Failing](a.Outcome); !ok || len(rec.requests()) != 1 {
		t.Fatalf("a call at offset 100 = %v after %d requests, want failing with no request", f, len(rec.requests()))
	}
	if a := feedJob(n, t, nil, searching.TestCall{}); a.Report == nil || a.Report.Capabilities.RecentPaging {
		t.Fatalf("the Test's report = %+v, want no recent paging", a.Report)
	}
}

const pagedDefinition = `id: shoal
links: [https://shoal.test/]
caps:
  categories: {1: Movies}
  modes: {search: [q]}
search:
  paths:
    - path: rss
      response: {type: xml}
  rows: {selector: item}
  fields:
    title: {selector: title}
    download: {selector: link}
    category: {text: 1}
`

func TestACatalogSiteListsOnlyItsNewest(t *testing.T) {
	var rec recorder
	srv := rec.serve(t, func(url.Values) string {
		return `<rss><channel><item><title>Orchard.2021</title><link>https://shoal.test/get/1</link></item></channel></rss>`
	})
	timings := searching.DefaultTimings()
	timings.Floor = 0
	n := searching.RemoteOf(searching.New(searching.Config{HTTP: &http.Client{}, Timings: timings, Log: slog.New(slog.DiscardHandler)}))
	sum := sha256.Sum256([]byte(pagedDefinition))
	ask := func(call searching.Call) searching.Answer {
		t.Helper()
		poll, err := n.Run(t.Context(), searching.Job{
			ID: t.Name() + "/" + strconv.Itoa(int(jobs.Add(1))), Indexer: "shoal", Kind: searching.Catalog, Address: srv.URL + "/",
			Definition: &searching.Definition{YAML: []byte(pagedDefinition), SHA: hex.EncodeToString(sum[:])}, Call: call,
		})
		if err != nil || poll.Answer == nil {
			t.Fatalf("Run = %+v, %v; want an answer within the hold", poll, err)
		}
		return *poll.Answer
	}
	if a := ask(searching.RecentCall{}); outcome(a) != "ok" || !slices.Equal(guids(a.Releases), []string{"https://shoal.test/get/1"}) {
		t.Fatalf("the newest = %s with GUIDs %v, want ok with the download link", outcome(a), guids(a.Releases))
	}
	a := ask(searching.RecentCall{Offset: 100})
	if f, ok := errors.AsType[*searching.Failing](a.Outcome); !ok || len(rec.requests()) != 1 {
		t.Fatalf("a call at offset 100 = %v after %d requests, want failing with no request", f, len(rec.requests()))
	}
	if a := ask(searching.TestCall{}); a.Report == nil || a.Report.Capabilities.RecentPaging {
		t.Fatalf("the Test's report = %+v, want no recent paging", a.Report)
	}
}
