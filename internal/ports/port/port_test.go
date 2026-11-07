package port

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/indexerfields"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/searching"
)

type codec int

func (codec) Options() []Option {
	return []Option{{Value: "1", Label: "H.264"}, {Value: "5", Label: "HEVC"}}
}

type sort string

func (sort) Options() []Option {
	return []Option{{Value: "seeders", Label: "Seeders"}, {Value: "added", Label: "Added"}}
}

type formSettings struct {
	UserPass
	PINField
	Codecs        []codec `port:"codecs" label:"Codecs" default:"1,5"`
	Sort          sort    `port:"sort" label:"Sort" default:"added"`
	Pages         int     `port:"pages" label:"Pages" min:"1" max:"5" default:"2"`
	FreeleechOnly bool    `port:"freeleechOnly" label:"Freeleech only" default:"true"`
}

type keySettings struct {
	PasskeyField
	Username string `port:"username" label:"Username"`
}

func search[S any](*Indexer[S], SearchCriteria) (Chain, error)          { return Chain{}, nil }
func parse[S any](*Indexer[S], *IndexerResponse) ([]TorrentInfo, error) { return nil, nil }
func doLogin[S any](*SignIn[S]) error                                   { return nil }
func entry(id ID) Entry {
	return Entry{
		ID: id, Name: "Lantern", Description: "Lantern is a private tracker for tests.", Privacy: Private,
		IndexerURLs:  []string{"https://lantern.test/"},
		Capabilities: Capabilities{MovieSearchParams: []Param{Q, ImdbID}, Categories: Categories(Map(1, newznab.Movies, "Movies"))},
	}
}

func TestDefineDerivesTheFormAndTheWordsFromTheSettingsAndTheFamily(t *testing.T) {
	p := Define(entry("lantern"), Impl[formSettings]{
		Login: CodeLogin[formSettings]{DoLogin: doLogin[formSettings]}, Movie: search[formSettings], Parse: parse[formSettings],
	})
	e := p.Entry()
	var names []string
	for _, f := range e.Fields {
		names = append(names, f.Name+":"+string(f.Type))
	}
	if got := strings.Join(names, " "); got != "username:text password:password pin:text codecs:multiselect sort:select pages:number freeleechOnly:checkbox authenticatorKey:password" {
		t.Errorf("fields = %s", got)
	}
	if e.Auth != FormWithCode || e.Key != "" || e.SignInWords != "username, password or PIN" {
		t.Errorf("auth %v, key %q, sign-in words %q", e.Auth, e.Key, e.SignInWords)
	}
	if key := e.Fields[len(e.Fields)-1]; key.Label != "Authenticator key" || !strings.HasPrefix(key.Help, "Lantern shows this key") || !indexerfields.Sealed(key) {
		t.Errorf("the authenticator key is %+v", key)
	}
	if pages := e.Fields[5]; *pages.Min != 1 || *pages.Max != 5 || pages.Default != "2" {
		t.Errorf("pages is %+v", pages)
	}

	k := Define(entry("ferry"), Impl[keySettings]{Login: NoLogin{}, Movie: search[keySettings], Parse: parse[keySettings]}).Entry()
	if k.Auth != StatelessKey || k.Key != "passkey" || k.SignInWords != "" || k.Auth.HoldsSession() {
		t.Errorf("a passkey port derived auth %v, key %q, sign-in words %q", k.Auth, k.Key, k.SignInWords)
	}
}

func TestDecodeFillsTheSettingsFromTheValuesAndTheDefaults(t *testing.T) {
	ss, err := settingsOf(reflect.TypeFor[formSettings]())
	if err != nil {
		t.Fatal(err)
	}
	got, err := decode[formSettings](ss, map[string]string{"username": "sam", "password": "pw", "codecs": "5", "freeleechOnly": "false"})
	want := formSettings{UserPass: UserPass{Username: "sam", Password: "pw"}, Codecs: []codec{5}, Sort: "added", Pages: 2}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("decode = %+v, %v; want %+v", got, err, want)
	}
	if _, err := decode[formSettings](ss, map[string]string{"pages": "two"}); err == nil {
		t.Error("decode took pages=two")
	}
}

type (
	untagged  struct{ Name string }
	unlabeled struct {
		Name string `port:"name"`
	}
	badBounds struct {
		Pages int `port:"pages" label:"Pages" min:"5" max:"1"`
	}
	boundedText struct {
		Name string `port:"name" label:"Name" max:"3"`
	}
	twice struct {
		UserPass
		Username string `port:"username" label:"User"`
	}
	noOptions struct {
		Speed float64 `port:"speed" label:"Speed"`
	}
	badDefault struct {
		Sort sort `port:"sort" label:"Sort" default:"name"`
	}
	unknownFlag struct {
		Name string `port:"name,secret" label:"Name"`
	}
	secret struct {
		Token string `port:"token" label:"Token"`
	}
)

func TestDefinePanicsOnAMisdeclaredPort(t *testing.T) {
	slow := entry("slow")
	slow.RateLimit = time.Second
	noURL := entry("nourl")
	noURL.IndexerURLs = nil
	badURL := entry("badurl")
	badURL.IndexerURLs = []string{"lantern.test"}
	tvOnly := entry("tv")
	tvOnly.Capabilities.TvSearchParams = []Param{Q}
	derived := entry("derived")
	derived.Key = "passkey"
	for name, define := range map[string]func(){
		"a rate limit below 2 s": func() {
			Define(slow, Impl[keySettings]{Login: NoLogin{}, Movie: search[keySettings], Parse: parse[keySettings]})
		},
		"no address": func() {
			Define(noURL, Impl[keySettings]{Login: NoLogin{}, Movie: search[keySettings], Parse: parse[keySettings]})
		},
		"an address with no scheme": func() {
			Define(badURL, Impl[keySettings]{Login: NoLogin{}, Movie: search[keySettings], Parse: parse[keySettings]})
		},
		"a mode with no generator": func() {
			Define(tvOnly, Impl[keySettings]{Login: NoLogin{}, Movie: search[keySettings], Parse: parse[keySettings]})
		},
		"a generator with no mode": func() {
			Define(entry("x"), Impl[keySettings]{Login: NoLogin{}, Movie: search[keySettings], TV: search[keySettings], Parse: parse[keySettings]})
		},
		"no parser": func() { Define(entry("x"), Impl[keySettings]{Login: NoLogin{}, Movie: search[keySettings]}) },
		"no family": func() { Define(entry("x"), Impl[keySettings]{Movie: search[keySettings], Parse: parse[keySettings]}) },
		"a derived field set": func() {
			Define(derived, Impl[keySettings]{Login: NoLogin{}, Movie: search[keySettings], Parse: parse[keySettings]})
		},
		"a secret marked as no key": func() {
			Define(entry("x"), Impl[secret]{Login: NoLogin{}, Movie: search[secret], Parse: parse[secret]})
		},
		"a sign-in with no fields": func() {
			Define(entry("x"), Impl[keySettings]{Login: FormLogin[keySettings]{DoLogin: doLogin[keySettings]}, Movie: search[keySettings], Parse: parse[keySettings]})
		},
		"a sign-in with no DoLogin": func() {
			Define(entry("x"), Impl[formSettings]{Login: TokenLogin[formSettings]{}, Movie: search[formSettings], Parse: parse[formSettings]})
		},
		"a family over other settings": func() {
			Define(entry("x"), Impl[formSettings]{Login: FormLogin[keySettings]{DoLogin: doLogin[keySettings]}, Movie: search[formSettings], Parse: parse[formSettings]})
		},
		"an untagged field": func() {
			Define(entry("x"), Impl[untagged]{Login: NoLogin{}, Movie: search[untagged], Parse: parse[untagged]})
		},
		"a field with no label": func() {
			Define(entry("x"), Impl[unlabeled]{Login: NoLogin{}, Movie: search[unlabeled], Parse: parse[unlabeled]})
		},
		"min above max": func() {
			Define(entry("x"), Impl[badBounds]{Login: NoLogin{}, Movie: search[badBounds], Parse: parse[badBounds]})
		},
		"bounds on text": func() {
			Define(entry("x"), Impl[boundedText]{Login: NoLogin{}, Movie: search[boundedText], Parse: parse[boundedText]})
		},
		"a name twice": func() {
			Define(entry("x"), Impl[twice]{Login: CookieLogin{}, Movie: search[twice], Parse: parse[twice]})
		},
		"a type with no form": func() {
			Define(entry("x"), Impl[noOptions]{Login: NoLogin{}, Movie: search[noOptions], Parse: parse[noOptions]})
		},
		"a default not offered": func() {
			Define(entry("x"), Impl[badDefault]{Login: NoLogin{}, Movie: search[badDefault], Parse: parse[badDefault]})
		},
		"an unknown tag option": func() {
			Define(entry("x"), Impl[unknownFlag]{Login: NoLogin{}, Movie: search[unknownFlag], Parse: parse[unknownFlag]})
		},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("Define passed it")
				}
			}()
			define()
		})
	}
}

func TestTheRegistryListsHidesAndRetires(t *testing.T) {
	lantern := entry("lantern")
	lantern.Replaces = []string{"lantern-yml"}
	ferry := entry("ferry")
	ferry.Name = "Ferry"
	define := func(e Entry) Port {
		return Define(e, Impl[keySettings]{Login: NoLogin{}, Movie: search[keySettings], Parse: parse[keySettings]})
	}
	r := NewRegistry([]Port{define(lantern), define(ferry)}, []Retired{{ID: "brook", Name: "Brook", Reason: "the site closed"}}, nil)
	if es := r.Entries(); len(es) != 2 || es[0].Name != "Ferry" || es[1].Name != "Lantern" {
		t.Errorf("entries = %+v, want Ferry then Lantern", es)
	}
	if id, ok := r.Hiding("lantern-yml"); !ok || id != "lantern" {
		t.Errorf("Hiding(lantern-yml) = %q, %v", id, ok)
	}
	d := driver{r: r}
	_, err := d.Search(t.Context(), &searching.Site{Port: "brook"}, searching.Query{})
	if s, ok := errors.AsType[*searching.Stopped](err); !ok || s.Cause != searching.PortRetired || s.Text != "the site closed" {
		t.Errorf("a retired port's search = %v, want port_retired with its reason", err)
	}
	if _, err := d.Recent(t.Context(), &searching.Site{Port: "gone"}); !isFailing(err) {
		t.Errorf("an unknown port's recent = %v, want failing", err)
	}
	if _, err := d.Captcha(t.Context(), &searching.Site{Port: "lantern"}); !isFailing(err) {
		t.Errorf("a captcha = %v, want failing", err)
	}

	for name, regs := range map[string]func(){
		"an id twice": func() { NewRegistry([]Port{define(lantern), define(lantern)}, nil, nil) },
		"a catalog id twice": func() {
			f := ferry
			f.Aliases = []string{"lantern-yml"}
			NewRegistry([]Port{define(lantern), define(f)}, nil, nil)
		},
		"a live id retired too": func() { NewRegistry([]Port{define(lantern)}, []Retired{{ID: "lantern"}}, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("NewRegistry took it")
				}
			}()
			regs()
		})
	}
}

func isFailing(err error) bool {
	_, ok := errors.AsType[*searching.Failing](err)
	return ok
}

func TestReadUpstreamNeedsItsCommitDateAndFiles(t *testing.T) {
	u := ReadUpstream([]byte(`{"commit": "ef9c1824", "date": "2026-11-06", "family": "", "files": {"hdbits.go": ["src/HDBits.cs"]}}`))
	if u.Commit != "ef9c1824" || !u.Date.Equal(time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)) || u.Files["hdbits.go"][0] != "src/HDBits.cs" {
		t.Errorf("ReadUpstream = %+v", u)
	}
	defer func() {
		if recover() == nil {
			t.Error("ReadUpstream took an upstream.json with no files")
		}
	}()
	ReadUpstream([]byte(`{"commit": "ef9c1824", "date": "2026-11-06"}`))
}
