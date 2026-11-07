package searching_test

import (
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/searching"
)

const brookFeed = `<rss version="2.0"><channel><title>Brook</title>
<item><title>Orchard.2021</title><guid>1</guid><enclosure url="http://brook.test/dl/1.torrent" length="6000" /></item>
<item><title>Meadow.2020</title><guid>2</guid><enclosure url="http://elsewhere.test/dl/2.torrent" length="10" /></item>
</channel></rss>`

// feedNode is a Node whose every host under brook.test and elsewhere.test
// is srv.
func feedNode(t *testing.T, srv *httptest.Server) searching.Remote {
	t.Helper()
	at := netip.MustParseAddrPort(srv.Listener.Addr().String())
	sites := searching.Endpoints{Sites: []searching.SiteAddress{{Domain: "brook.test", At: at}, {Domain: "elsewhere.test", At: at}}}
	timings := searching.DefaultTimings()
	timings.Floor = 0
	client := &http.Client{Transport: &http.Transport{DialContext: sites.Dial((&net.Dialer{}).DialContext)}}
	return searching.RemoteOf(searching.New(searching.Config{HTTP: client, Timings: timings, Log: slog.New(slog.DiscardHandler)}))
}

func feedJob(n searching.Remote, t *testing.T, session *searching.Session, call searching.Call) searching.Answer {
	t.Helper()
	poll, err := n.Run(t.Context(), searching.Job{
		ID: t.Name() + "/" + strconv.Itoa(int(jobs.Add(1))), Indexer: "brook", Kind: searching.RSS, Address: "http://brook.test/rss",
		Session: session, RSS: searching.RSSReading{Link: "enclosure"}, Call: call,
	})
	if err != nil || poll.Answer == nil {
		t.Fatalf("Run = %+v, %v; want an answer within the hold", poll, err)
	}
	return *poll.Answer
}

func TestAnRSSFeedGetsTheSessionOnlyOnItsOwnHost(t *testing.T) {
	sent := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent[r.Host+r.URL.Path] = r.Header.Get("Cookie") + " " + r.Header.Get("User-Agent")
		if r.URL.Path == "/rss" {
			_, _ = w.Write([]byte(brookFeed))
			return
		}
		_, _ = w.Write([]byte("d4:infoe"))
	}))
	t.Cleanup(srv.Close)
	n := feedNode(t, srv)
	session := &searching.Session{
		Cookies:   []searching.Cookie{{Name: "uid", Value: "7", Domain: "brook.test", Path: "/", HostOnly: true}},
		UserAgent: "Pasted/1.0", Pasted: true, Earned: time.Now(),
	}
	a := feedJob(n, t, session, searching.RecentCall{})
	if a.Outcome != nil || len(a.Releases) != 2 || a.Releases[0].Title != "Orchard.2021" || a.Releases[0].Size != 6000 {
		t.Fatalf("Recent = %+v, %v; want both items", a.Releases, a.Outcome)
	}
	for _, r := range a.Releases {
		if got := feedJob(n, t, session, searching.FetchCall{Link: r.Link}); got.Outcome != nil {
			t.Fatalf("Fetch of %s = %v", r.Title, got.Outcome)
		}
	}
	for path, want := range map[string]string{"brook.test/rss": "uid=7 Pasted/1.0", "brook.test/dl/1.torrent": "uid=7 Pasted/1.0", "elsewhere.test/dl/2.torrent": " Pasted/1.0"} {
		if got := sent[path]; got != want {
			t.Errorf("%s got cookie and User-Agent %q, want %q", path, got, want)
		}
	}
	if a := feedJob(n, t, nil, searching.SearchCall{Query: searching.Query{Mode: "search", Text: "orchard"}}); outcome(a) != "*searching.Failing an RSS feed has no search" {
		t.Errorf("a search of an RSS feed = %s, want a failure", outcome(a))
	}
}

func TestARefusedCookieStopsTheFeedAndOtherRefusalsFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	t.Cleanup(srv.Close)
	n := feedNode(t, srv)
	session := &searching.Session{Cookies: []searching.Cookie{{Name: "uid", Value: "old", Domain: "brook.test"}}, Pasted: true, Earned: time.Now()}
	if got, want := outcome(feedJob(n, t, session, searching.TestCall{})), "*searching.Stopped the site's cookie expired"; got != want {
		t.Errorf("a refused cookie = %s, want %s", got, want)
	}
	if got, want := outcome(feedJob(feedNode(t, srv), t, nil, searching.TestCall{})), "*searching.Failing the feed responded with status 403"; got != want {
		t.Errorf("a refusal with no cookie = %s, want %s", got, want)
	}
}
