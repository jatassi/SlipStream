package subsource_test

import (
	"errors"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/subsource"
	"github.com/jatassi/slipstream/internal/subtitling"
)

func client(t *testing.T, h http.HandlerFunc) *subsource.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	lines := calls.NewLines(calls.NoRedirects(srv.Client()), "SlipStream/test", calls.Timings{Attempts: 1, Try: 5 * time.Second, RetryMin: time.Millisecond, RetryMax: time.Millisecond, Wait: time.Millisecond}, rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
	return subsource.New(lines, subsource.Endpoints{API: endpoint.MustParseURL(srv.URL), KeyPage: endpoint.MustParseURL("https://subsource.invalid/profile")}, subsource.Timings{Interval: time.Millisecond, Burst: 1})
}

func creds() calls.Credentials {
	return calls.Credentials{Keys: map[calls.ID]string{subsource.ID: "secret-key"}, Tally: &calls.Tally{}}
}

func TestSignals(t *testing.T) {
	now := time.Now().UTC()
	midnight := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	for _, row := range []struct {
		name   string
		status int
		header map[string]string
		body   string
		kind   calls.OutcomeKind
		words  string
		until  time.Time
	}{
		{"a rejected key quotes the message", 401, nil, `{"error":"Invalid API key","message":"The provided API key is invalid or inactive"}`, calls.Refused, "refused the key: The provided API key is invalid or inactive", time.Time{}},
		{"a spent day pauses to UTC midnight", 429, map[string]string{"X-Ratelimit-Remaining-Day": "0", "X-Ratelimit-Remaining-Hour": "0"}, `{"retryAfter":30}`, calls.Capped, "has used up today's requests", midnight},
	} {
		t.Run(row.name, func(t *testing.T) {
			c := client(t, func(w http.ResponseWriter, _ *http.Request) {
				for k, v := range row.header {
					w.Header().Set(k, v)
				}
				w.WriteHeader(row.status)
				_, _ = w.Write([]byte(row.body))
			})
			_, err := c.CheckKey(t.Context(), creds())
			f, ok := errors.AsType[*calls.Failed](err)
			if !ok || f.Outcome.Kind != row.kind || f.Outcome.Words != row.words || !f.Outcome.Until.Equal(row.until) {
				t.Errorf("CheckKey = %v (%+v), want kind %d %q until %v", err, f, row.kind, row.words, row.until)
			}
		})
	}
}

func TestASpentHourWaitsRetryAfter(t *testing.T) {
	c := client(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ratelimit-Remaining-Hour", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"retryAfter":600}`))
	})
	before := time.Now()
	_, err := c.CheckKey(t.Context(), creds())
	f, ok := errors.AsType[*calls.Failed](err)
	if !ok || f.Outcome.Kind != calls.Limited || f.Outcome.Until.Before(before.Add(600*time.Second)) || f.Outcome.Until.After(time.Now().Add(600*time.Second)) {
		t.Fatalf("CheckKey = %v (%+v), want limited 600 s", err, f)
	}
}

func TestAnEpisodeKeepsItsOwnSubtitlesAndPacks(t *testing.T) {
	c := client(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/movies/search") {
			_, _ = w.Write([]byte(`{"success":true,"data":[{"movieId":7,"type":"tvseries","season":2},{"movieId":6,"type":"tvseries","season":1}]}`))
			return
		}
		if r.URL.Query().Get("movieId") != "6" || r.URL.Query().Get("language") != "brazilian_portuguese" {
			t.Errorf("listed %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":[
			{"subtitleId":1,"language":"brazilian_portuguese","releaseInfo":["Show.S01E02.1080p"]},
			{"subtitleId":2,"language":"brazilian_portuguese","releaseInfo":["Show.S01E03.1080p"]},
			{"subtitleId":3,"language":"brazilian_portuguese","releaseInfo":["Show - Season 1 (WEBRip)"]},
			{"subtitleId":4,"language":"english","releaseInfo":["Show.S01E02.1080p"]}]}`))
	})
	pt, _ := language.Parse("pt-BR")
	offers, err := c.Search(t.Context(), creds(), subtitling.Query{Subject: subtitling.Subject{Kind: subtitling.Episode, IMDb: "tt1", Season: 1, Episodes: []int{2}}, Language: pt})
	if err != nil {
		t.Fatal(err)
	}
	if len(offers) != 2 || offers[0].Key != "1" || offers[0].Pack || offers[1].Key != "3" || !offers[1].Pack || offers[1].Season != 1 {
		t.Fatalf("offers = %+v, want episode 2's subtitle and the season pack", offers)
	}
}
