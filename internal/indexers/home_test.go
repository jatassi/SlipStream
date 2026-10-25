package indexers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/indexers"
	searchv1 "github.com/jatassi/slipstream/internal/pb/slipstream/search/v1"
)

func TestTheHomeNodeKeepsToTheIndexersHostTimeAndSize(t *testing.T) {
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("d4:infoe")) }))
	t.Cleanup(elsewhere.Close)
	mux := http.NewServeMux()
	mux.HandleFunc("/torznab/api", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("t") {
		case "caps":
			_, _ = w.Write([]byte(strings.Repeat("x", 8<<20+1)))
		case "search":
			<-r.Context().Done()
		}
	})
	mux.HandleFunc("/away.torrent", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/a.torrent", http.StatusFound)
	})
	indexer := httptest.NewServer(mux)
	t.Cleanup(indexer.Close)
	home := indexers.NewHome(&http.Client{}, indexers.Timings{Request: time.Second})
	idx := &searchv1.Indexer{Url: indexer.URL + "/torznab", ApiKey: "k"}

	patient := indexers.NewHome(&http.Client{}, indexers.Timings{Request: time.Minute})
	_, err := patient.Caps(t.Context(), &searchv1.CapsRequest{Indexer: idx})
	if want := "resource_exhausted: the Indexer's response is over 8388608 bytes"; err == nil || err.Error() != want {
		t.Errorf("an oversized caps = %v, want %s", err, want)
	}
	_, err = home.Search(t.Context(), &searchv1.SearchRequest{Indexer: idx, Params: map[string]string{"t": "search"}})
	if want := "aborted: reach the Indexer: context deadline exceeded"; err == nil || err.Error() != want {
		t.Errorf("a slow search = %v, want %s", err, want)
	}
	_, err = home.FetchTorrent(t.Context(), &searchv1.FetchTorrentRequest{Indexer: idx, Url: indexer.URL + "/away.torrent"})
	if want := "aborted: reach the Indexer: the Indexer redirected off its own host"; err == nil || err.Error() != want {
		t.Errorf("a redirect elsewhere = %v, want %s", err, want)
	}
	_, err = home.FetchTorrent(t.Context(), &searchv1.FetchTorrentRequest{Indexer: idx, Url: elsewhere.URL + "/a.torrent"})
	if want := "invalid_argument: the torrent's URL is not on the Indexer's host"; err == nil || err.Error() != want {
		t.Errorf("a torrent elsewhere = %v, want %s", err, want)
	}
}
