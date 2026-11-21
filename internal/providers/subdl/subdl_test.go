package subdl_test

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
	"github.com/jatassi/slipstream/internal/providers/subdl"
	"github.com/jatassi/slipstream/internal/subtitling"
)

func client(t *testing.T, h http.HandlerFunc) *subdl.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	lines := calls.NewLines(calls.NoRedirects(srv.Client()), "SlipStream/test", calls.Timings{Attempts: 1, Try: 5 * time.Second, RetryMin: time.Millisecond, RetryMax: time.Millisecond, Wait: time.Second}, rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
	timings := subdl.DefaultTimings()
	timings.Interval, timings.Busy.Every = time.Millisecond, time.Millisecond
	return subdl.New(lines, subdl.Endpoints{API: endpoint.MustParseURL(srv.URL), KeyPage: endpoint.MustParseURL("https://subdl.invalid/panel/api")}, timings)
}

func creds() calls.Credentials {
	return calls.Credentials{Keys: map[calls.ID]string{subdl.ID: "secret-key"}, Tally: &calls.Tally{}}
}

func film(t *testing.T) subtitling.Query {
	t.Helper()
	en, err := language.Parse("en")
	if err != nil {
		t.Fatal(err)
	}
	return subtitling.Query{Subject: subtitling.Subject{Kind: subtitling.Feature, IMDb: "tt0113277"}, Language: en}
}

func TestSignals(t *testing.T) {
	reset := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	for _, row := range []struct {
		name   string
		status int
		header map[string]string
		body   string
		kind   calls.OutcomeKind
		words  string
		until  time.Time
	}{
		{"a rejected key quotes the message", 401, nil, `{"error":{"code":"unauthorized","message":"Not Authorized","docs_url":"x"}}`, calls.Refused, "refused the key: Not Authorized", time.Time{}},
		{"a spent quota pauses until the reset", 429, map[string]string{"X-Ratelimit-Reset": "2026-11-19T00:00:00.000Z"}, `{"error":{"code":"quota_exceeded","message":"Quota"}}`, calls.Capped, "has used up its allowance until it resets", reset},
		{"a challenge page is not a refused key", 403, map[string]string{"Content-Type": "text/html"}, `<!DOCTYPE html>`, calls.Unreachable, "responded with a challenge page", time.Time{}},
		{"402 is down", 402, nil, ``, calls.Unreachable, "responded with Payment Required", time.Time{}},
	} {
		t.Run(row.name, func(t *testing.T) {
			c := client(t, func(w http.ResponseWriter, _ *http.Request) {
				for k, v := range row.header {
					w.Header().Set(k, v)
				}
				w.WriteHeader(row.status)
				_, _ = w.Write([]byte(row.body))
			})
			_, err := c.Search(t.Context(), creds(), film(t))
			f, ok := errors.AsType[*calls.Failed](err)
			if !ok || f.Outcome.Kind != row.kind || f.Outcome.Words != row.words || !f.Outcome.Until.Equal(row.until) {
				t.Errorf("Search = %v (%+v), want kind %d %q until %v", err, f, row.kind, row.words, row.until)
			}
		})
	}
}

func TestBusyTriesAgain(t *testing.T) {
	var answers atomic.Int32
	c := client(t, func(w http.ResponseWriter, _ *http.Request) {
		if answers.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"service_busy","message":"busy"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"subtitles":[]}`))
	})
	if _, err := c.Search(t.Context(), creds(), film(t)); err != nil || answers.Load() != 2 {
		t.Errorf("Search = %v after %d calls, want an answer on the second", err, answers.Load())
	}
}

func TestKeyOnlyInTheHeader(t *testing.T) {
	var got *http.Request
	c := client(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte(`{"subtitles":[{"subtitlePage":"/s/info/abc","language":"EN","url":"https://dl.subdl.com/subtitle/1.zip?api_key=secret-key"}]}`))
	})
	offers, err := c.Search(t.Context(), creds(), film(t))
	if err != nil {
		t.Fatal(err)
	}
	if got.Header.Get("Authorization") != "Bearer secret-key" || strings.Contains(got.URL.String(), "secret-key") {
		t.Errorf("sent %s with Authorization %q, want the key in the header alone", got.URL, got.Header.Get("Authorization"))
	}
	if q := got.URL.Query(); q.Get("client") != "custom_integration" || q.Get("releases") != "1" || q.Get("comment") != "1" || q.Has("bazarr") {
		t.Errorf("search query = %s", got.URL.RawQuery)
	}
	if len(offers) != 1 || offers[0].Key != "abc" || offers[0].Page != "https://subdl.invalid/s/info/abc" {
		t.Errorf("offers = %+v, want abc from subtitlePage and its page on SubDL's site", offers)
	}
}

func TestRedirectRefused(t *testing.T) {
	var followed atomic.Bool
	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { followed.Store(true) }))
	t.Cleanup(other.Close)
	c := client(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/subtitle/1.zip", http.StatusFound)
	})
	_, err := c.Download(t.Context(), creds(), subtitling.Offer{Provider: subdl.ID, Key: "abc"})
	if f, ok := errors.AsType[*calls.Failed](err); !ok || f.Outcome.Kind != calls.Unreachable || followed.Load() {
		t.Errorf("Download = %v with the redirect followed %t, want it unreachable and not followed", err, followed.Load())
	}
}
