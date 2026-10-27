package site_test

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/site/definitions"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// controlled is a Controller whose one Searching Node reaches w's
// fixture-site, with every catalog site's definition added as a custom
// one.
func controlled(t *testing.T, w catalogWorld) *indexers.Indexers {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), indexers.Schema())
	self := role.NodeID{UUID: storage.NewUUID()}
	if err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, self, "home")
	}); err != nil {
		t.Fatal(err)
	}
	x := indexers.New(indexers.Config{
		Library:   lib,
		Sealer:    secrettest.Sealer(t),
		Roster:    indexerstest.Alone(self),
		Searching: func(role.NodeID) searching.Remote { return w.n },
		Types:     map[works.TypeID]indexers.Facet{indexerstest.Type: indexerstest.Facet{}},
		Changed:   discard[indexers.Changed]{},
		Status:    discard[indexers.StatusChanged]{},
		Catalogs:  discard[indexers.CatalogChanged]{},
		Timings:   indexers.DefaultTimings(),
		Log:       slog.New(slog.DiscardHandler),
	})
	for _, label := range []string{site.Larkspur, site.Quarry, site.Cinder} {
		src, err := fs.ReadFile(definitions.FS, label+".yml")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := x.AddCustom(t.Context(), identity.Admin{}, src); err != nil {
			t.Fatal(err)
		}
	}
	return x
}

// failedWith is err's new captcha when it is a failed Test whose sentence
// holds want and that carries one.
func failedWith(t *testing.T, err error, want string) indexers.Captcha {
	t.Helper()
	var failed *indexers.TestFailedError
	if !errors.As(err, &failed) || !strings.Contains(failed.Sentence, want) || failed.Captcha == nil {
		t.Fatalf("Create = %v (%+v), want a failed Test saying %q with a new captcha", err, failed, want)
	}
	return *failed.Captcha
}

func shows(t *testing.T, c indexers.Captcha) string {
	t.Helper()
	text, err := site.CaptchaText(c.Image)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestAnAdminAddsLarkspurByTypingItsCaptcha(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newCatalogWorld(t, faketest.InMemory)
		x := controlled(t, w)
		admin := identity.Admin{}
		d := indexers.Draft{
			Kind: indexers.Catalog, Name: "Larkspur", Definition: site.Larkspur, On: true, AutoSearch: true,
			Values: map[string]string{"username": site.Username, "password": site.Password(w.s, site.Larkspur), "sort": "seeders"},
		}
		_, err := x.Create(t.Context(), admin, d, nil)
		first := failedWith(t, err, "needs a captcha")
		_, err = x.Create(t.Context(), admin, d, &indexers.CaptchaAnswer{Attempt: first.Attempt, Text: "wrong"})
		second := failedWith(t, err, site.WrongCaptcha)
		if _, err := x.Create(t.Context(), admin, d, &indexers.CaptchaAnswer{Attempt: first.Attempt, Text: shows(t, first)}); !errors.Is(err, indexers.ErrCaptchaExpired) {
			t.Errorf("a second answer to one attempt = %v, want it expired", err)
		}
		other := d
		other.Values = map[string]string{"username": "someone", "password": "x"}
		if _, err := x.Create(t.Context(), admin, other, &indexers.CaptchaAnswer{Attempt: second.Attempt, Text: shows(t, second)}); !errors.Is(err, indexers.ErrCaptchaExpired) {
			t.Errorf("an answer for another username = %v, want it expired", err)
		}
		third, err := x.Captcha(t.Context(), admin, indexers.CaptchaFor{Draft: &d})
		if err != nil {
			t.Fatal(err)
		}
		v, err := x.Create(t.Context(), admin, d, &indexers.CaptchaAnswer{Attempt: third.Attempt, Text: shows(t, third)})
		if err != nil {
			t.Fatalf("Create with the code = %v", err)
		}
		def, ok := v.Source.(indexers.Definition)
		if !ok || !v.SignsIn || v.Session == nil || v.FeedHost != "" || def.BaseAddress != "https://"+site.Host(site.Larkspur)+"/" ||
			def.Plain["username"] != site.Username || def.Plain["sort"] != "seeders" || def.Plain["password"] != "" {
			t.Errorf("the Indexer = %+v, want it signed in with its session, at Larkspur's first link, the password sealed", v)
		}
		if _, err := x.Test(t.Context(), admin, v.ID, nil); err != nil {
			t.Errorf("a Test with the session it holds = %v, want a pass without a captcha", err)
		}
		expiring, err := x.Captcha(t.Context(), admin, indexers.CaptchaFor{Indexer: v.ID})
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(indexers.DefaultTimings().Captcha)
		if _, err := x.Test(t.Context(), admin, v.ID, &indexers.CaptchaAnswer{Attempt: expiring.Attempt, Text: shows(t, expiring)}); !errors.Is(err, indexers.ErrCaptchaExpired) {
			t.Errorf("an answer after five minutes = %v, want it expired", err)
		}
	})
}

func TestACatalogDraftTakesOnlyWhatItsDefinitionDoes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newCatalogWorld(t, faketest.InMemory)
		x := controlled(t, w)
		admin := identity.Admin{}
		quarry := indexers.Draft{Kind: indexers.Catalog, Name: "Quarry", Definition: site.Quarry, Values: map[string]string{"apikey": site.Key(w.s, site.Quarry)}}
		for _, tc := range []struct {
			name string
			edit func(*indexers.Draft)
			want func(error) bool
		}{
			{"a definition the custom ones lack before the catalog is downloaded", func(d *indexers.Draft) { d.Definition = "nowhere" }, func(err error) bool { return errors.Is(err, indexers.ErrCatalogUnavailable) }},
			{"an address it doesn't list", func(d *indexers.Draft) { d.BaseAddress = "https://elsewhere.test/" }, func(err error) bool { return errors.Is(err, indexers.ErrNotALink) }},
			{"a setting it doesn't have", func(d *indexers.Draft) { d.Values = map[string]string{"passkey": "x"} }, func(err error) bool {
				var v *indexers.ValueError
				return errors.As(err, &v) && v.Field == "passkey"
			}},
			{"a captcha it doesn't show", func(*indexers.Draft) {}, func(err error) bool { return errors.Is(err, indexers.ErrNoCaptcha) }},
		} {
			d := quarry
			tc.edit(&d)
			var err error
			if tc.name == "a captcha it doesn't show" {
				_, err = x.Captcha(t.Context(), admin, indexers.CaptchaFor{Draft: &d})
			} else {
				_, err = x.Create(t.Context(), admin, d, nil)
			}
			if !tc.want(err) {
				t.Errorf("%s: %v", tc.name, err)
			}
		}
		v, err := x.Create(t.Context(), admin, quarry, nil)
		if err != nil {
			t.Fatal(err)
		}
		if v.SignsIn || v.Session != nil || v.Caps.Modes["movie"] == nil {
			t.Errorf("Quarry = %+v, want it stateless with its movie search", v)
		}
		if _, err := x.Update(t.Context(), admin, v.ID, indexers.Edit{Values: map[string]string{"apikey": ""}}, nil); err != nil {
			t.Errorf("an edit that sends the sealed API key empty = %v, want it kept and the Test passed", err)
		}
		var failed *indexers.TestFailedError
		if _, err := x.Update(t.Context(), admin, v.ID, indexers.Edit{Values: map[string]string{"apikey": "nope"}}, nil); !errors.As(err, &failed) {
			t.Errorf("an edit to a wrong API key = %v, want its Test failed", err)
		}
		cinder := indexers.Draft{Kind: indexers.Catalog, Name: "Cinder", Definition: site.Cinder}
		if _, err := x.Create(t.Context(), admin, cinder, nil); !errors.Is(err, indexers.ErrCookieNeeded) {
			t.Errorf("Cinder without a cookie = %v, want ErrCookieNeeded", err)
		}
		cinder.Cookie = &indexers.PastedCookie{
			Cookie:    site.SessionCookie(w.s) + "; " + site.ClearanceCookie + "=" + site.Clearance(w.s, "seedbox", browser),
			UserAgent: browser,
		}
		if v, err := x.Create(t.Context(), admin, cinder, nil); err != nil || !v.SignsIn || v.Session == nil || !v.Session.Pasted {
			t.Errorf("Cinder with a pasted cookie = %+v, %v; want it signed in with the pasted session", v, err)
		}
	})
}
