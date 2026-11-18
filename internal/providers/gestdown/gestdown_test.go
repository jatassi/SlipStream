package gestdown_test

import (
	"errors"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/gestdown"
	"github.com/jatassi/slipstream/internal/subtitling"
)

func client(t *testing.T, h http.HandlerFunc) *gestdown.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	lines := calls.NewLines(calls.NoRedirects(srv.Client()), "SlipStream/test", calls.Timings{Attempts: 1, Try: 5 * time.Second, RetryMin: time.Millisecond, RetryMax: time.Millisecond, Wait: time.Second}, rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
	timings := gestdown.DefaultTimings()
	timings.Interval, timings.Refresh.Every = time.Millisecond, time.Millisecond
	return gestdown.New(lines, gestdown.Endpoints{API: endpoint.MustParseURL(srv.URL)}, timings)
}

func episode(t *testing.T) subtitling.Query {
	t.Helper()
	en, err := language.Parse("en")
	if err != nil {
		t.Fatal(err)
	}
	return subtitling.Query{Subject: subtitling.Subject{Kind: subtitling.Episode, TVDB: "81189", Season: 1, Episodes: []int{1}}, Language: en}
}

const show = `{"shows":[{"id":"1WLU6zJky4nlJQa5oojv5R"}]}`

func TestARefreshingShowIsTriedAgain(t *testing.T) {
	var lists atomic.Int32
	c := client(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/shows/") {
			_, _ = w.Write([]byte(show))
			return
		}
		if r.URL.Path != "/subtitles/get/1WLU6zJky4nlJQa5oojv5R/1/1/en" {
			t.Errorf("listed %s", r.URL.Path)
		}
		if lists.Add(1) < 3 {
			w.WriteHeader(http.StatusLocked)
			return
		}
		_, _ = w.Write([]byte(`{"matchingSubtitles":[
			{"subtitleId":"3Df15bUM1njyzL2GHv0PaX","version":"0TV, LOL","completed":true,"hearingImpaired":true,"downloadCount":5},
			{"subtitleId":"4Df15bUM1njyzL2GHv0PaX","version":"DIMENSION","completed":false}]}`))
	})
	offers, err := c.Search(t.Context(), calls.Credentials{Tally: &calls.Tally{}}, episode(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(offers) != 1 || offers[0].Key != "3Df15bUM1njyzL2GHv0PaX" || !offers[0].HI || len(offers[0].Releases) != 2 || offers[0].Episodes[0] != 1 || lists.Load() != 3 {
		t.Fatalf("offers = %+v after %d lists, want the completed subtitle on the third", offers, lists.Load())
	}
}

func TestAnUnknownShowAndAFilmFindNothing(t *testing.T) {
	c := client(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`"Couldn't find show: 81189"`))
	})
	film := episode(t)
	film.Subject = subtitling.Subject{Kind: subtitling.Feature, IMDb: "tt0113277"}
	for _, q := range []subtitling.Query{episode(t), film} {
		if offers, err := c.Search(t.Context(), calls.Credentials{Tally: &calls.Tally{}}, q); err != nil || len(offers) != 0 {
			t.Errorf("Search(%+v) = %+v, %v, want nothing", q.Subject, offers, err)
		}
	}
}

func TestUpstreamOutIsCapped(t *testing.T) {
	c := client(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"No more credentials available"}`))
	})
	before := time.Now()
	_, err := c.Download(t.Context(), calls.Credentials{Tally: &calls.Tally{}}, subtitling.Offer{Key: "3Df15bUM1njyzL2GHv0PaX"})
	f, ok := errors.AsType[*calls.Failed](err)
	if !ok || f.Outcome.Kind != calls.Capped || f.Outcome.Until.Before(before.Add(time.Hour)) {
		t.Fatalf("Download = %v (%+v), want capped for an hour", err, f)
	}
}
