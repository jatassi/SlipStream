package searching

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
)

func TestAJobAndItsAnswerCrossTheWireWhole(t *testing.T) {
	earned := time.UnixMilli(1_780_000_000_000).UTC()
	session := &Session{
		Cookies:   []Cookie{{Name: "uid", Value: "7", Domain: "site.test", Path: "/", Expires: earned.Add(time.Hour), Secure: true, HostOnly: true}},
		UserAgent: "Mozilla/5.0", Pasted: true, Earned: earned,
	}
	job := Job{
		ID: "j", Indexer: "i", Revision: 3, Kind: Torznab, Address: "https://site.test/torznab", Settings: map[string]string{"username": "u"},
		APIKey: "k", Session: session, Categories: []int{2000, 2040},
		Call: SearchCall{Query: Query{Mode: "tvsearch", IDs: map[string]string{"tvdbid": "1"}, Text: "Lantern", Season: 2, Episode: "5"}},
	}
	got, site, err := decodeJob(encodeJob(job))
	if err != nil || !reflect.DeepEqual(got, job) || site.Address.Host != "site.test" || site.APIKey != "k" {
		t.Fatalf("the job came across as %+v, %+v, %v; want %+v", got, site, err, job)
	}

	hash := bencode.InfoHash{1, 2, 3}
	answer := Answer{
		Started: earned, Outcome: &Stopped{Cause: RequestRejected, Text: "Function not available", SolverTried: true}, Session: session,
		Releases: []Release{{
			Title: "Lantern.S02E05", Size: 1 << 30, Published: earned, Seeders: 9, Leechers: 1, Grabs: 40,
			Link: LinkOf("https://site.test/dl/1"), Magnet: LinkOf("magnet:?xt=urn:btih:0102030000000000000000000000000000000000"), InfoHash: &hash,
			Categories: []int{5040}, IDs: map[string]string{"tvdbid": "1"}, Factors: Factors{Download: 0, Upload: 2},
			Floor: Floor{MinRatio: 1, MinSeed: 48 * time.Hour}, Details: "https://site.test/t/1",
			Source: &Source{ID: "12", Name: "Larkspur", Privacy: "private", Origin: "prowlarr"},
		}},
	}
	back, err := decodeAnswer(encodeAnswer(answer))
	if err != nil || !same(back, answer) {
		t.Fatalf("the answer came across as %+v, %v; want %+v", back, err, answer)
	}
	for _, outcome := range []error{nil, &Limited{RetryAfter: 2 * time.Second}} {
		a := Answer{Outcome: outcome, Report: &Report{Capabilities: Capabilities{Modes: map[string][]string{"search": {"q"}}, Categories: []int{2000}}, Items: 3, Sources: []string{"Larkspur"}}}
		if back, err := decodeAnswer(encodeAnswer(a)); err != nil || !same(back, a) {
			t.Fatalf("a Test's answer came across as %+v, %v; want %+v", back, err, a)
		}
	}
}

// same compares two answers field by field, and their outcomes by value.
func same(a, b Answer) bool {
	return a.Started.Equal(b.Started) && outcomeValue(a.Outcome) == outcomeValue(b.Outcome) &&
		reflect.DeepEqual(a.Session, b.Session) && reflect.DeepEqual(a.Releases, b.Releases) &&
		reflect.DeepEqual(a.Got, b.Got) && reflect.DeepEqual(a.Report, b.Report)
}

func outcomeValue(err error) string {
	if err == nil {
		return "ok"
	}
	return fmt.Sprintf("%#v", reflect.ValueOf(err).Elem().Interface())
}
