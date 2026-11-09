package tvdb_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/tvdb"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tvdb"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	provider "github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/works"
)

func inBothTiers(t *testing.T, s scenario.Scenario, test func(t *testing.T, f faketest.Served)) {
	t.Run("loopback", func(t *testing.T) { test(t, faketest.Loopback(t, tvdb.Fake, s)) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { test(t, faketest.InMemory(t, tvdb.Fake, s)) })
	})
}

func load(t *testing.T) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("series")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func send(t *testing.T, f faketest.Served, method, path, token string, body []byte) (status int, got string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, f.Address.URL(path), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	read, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(read)
}

func login(t *testing.T, f faketest.Served, body string) (status int, tokenOrBody string) {
	t.Helper()
	status, got := send(t, f, http.MethodPost, "/v4/login", "", []byte(body))
	if status != http.StatusOK {
		return status, got
	}
	var l struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(got), &l); err != nil || l.Data.Token == "" {
		t.Fatalf("login = %s: %v", got, err)
	}
	return status, l.Data.Token
}

const (
	refusedKey = `{"status":"failure","message":"Unauthorized","data":null}`
	refusedPin = `{"status":"failure","message":"Unauthorized: the subscriber PIN does not match this key","data":null}`
)

func TestEveryCallButLoginTakesALiveToken(t *testing.T) {
	inBothTiers(t, load(t), func(t *testing.T, f faketest.Served) {
		for _, path := range []string{"/v4/languages", "/v4/search?query=lantern&type=series", "/v4/series/1001/extended", "/v4/series/1001/episodes/official/eng?page=0", "/v4/updates?since=0&type=series"} {
			if status, body := send(t, f, http.MethodGet, path, "", nil); status != http.StatusUnauthorized || body != refusedKey {
				t.Errorf("GET %s without a token = %d %s, want 401 %s", path, status, body, refusedKey)
			}
			if status, _ := send(t, f, http.MethodGet, path, "not-a-token", nil); status != http.StatusUnauthorized {
				t.Errorf("GET %s with a made-up token = %d, want 401", path, status)
			}
		}
		if status, body := login(t, f, `{"apikey":""}`); status != http.StatusUnauthorized || body != refusedKey {
			t.Errorf("login with no key = %d %s, want 401 %s", status, body, refusedKey)
		}
		_, token := login(t, f, `{"apikey":"any key"}`)
		if status, body := send(t, f, http.MethodGet, "/v4/languages", token, nil); status != http.StatusOK || !strings.Contains(body, `{"id":"eng","name":"English","nativeName":"English","shortCode":"en"}`) {
			t.Errorf("GET /v4/languages = %d %s, want 200 listing English", status, body)
		}
	})
}

func TestExpireTokensEndsEveryTokenAndALoginWorksAgain(t *testing.T) {
	inBothTiers(t, load(t), func(t *testing.T, f faketest.Served) {
		_, token := login(t, f, `{"apikey":"key"}`)
		f.Cue(t, tvdb.ExpireTokens.Name, struct{}{})
		if status, _ := send(t, f, http.MethodGet, "/v4/languages", token, nil); status != http.StatusUnauthorized {
			t.Errorf("GET /v4/languages on an expired token = %d, want 401", status)
		}
		_, again := login(t, f, `{"apikey":"key"}`)
		if status, _ := send(t, f, http.MethodGet, "/v4/languages", again, nil); status != http.StatusOK || again == token {
			t.Errorf("GET /v4/languages on a new token = %d (new token %v), want 200", status, again != token)
		}
	})
}

func TestRejectKeyAndRejectPinAnswerTheirOwnBodies(t *testing.T) {
	inBothTiers(t, load(t), func(t *testing.T, f faketest.Served) {
		mine, theirs := "mine", "theirs"
		_, pinned := login(t, f, `{"apikey":"theirs","pin":"1234"}`)
		_, bare := login(t, f, `{"apikey":"theirs"}`)
		f.Cue(t, tvdb.RejectPin.Name, tvdb.Key{Key: &theirs})
		if status, body := login(t, f, `{"apikey":"theirs","pin":"1234"}`); status != http.StatusUnauthorized || body != refusedPin {
			t.Errorf("login with a rejected PIN = %d %s, want 401 %s", status, body, refusedPin)
		}
		if status, _ := send(t, f, http.MethodGet, "/v4/languages", pinned, nil); status != http.StatusUnauthorized {
			t.Errorf("the PIN's old token = %d, want 401", status)
		}
		if status, _ := send(t, f, http.MethodGet, "/v4/languages", bare, nil); status != http.StatusOK {
			t.Errorf("the key's token without a PIN = %d, want 200", status)
		}
		f.Cue(t, tvdb.RejectKey.Name, tvdb.Key{Key: &mine})
		if status, body := login(t, f, `{"apikey":"mine"}`); status != http.StatusUnauthorized || body != refusedKey {
			t.Errorf("login with a rejected key = %d %s, want 401 %s", status, body, refusedKey)
		}
		if status, _ := login(t, f, `{"apikey":"theirs"}`); status != http.StatusOK {
			t.Errorf("login with another key = %d, want 200", status)
		}
	})
}

func client(f faketest.Served) *provider.Client {
	return provider.New(lines(f.Client), provider.Endpoints{API: endpoint.MustParseURL(f.Address.URL("/v4"))}, provider.DefaultTimings())
}

func lines(c *http.Client) *metadata.Lines {
	return metadata.NewLines(c, "SlipStream/test ( github.com/jatassi/SlipStream )", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // jitter needs no secrecy
}

func ask(pin string) metadata.Ask {
	o := metadata.AskOptions{Language: "en-US", Keys: map[metadata.ProviderID]string{provider.ID: "key"}}
	if pin != "" {
		o.Pins = map[metadata.ProviderID]string{provider.ID: pin}
	}
	return metadata.NewAsk(o)
}

func TestTokenExpiryCueRelogsIn(t *testing.T) {
	inBothTiers(t, load(t), func(t *testing.T, f faketest.Served) {
		c := client(f)
		if err := c.Test(t.Context(), ask("")); err != nil {
			t.Fatal(err)
		}
		f.Cue(t, tvdb.ExpireTokens.Name, struct{}{})
		if _, err := c.Series(t.Context(), ask(""), 1001); err != nil {
			t.Errorf("Series after the tokens expired = %v, want the client to log in again", err)
		}
	})
}

func TestRejectedPinCue(t *testing.T) {
	inBothTiers(t, load(t), func(t *testing.T, f faketest.Served) {
		c := client(f)
		if err := c.Test(t.Context(), ask("1234")); err != nil {
			t.Fatal(err)
		}
		f.Cue(t, tvdb.RejectPin.Name, tvdb.Key{})
		err := c.Test(t.Context(), ask("1234"))
		if !errors.Is(err, works.ErrRefused) || !strings.HasSuffix(err.Error(), "TVDB: refused the PIN") {
			t.Errorf("Test with a rejected PIN = %v, want works.ErrRefused for the PIN", err)
		}
		if err := c.Test(t.Context(), ask("")); err != nil {
			t.Errorf("Test with the key alone = %v, want it taken", err)
		}
		f.Cue(t, tvdb.RejectKey.Name, tvdb.Key{})
		err = c.Test(t.Context(), ask(""))
		if !errors.Is(err, works.ErrRefused) || !strings.Contains(err.Error(), "TVDB: refused the key") {
			t.Errorf("Test with a rejected key = %v, want works.ErrRefused for the key", err)
		}
	})
}

func aired(t *testing.T, c *provider.Client, id provider.SeriesID) []provider.Episode {
	t.Helper()
	episodes, err := c.Episodes(t.Context(), ask(""), id, provider.OrderAired)
	if err != nil {
		t.Fatal(err)
	}
	return episodes
}

func codes(episodes []provider.Episode) []string {
	var out []string
	for _, e := range episodes {
		out = append(out, fmt.Sprintf("%dx%d", e.Season, e.Number))
	}
	return out
}

func TestMoveAndRemoveEpisodeCuesReshapeTheOrders(t *testing.T) {
	inBothTiers(t, load(t), func(t *testing.T, f faketest.Served) {
		c := client(f)
		if got, want := codes(aired(t, c, 1001)), []string{"0x1", "0x2", "1x1", "1x2", "1x3", "2x1", "2x2", "2x3", "3x1", "3x2", "3x3"}; !slices.Equal(got, want) {
			t.Fatalf("Aired episodes = %v, want %v", got, want)
		}
		f.Cue(t, tvdb.MoveEpisode.Name, tvdb.Move{ID: 100102, Season: 3, Number: 4})
		f.Cue(t, tvdb.RemoveEpisode.Name, tvdb.ID{ID: 100112})
		episodes := aired(t, c, 1001)
		if got, want := codes(episodes), []string{"0x1", "1x1", "1x3", "2x1", "2x2", "2x3", "3x1", "3x2", "3x3", "3x4"}; !slices.Equal(got, want) {
			t.Errorf("Aired episodes after the cues = %v, want %v", got, want)
		}
		if moved := episodes[len(episodes)-1]; moved.ID != 100102 || moved.Name != "A Winter Light" {
			t.Errorf("S03E04 = %+v, want the special A Winter Light", moved)
		}
		dvd, err := c.Episodes(t.Context(), ask(""), 1001, provider.OrderDVD)
		if err != nil || slices.ContainsFunc(dvd, func(e provider.Episode) bool { return e.ID == 100112 }) {
			t.Errorf("DVD episodes = %+v, %v; want 100112 gone from every order", dvd, err)
		}
	})
}

func TestSeasonChangeAndDeleteCuesListTheirSeriesAsUpdated(t *testing.T) {
	inBothTiers(t, load(t), func(t *testing.T, f faketest.Served) {
		c := client(f)
		f.Cue(t, tvdb.AddSeason.Name, tvdb.ID{ID: 1002})
		returned, err := c.Series(t.Context(), ask(""), 1002)
		if err != nil || returned.Status != provider.Continuing || returned.NextAired == "" ||
			!slices.ContainsFunc(returned.Seasons, func(s provider.Season) bool { return s.Number == 3 && s.Order == provider.OrderAired }) {
			t.Errorf("Harbor Nights after a season = %+v, %v; want it continuing with season 3 next airing", returned, err)
		}

		f.Cue(t, tvdb.Change.Name, tvdb.ID{ID: 1004})
		f.Cue(t, tvdb.Delete.Name, tvdb.ID{ID: 1003})
		if _, err := c.Series(t.Context(), ask(""), 1003); !errors.Is(err, works.ErrNotFound) {
			t.Errorf("a deleted series = %v, want works.ErrNotFound", err)
		}
		changed, err := c.Series(t.Context(), ask(""), 1004)
		if err != nil || !strings.HasSuffix(changed.Overview, "Revision 1.") {
			t.Errorf("a changed series = %+v, %v; want its overview revised", changed, err)
		}
		updates, err := c.Updates(t.Context(), ask(""), time.Unix(0, 0), 1)
		if want := []provider.SeriesID{1002, 1004, 1003}; err != nil || !slices.Equal(updates.Series, want) || updates.Pages != 1 {
			t.Errorf("Updates = %+v, %v; want %v on one page", updates, err, want)
		}
	})
}

func TestSearchFindsAnAlternativeTitle(t *testing.T) {
	inBothTiers(t, load(t), func(t *testing.T, f faketest.Served) {
		found, err := client(f).Search(t.Context(), ask(""), "nights at the HARBOR", 0)
		if err != nil || len(found) != 1 || found[0].ID != 1002 || found[0].Name != "Harbor Nights" || found[0].Network != "Harbor TV" || found[0].Year != 2015 {
			t.Errorf("Search = %+v, %v; want Harbor Nights", found, err)
		}
	})
}

func TestTheFakeConforms(t *testing.T) {
	inBothTiers(t, load(t), func(t *testing.T, f faketest.Served) {
		refused := "refused"
		f.Cue(t, tvdb.RejectKey.Name, tvdb.Key{Key: &refused})
		conformance.Run(t, conformance.Target{
			Endpoints: provider.Endpoints{API: endpoint.MustParseURL(f.Address.URL("/v4"))},
			Client:    f.Client,
			Key:       "key",
			Refused:   refused,
			Known:     conformance.Known{ID: 1001, Name: "The Lantern Keepers"},
			Lines:     lines,
			Timings:   provider.DefaultTimings(),
		})
	})
}

// withRecorded adds a series recorded from TVDB beside the fictional ones.
func withRecorded(s scenario.Scenario) scenario.Scenario {
	tv := *s.TVDB
	tv.Series = append(slices.Clone(tv.Series), scenario.TVDBSeries{
		ID: 5001, Name: "오징어 게임", English: "Squid Game", Status: "ended", Network: "Netflix",
		Overview: "Hundreds of cash-strapped players play children's games.", Genres: []string{"Drama", "Thriller"},
		OriginalLanguage: "kor", OriginalCountry: "kor", Image: "/banners/v4/series/5001/posters/sg.jpg",
		Seasons: []scenario.TVDBSeason{
			{Number: 0, Episodes: []scenario.TVDBEpisode{{ID: 5100, Number: 1, Name: "Behind the Games"}}},
			{Number: 1, Name: "Season One", Image: "/banners/v4/season/5101/posters/s1.jpg", Episodes: []scenario.TVDBEpisode{
				{ID: 5101, Number: 1, Name: "Red Light, Green Light", Aired: "2021-09-17", Overview: "Gi-hun plays a game.", Image: "/banners/v4/episode/5101/screencap/e1.jpg"},
			}},
		},
	})
	s.TVDB = &tv
	return s
}

func TestARecordedSeriesAnswersItsRecord(t *testing.T) {
	inBothTiers(t, withRecorded(load(t)), func(t *testing.T, f faketest.Served) {
		c := client(f)
		got, err := c.Series(t.Context(), ask(""), 5001)
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != "Squid Game" || got.Original != "오징어 게임" || got.OriginalLanguage != "kor" || got.Country != "kor" ||
			got.Overview != "Hundreds of cash-strapped players play children's games." || !slices.Equal(got.Genres, []string{"Drama", "Thriller"}) ||
			got.Poster != "/banners/v4/series/5001/posters/sg.jpg" {
			t.Errorf("the recorded series = %+v, want its recorded name, language, country, overview, genres and poster", got)
		}
		seasons := map[int]provider.Season{}
		for _, s := range got.Seasons {
			seasons[s.Number] = s
		}
		if s := seasons[1]; s.Name != "Season One" || s.Poster != "/banners/v4/season/5101/posters/s1.jpg" {
			t.Errorf("season 1 = %+v, want its recorded name and poster", s)
		}
		if s := seasons[0]; s.Name != "Specials" || !strings.HasPrefix(s.Poster, "/banners/v4/seasons/") {
			t.Errorf("season 0 = %+v, want Specials with a made-up poster", s)
		}
		episodes := aired(t, c, 5001)
		if e := episodes[1]; e.Overview != "Gi-hun plays a game." || e.Still != "/banners/v4/episode/5101/screencap/e1.jpg" {
			t.Errorf("S01E01 = %+v, want its recorded overview and still", e)
		}
		if e := episodes[0]; e.Overview != "Behind the Games, an episode of Squid Game." || !strings.HasPrefix(e.Still, "/banners/v4/episodes/5100/") {
			t.Errorf("S00E01 = %+v, want a made-up overview and still", e)
		}
	})
}

func TestAFictionalSeriesBesideARecordedOneIsUnchanged(t *testing.T) {
	inBothTiers(t, withRecorded(load(t)), func(t *testing.T, f faketest.Served) {
		fictional, err := client(f).Series(t.Context(), ask(""), 1001)
		if err != nil {
			t.Fatal(err)
		}
		if fictional.Overview != "The Lantern Keepers is a fictional series." || fictional.OriginalLanguage != "eng" || fictional.Country != "usa" ||
			len(fictional.Genres) != 2 || !strings.HasPrefix(fictional.Poster, "/banners/v4/series/1001/posters/") {
			t.Errorf("the fictional series = %+v, want its made-up overview, language, country, genres and poster", fictional)
		}
	})
}

func TestARecordedSeriesNamesItsOriginalLanguageInSearch(t *testing.T) {
	inBothTiers(t, withRecorded(load(t)), func(t *testing.T, f faketest.Served) {
		_, token := login(t, f, `{"apikey":"key"}`)
		status, body := send(t, f, http.MethodGet, "/v4/search?query=squid&type=series", token, nil)
		var found struct {
			Data []struct {
				ImageURL        string `json:"image_url"`
				Thumbnail       string `json:"thumbnail"`
				PrimaryLanguage string `json:"primary_language"`
				Country         string `json:"country"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(body), &found); status != http.StatusOK || err != nil || len(found.Data) != 1 {
			t.Fatalf("GET /v4/search?query=squid = %d %s: %v", status, body, err)
		}
		hit := found.Data[0]
		if hit.ImageURL != "https://artworks.thetvdb.com/banners/v4/series/5001/posters/sg.jpg" || hit.Thumbnail != "https://artworks.thetvdb.com/banners/v4/series/5001/posters/sg_t.jpg" ||
			hit.PrimaryLanguage != "kor" || hit.Country != "kor" {
			t.Errorf("the search hit = %+v, want the recorded poster, its thumbnail, language and country", hit)
		}
	})
}
