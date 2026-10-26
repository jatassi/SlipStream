package indexers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/magnet"
	searchv1 "github.com/jatassi/slipstream/internal/pb/slipstream/search/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/search/v1/searchv1connect"
	"github.com/jatassi/slipstream/internal/role"
)

const link = "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=Night+Harbor&tr=https%3A%2F%2Ftracker.example.test%2Fannounce"

func TestTheHomeNodeReturnsARedirectToAMagnetLink(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dl/1", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, link, http.StatusFound) })
	mux.HandleFunc("/dl/2", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dl/1", http.StatusMovedPermanently)
	})
	indexer := httptest.NewServer(mux)
	t.Cleanup(indexer.Close)
	home := indexers.NewHome(&http.Client{}, indexers.Timings{Request: time.Second})
	idx := &searchv1.Indexer{Url: indexer.URL, ApiKey: "k"}
	for _, path := range []string{"/dl/1", "/dl/2"} {
		res, err := home.FetchTorrent(t.Context(), &searchv1.FetchTorrentRequest{Indexer: idx, Url: indexer.URL + path})
		if err != nil || res.GetMagnet() != link {
			t.Errorf("a redirect to a magnet link from %s = %v, %v; want the magnet link %s", path, res, err, link)
		}
	}
}

func TestFetchReadsAMagnetLinkWithoutANode(t *testing.T) {
	calls := 0
	x := indexers.New(indexers.Config{Searching: func(role.NodeID) searchv1connect.SearchingServiceHandler {
		calls++
		return nil
	}})
	got, err := x.Fetch(t.Context(), indexers.Indexer{}, link)
	want := indexers.Magnet{Link: magnet.Link{
		InfoHash: must(bencode.ParseInfoHash("0123456789abcdef0123456789abcdef01234567")),
		Name:     "Night Harbor", Trackers: []string{"https://tracker.example.test/announce"},
	}}
	if err != nil || !reflect.DeepEqual(got, want) || calls != 0 {
		t.Fatalf("Fetch of a magnet link = %+v, %v after %d Node calls; want %+v after none", got, err, calls, want)
	}
	if _, err := x.Fetch(t.Context(), indexers.Indexer{}, "magnet:?xt=urn:btmh:1220aa"); !errors.Is(err, bencode.ErrV2Only) {
		t.Errorf("Fetch of a btmh-only magnet link = %v, want ErrV2Only", err)
	}
	if _, err := x.Fetch(t.Context(), indexers.Indexer{}, "magnet:?dn=a"); !errors.As(err, new(*magnet.InvalidError)) {
		t.Errorf("Fetch of a magnet link with no info hash = %v, want *magnet.InvalidError", err)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
