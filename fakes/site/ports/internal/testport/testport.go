// Package testport is a port the fixture harness proves itself with: a
// site that signs in with a form to a cookie session and lists releases
// as JSON. Driver runs it by hand through the searching surface, so the
// harness is proven without internal/ports/port's driver.
package testport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/searching"
)

const ID port.ID = "testport"

type settings struct{ port.UserPass }

var errByHand = errors.New("testport runs through testport.Driver")

// Port is testport's entry. Its Impl is never run.
var Port = port.Define(port.Entry{
	ID: ID, Name: "Testport", Description: "Testport is a private tracker the fixture harness tests itself with.",
	Privacy: port.Private, IndexerURLs: []string{"https://testport.fixture-site.fake.localhost/"},
	Capabilities: port.Capabilities{
		SearchParams: []port.Param{port.Q}, MovieSearchParams: []port.Param{port.Q, port.ImdbID},
		Categories: port.Categories(port.Map(1, newznab.Movies, "Movies")),
	},
}, port.Impl[settings]{
	Login: port.FormLogin[settings]{DoLogin: func(*port.SignIn[settings]) error { return errByHand }},
	Basic: func(*port.Indexer[settings], port.SearchCriteria) (port.Chain, error) { return port.Chain{}, errByHand },
	Movie: func(*port.Indexer[settings], port.SearchCriteria) (port.Chain, error) { return port.Chain{}, errByHand },
	Parse: func(*port.Indexer[settings], *port.IndexerResponse) ([]port.TorrentInfo, error) {
		return nil, errByHand
	},
})

// Registry holds Port alone.
func Registry() port.Registry { return port.NewRegistry([]port.Port{Port}, nil, nil) }

// Driver is testport's protocol: GET /api/search with the session's
// cookie, a 401 bringing one sign-in at POST /login.
type Driver struct{}

func (Driver) Search(ctx context.Context, s *searching.Site, q searching.Query) ([]searching.Release, error) {
	return search(ctx, s, q.Text, q.IDs["imdbid"])
}

func (Driver) Recent(ctx context.Context, s *searching.Site) ([]searching.Release, error) {
	return search(ctx, s, "", "")
}

func (Driver) Fetch(ctx context.Context, s *searching.Site, link searching.Link) (searching.Got, error) {
	res, err := get(ctx, s, link.Reveal(), searching.Download)
	if err != nil {
		return nil, err
	}
	if res.Status != http.StatusOK {
		return nil, &searching.Failing{Err: fmt.Errorf("the download responded with status %d", res.Status)}
	}
	return searching.Torrent{Raw: res.Body}, nil
}

func (Driver) Test(ctx context.Context, s *searching.Site, _ *searching.CaptchaAnswer) (searching.Report, error) {
	rs, err := search(ctx, s, "", "")
	return searching.Report{Items: len(rs)}, err
}

func (Driver) Captcha(context.Context, *searching.Site) (searching.Captcha, error) {
	return searching.Captcha{}, &searching.Failing{Err: errors.New("testport has no captcha")}
}

func search(ctx context.Context, s *searching.Site, text, imdb string) ([]searching.Release, error) {
	q := url.Values{"q": {text}}
	if imdb != "" {
		q.Set("imdb", imdb)
	}
	at := s.Address.JoinPath("api", "search")
	at.RawQuery = q.Encode()
	seen := s.Held()
	res, err := get(ctx, s, at.String(), searching.Read)
	if err == nil && res.Status == http.StatusUnauthorized {
		err = s.SignIn(ctx, seen, false, func(ctx context.Context, in *searching.SigningIn) (*searching.Session, error) {
			return signIn(ctx, s, in)
		})
		if err == nil {
			res, err = get(ctx, s, at.String(), searching.Read)
		}
	}
	if err != nil {
		return nil, err
	}
	if res.Status != http.StatusOK {
		return nil, &searching.Failing{Err: fmt.Errorf("the search responded with status %d", res.Status)}
	}
	return parse(res.Body)
}

func signIn(ctx context.Context, s *searching.Site, in *searching.SigningIn) (*searching.Session, error) {
	form := url.Values{"username": {s.Settings["username"]}, "password": {s.Settings["password"]}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Address.JoinPath("login").String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, &searching.Failing{Err: err}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := in.Do(ctx, req, searching.Send{NoFollow: true})
	if err != nil {
		return nil, err
	}
	if res.Status == http.StatusForbidden {
		var refusal struct{ Error string }
		_ = json.Unmarshal(res.Body, &refusal)
		return nil, &searching.Stopped{Cause: searching.CredentialsRejected, Text: refusal.Error}
	}
	return &searching.Session{Cookies: in.Cookies(), Earned: time.Now()}, nil
}

func get(ctx context.Context, s *searching.Site, at string, p searching.Purpose) (*searching.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, at, http.NoBody)
	if err != nil {
		return nil, &searching.Failing{Err: err}
	}
	return s.Do(ctx, req, searching.Send{Purpose: p})
}

type listing struct {
	Results []struct {
		Name      string    `json:"name"`
		Size      int64     `json:"size"`
		Added     time.Time `json:"added"`
		Seeders   int       `json:"seeders"`
		Leechers  int       `json:"leechers"`
		Snatched  int       `json:"snatched"`
		InfoHash  string    `json:"infohash"`
		IMDb      string    `json:"imdb"`
		Freeleech bool      `json:"freeleech"`
		Download  string    `json:"download"`
		Details   string    `json:"details"`
	} `json:"results"`
}

func parse(body []byte) ([]searching.Release, error) {
	var l listing
	if err := json.Unmarshal(body, &l); err != nil {
		return nil, &searching.Failing{Err: fmt.Errorf("read the search: %w", err)}
	}
	out := make([]searching.Release, 0, len(l.Results))
	for _, r := range l.Results {
		hash, err := bencode.ParseInfoHash(r.InfoHash)
		if err != nil {
			return nil, &searching.Failing{Err: err}
		}
		rel := searching.Release{
			Title: r.Name, Size: r.Size, Published: r.Added, Seeders: r.Seeders, Leechers: r.Leechers, Grabs: r.Snatched,
			Link: searching.LinkOf(r.Download), InfoHash: &hash, Categories: []int{int(newznab.Movies.ID)},
			IDs: map[string]string{"imdbid": r.IMDb}, Factors: searching.Factors{Download: 1, Upload: 1},
			Floor: searching.Floor{MinRatio: 1, MinSeed: 48 * time.Hour}, Details: r.Details,
		}
		if r.Freeleech {
			rel.Factors.Download = 0
			rel.Flags = []searching.Flag{searching.Freeleech}
		}
		out = append(out, rel)
	}
	return out, nil
}
