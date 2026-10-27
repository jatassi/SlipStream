package searching_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/searching"
)

const caps = `<caps><searching><search available="yes" /></searching></caps>`

// node is a Node's Searching service with a 1s request time, no spacing,
// and its debug log in log.
func node(t *testing.T, log *bytes.Buffer) searching.Remote {
	t.Helper()
	timings := searching.DefaultTimings()
	timings.Request, timings.Floor = time.Second, 0
	return searching.RemoteOf(searching.New(searching.Config{
		HTTP: &http.Client{}, Timings: timings, Log: slog.New(slog.NewTextHandler(log, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}))
}

var jobs atomic.Int32

func run(t *testing.T, n searching.Remote, address, key string, call searching.Call) searching.Answer {
	t.Helper()
	poll, err := n.Run(t.Context(), searching.Job{ID: t.Name() + "/" + strconv.Itoa(int(jobs.Add(1))), Indexer: "i", Kind: searching.Torznab, Address: address, APIKey: key, Call: call})
	if err != nil || poll.Answer == nil {
		t.Fatalf("Run = %+v, %v; want an answer within the hold", poll, err)
	}
	return *poll.Answer
}

func outcome(a searching.Answer) string {
	if a.Outcome == nil {
		return "ok"
	}
	return fmt.Sprintf("%T %v", a.Outcome, a.Outcome)
}

func TestTheNodeKeepsToTheRequestTimeAndSize(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/torznab/api", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("t") {
		case "caps":
			_, _ = w.Write([]byte(strings.Repeat("x", 8<<20+1)))
		case "search":
			<-r.Context().Done()
		}
	})
	indexer := httptest.NewServer(mux)
	t.Cleanup(indexer.Close)
	n := node(t, &bytes.Buffer{})
	address := indexer.URL + "/torznab"
	if got, want := outcome(run(t, n, address, "k", searching.TestCall{})), "*searching.Failing the Indexer's response is over 8388608 bytes"; got != want {
		t.Errorf("an oversized caps = %s, want %s", got, want)
	}
	if got, want := outcome(run(t, n, address, "k", searching.SearchCall{Query: searching.Query{Mode: "search"}})), "*searching.Failing reach the Indexer: context deadline exceeded"; got != want {
		t.Errorf("a slow search = %s, want %s", got, want)
	}
}

func TestALinkMayLeaveTheIndexersHostWithoutItsKey(t *testing.T) {
	var asked []string
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.String())
		_, _ = w.Write([]byte("d4:infoe"))
	}))
	t.Cleanup(elsewhere.Close)
	mux := http.NewServeMux()
	mux.HandleFunc("/away.torrent", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/a.torrent", http.StatusFound)
	})
	indexer := httptest.NewServer(mux)
	t.Cleanup(indexer.Close)
	n := node(t, &bytes.Buffer{})
	for _, link := range []string{indexer.URL + "/away.torrent", elsewhere.URL + "/b.torrent"} {
		a := run(t, n, indexer.URL, "s3cret", searching.FetchCall{Link: searching.LinkOf(link)})
		if got, ok := a.Got.(searching.Torrent); a.Outcome != nil || !ok || string(got.Raw) != "d4:infoe" {
			t.Errorf("Fetch of %s = %+v, %s; want the .torrent from the other host", link, a.Got, outcome(a))
		}
	}
	if want := "[/a.torrent /b.torrent]"; fmt.Sprint(asked) != want {
		t.Fatalf("the other host was asked for %v, want %s and never the key", asked, want)
	}
}

func TestARedirectToAMagnetLinkIsReturned(t *testing.T) {
	const link = "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=Night+Harbor"
	mux := http.NewServeMux()
	mux.HandleFunc("/dl/1", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, link, http.StatusFound) })
	mux.HandleFunc("/dl/2", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dl/1", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/dl/loop", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/dl/loop", http.StatusFound) })
	indexer := httptest.NewServer(mux)
	t.Cleanup(indexer.Close)
	n := node(t, &bytes.Buffer{})
	for _, path := range []string{"/dl/1", "/dl/2"} {
		a := run(t, n, indexer.URL, "k", searching.FetchCall{Link: searching.LinkOf(indexer.URL + path)})
		got, ok := a.Got.(searching.Magnet)
		if m, err := got.Link.Magnet(); !ok || err != nil || m.InfoHash.String() != "0123456789abcdef0123456789abcdef01234567" {
			t.Errorf("a redirect to a magnet link from %s = %+v, %s; want the magnet link", path, a.Got, outcome(a))
		}
	}
	a := run(t, n, indexer.URL, "k", searching.FetchCall{Link: searching.LinkOf(indexer.URL + "/dl/loop")})
	if got, want := outcome(a), "*searching.Failing reach the Indexer: the Indexer redirected 10 times"; got != want {
		t.Errorf("a redirect loop = %s, want %s", got, want)
	}
}

func TestEachResponseClassLandsInItsOutcome(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/limited/api", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	mux.HandleFunc("/down/api", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) })
	mux.HandleFunc("/gone/api", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	mux.HandleFunc("/refusing/api", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`<error code="100" description="Invalid API Key" />`))
	})
	mux.HandleFunc("/garbled/api", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<rss><channel><item>")) })
	mux.HandleFunc("/empty/api", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("t") == "caps" {
			_, _ = w.Write([]byte(caps))
			return
		}
		_, _ = w.Write([]byte("<rss><channel></channel></rss>"))
	})
	indexer := httptest.NewServer(mux)
	t.Cleanup(indexer.Close)
	n := node(t, &bytes.Buffer{})
	search := searching.SearchCall{Query: searching.Query{Mode: "search", Text: "orchard"}}
	for _, tc := range []struct{ path, want string }{
		{"/limited", "*searching.Limited the Indexer limited requests for 30s"},
		{"/down", "*searching.Failing the Indexer responded with status 502"},
		{"/gone", "*searching.Failing the Indexer responded with status 404"},
		{"/refusing", "*searching.Stopped the Indexer refused the account: Invalid API Key"},
		{"/garbled", "*searching.Failing read the results: XML syntax error on line 1: unexpected EOF"},
		{"/empty", "ok"},
	} {
		if got := outcome(run(t, n, indexer.URL+tc.path, "k", search)); got != tc.want {
			t.Errorf("a search of %s = %s, want %s", tc.path, got, tc.want)
		}
	}
	a := run(t, n, indexer.URL+"/empty", "k", searching.TestCall{})
	if a.Outcome != nil || a.Report == nil || a.Report.Items != 0 || fmt.Sprint(a.Report.Capabilities.Modes) != "map[search:[q]]" {
		t.Fatalf("a Test of an empty feed = %+v, %s; want its caps and zero items passing", a.Report, outcome(a))
	}
}

func TestTheLogAndErrorsCarryNoKeyOrLink(t *testing.T) {
	const key, passkey = "s3cretkey", "p4sskey"
	indexer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" {
			_, _ = w.Write([]byte("<rss><channel></channel></rss>"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(indexer.Close)
	var log bytes.Buffer
	n := node(t, &log)
	run(t, n, indexer.URL, key, searching.SearchCall{Query: searching.Query{Mode: "search"}})
	failed := run(t, n, indexer.URL, key, searching.FetchCall{Link: searching.LinkOf(indexer.URL + "/dl?passkey=" + passkey)})
	unreachable := run(t, n, indexer.URL, key, searching.FetchCall{Link: searching.LinkOf("http://127.0.0.1:1/dl?passkey=" + passkey)})
	link := searching.LinkOf("https://tracker.test/dl?passkey=" + passkey)
	var logged bytes.Buffer
	slog.New(slog.NewTextHandler(&logged, nil)).Info("fetch", "link", link)
	for name, text := range map[string]string{
		"the request log": log.String(), "a failure": outcome(failed), "an unreachable host": outcome(unreachable),
		"a printed link": fmt.Sprintf("%v %+v %s", link, link, link), "a logged link": logged.String(),
	} {
		if strings.Contains(text, key) || strings.Contains(text, passkey) {
			t.Errorf("%s carries a secret: %s", name, text)
		}
	}
	if !strings.Contains(log.String(), "An Indexer responded") {
		t.Fatalf("the request log is %q, want each response", log.String())
	}
}
