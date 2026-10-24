package omdb_test

import (
	"errors"
	"math/rand/v2"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	fakeomdb "github.com/jatassi/slipstream/fakes/omdb"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/omdb"
	"github.com/jatassi/slipstream/internal/works"
)

// counted is f's client, counting each request it sends.
type counted struct {
	next http.RoundTripper
	sent atomic.Int64
}

func (c *counted) RoundTrip(r *http.Request) (*http.Response, error) {
	c.sent.Add(1)
	return c.next.RoundTrip(r) //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
}

func client(f faketest.Served) (*omdb.Client, *counted) {
	sent := &counted{next: f.Client.Transport}
	c := *f.Client
	c.Transport = sent
	lines := metadata.NewLines(&c, "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
	return omdb.New(lines, omdb.Endpoints{API: endpoint.MustParseURL(f.Address.URL(""))}, omdb.DefaultTimings()), sent
}

func inEachTier(t *testing.T, s scenario.Scenario, test func(t *testing.T, c *omdb.Client, f faketest.Served)) {
	t.Run("loopback", func(t *testing.T) {
		f := faketest.Loopback(t, fakeomdb.Fake, s)
		c, _ := client(f)
		test(t, c, f)
	})
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			f := faketest.InMemory(t, fakeomdb.Fake, s)
			c, _ := client(f)
			test(t, c, f)
		})
	})
}

func ptr[T any](v T) *T { return &v }

func withTitles(t *testing.T, titles ...scenario.OMDbTitle) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.OMDb = &scenario.OMDb{Titles: titles}
	return s
}

var (
	keyed   = metadata.NewAsk(metadata.AskOptions{Ratings: true, Keys: map[metadata.ProviderID]string{omdb.ID: "yours"}})
	orchard = scenario.OMDbTitle{IMDbID: "tt9000001", IMDbRating: ptr(7.4), IMDbVotes: ptr(int64(1234567)), RottenTomatoes: ptr(91), Metacritic: ptr(78), Awards: "Won 1 Oscar. 12 wins & 30 nominations total"}
)

func TestTitleReadsEachRatingOnItsScaleAndTheAwardsLine(t *testing.T) {
	unrated := scenario.OMDbTitle{IMDbID: "tt9000002", IMDbRating: ptr(6.0)}
	inEachTier(t, withTitles(t, orchard, unrated), func(t *testing.T, c *omdb.Client, _ faketest.Served) {
		got, err := c.Title(t.Context(), keyed, "tt9000001")
		if err != nil {
			t.Fatal(err)
		}
		want := omdb.Title{
			Ratings: []works.Rating{{Source: "imdb", Value: 7.4, Votes: ptr(int64(1234567))}, {Source: "rottentomatoes", Value: 91}, {Source: "metacritic", Value: 78}},
			Awards:  "Won 1 Oscar. 12 wins & 30 nominations total",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Title = %+v, want %+v", got, want)
		}

		got, err = c.Title(t.Context(), keyed, "tt9000002")
		if want := (omdb.Title{Ratings: []works.Rating{{Source: "imdb", Value: 6}}}); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("Title of a title with one rating, no votes and no awards = %+v, %v; want %+v", got, err, want)
		}
	})
}

func TestAnUnknownIDIsNotFoundAndTestStillAnswers(t *testing.T) {
	inEachTier(t, withTitles(t, orchard), func(t *testing.T, c *omdb.Client, _ faketest.Served) {
		for _, id := range []string{"tt9000002", "orchard"} {
			_, err := c.Title(t.Context(), keyed, id)
			if want := "GET " + id + ": OMDb: has no such record"; err == nil || err.Error() != want || !errors.Is(err, works.ErrNotFound) {
				t.Errorf("Title(%s) = %v, want %s", id, err, want)
			}
		}
		if err := c.Test(t.Context(), keyed); err != nil {
			t.Errorf("Test with a key the fake takes = %v, want nil", err)
		}
	})
}

func TestARevokedKeyIsRefusedInOMDbsWords(t *testing.T) {
	inEachTier(t, withTitles(t, orchard), func(t *testing.T, c *omdb.Client, f faketest.Served) {
		f.Cue(t, fakeomdb.RevokeKey.Name, fakeomdb.Key{Key: ptr("yours")})
		for name, call := range map[string]func() error{
			"Title": func() error { _, err := c.Title(t.Context(), keyed, "tt9000001"); return err },
			"Test":  func() error { return c.Test(t.Context(), keyed) },
		} {
			err := call()
			if want := "OMDb: refused the key: Invalid API key!"; !errors.Is(err, works.ErrRefused) || (err == nil || !strings.HasSuffix(err.Error(), want)) {
				t.Errorf("%s = %v, want works.ErrRefused ending %q", name, err, want)
			}
		}
	})
}

func TestOMDbCapLimitsUntilUTCMidnightWithoutSending(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := faketest.InMemory(t, fakeomdb.Fake, withTitles(t, orchard))
		c, sent := client(f)
		if _, err := c.Title(t.Context(), keyed, "tt9000001"); err != nil {
			t.Fatal(err)
		}
		f.Cue(t, fakeomdb.Cap.Name, fakeomdb.Key{})

		midnight := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
		_, err := c.Title(t.Context(), keyed, "tt9000001")
		se, ok := errors.AsType[*works.SourceError](err)
		if !ok || !errors.Is(err, works.ErrUnavailable) || !se.Until.Equal(midnight) || se.Words != "is out of its daily allowance until "+midnight.Format(time.RFC3339) {
			t.Fatalf("Title after the cap = %v (%+v), want works.ErrUnavailable until %s", err, se, midnight)
		}

		before := sent.sent.Load()
		time.Sleep(time.Until(midnight) - time.Second)
		if err := c.Test(t.Context(), keyed); !errors.Is(err, works.ErrUnavailable) || sent.sent.Load() != before {
			t.Errorf("Test a second before midnight = %v after %d more requests, want works.ErrUnavailable sending none", err, sent.sent.Load()-before)
		}

		time.Sleep(time.Second)
		if _, err := c.Title(t.Context(), keyed, "tt9000001"); err != nil || sent.sent.Load() != before+1 {
			t.Errorf("Title at midnight = %v after %d more requests, want the ratings in one", err, sent.sent.Load()-before)
		}
	})
}
