package metadata_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/works"
)

type credentials struct {
	Key string `json:"apikey"`
	Pin string `json:"pin,omitempty"`
}

// door is a provider that logs in: POST /login trades a key it accepts,
// and the PIN that goes with it, for a token that lasts life, and every
// other path answers 401 without a live token.
type door struct {
	mu      sync.Mutex
	accepts map[string]string
	life    time.Duration
	// refuse answers 401 to every token, live or not.
	refuse bool
	// login answers every login with this status when set.
	login  int
	logins []credentials
	tokens map[string]time.Time
	// gets is the credentials behind the token each other request carried.
	gets []credentials
}

func newDoor(accepts map[string]string) *door {
	return &door{accepts: accepts, life: time.Hour, tokens: map[string]time.Time{}}
}

func (d *door) handle(n int, w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if r.URL.Path == "/login" {
		d.logIn(w, r)
		return
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	expires, live := d.tokens[token]
	if !live || !time.Now().Before(expires) || d.refuse {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":"failure","message":"Unauthorized"}`))
		return
	}
	var i int
	_, _ = fmt.Sscanf(token, "token-%d", &i)
	d.gets = append(d.gets, d.logins[i-1])
	ok(w, n)
}

func (d *door) logIn(w http.ResponseWriter, r *http.Request) {
	var c credentials
	_ = json.NewDecoder(r.Body).Decode(&c)
	d.logins = append(d.logins, c)
	pin, known := d.accepts[c.Key]
	switch {
	case d.login != 0:
		w.WriteHeader(d.login)
		return
	case !known:
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":"failure","message":"Unauthorized: unknown apikey"}`))
		return
	case c.Pin != pin:
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":"failure","message":"Unauthorized: wrong pin"}`))
		return
	}
	token := fmt.Sprintf("token-%d", len(d.logins))
	expires := time.Now().Add(d.life)
	d.tokens[token] = expires
	_, _ = fmt.Fprintf(w, `{"token":%q,"expires":%d}`, token, expires.Unix())
}

// expire ends every token the door issued, as when the provider restarts.
func (d *door) expire() {
	d.mu.Lock()
	defer d.mu.Unlock()
	clear(d.tokens)
}

func (d *door) set(f func(d *door)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	f(d)
}

func (d *door) loggedIn() []credentials {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.logins)
}

func (d *door) lastGet() credentials {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.gets[len(d.gets)-1]
}

func gate(base *url.URL) metadata.Provider {
	return metadata.Provider{
		ID: "gate", Label: "Gate", Rate: 20, Burst: 20,
		Auth: &metadata.Auth{
			Pin: true,
			Login: func(key, pin string) (*url.URL, []byte) {
				body, _ := json.Marshal(credentials{Key: key, Pin: pin}) //nolint:gosec // the login carries the key by design
				return base.JoinPath("login"), body
			},
			Token: func(body []byte) (string, time.Time, error) {
				var t struct {
					Token   string `json:"token"`
					Expires int64  `json:"expires"`
				}
				if err := json.Unmarshal(body, &t); err != nil {
					return "", time.Time{}, fmt.Errorf("read the login: %w", err)
				}
				return t.Token, time.Unix(t.Expires, 0), nil
			},
			Refused: func(_ int, body []byte) metadata.Credential {
				if bytes.Contains(body, []byte("pin")) {
					return metadata.CredentialPin
				}
				return metadata.CredentialKey
			},
		},
	}
}

func keyed(key, pin string) metadata.Ask {
	o := metadata.AskOptions{Keys: map[metadata.ProviderID]string{"gate": key}}
	if pin != "" {
		o.Pins = map[metadata.ProviderID]string{"gate": pin}
	}
	return metadata.NewAsk(o)
}

func (s *sent) paths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.reqs))
	for i, r := range s.reqs {
		out[i] = r.URL.Path
	}
	return out
}

// doorLine is a Line to a door accepting key "k", with no PIN.
func doorLine(t *testing.T) (*metadata.Line, *door, *url.URL, *sent) {
	t.Helper()
	d := newDoor(map[string]string{"k": ""})
	lines, base, s := serve(t, d.handle)
	return lines.Line(gate(base)), d, base, s
}

func getOnce(t *testing.T, line *metadata.Line, ask metadata.Ask, base *url.URL) ([]metadata.Outcome, error) {
	t.Helper()
	ask, outcomes := metadata.Tallied(ask)
	err := line.Get(t.Context(), ask, metadata.Request{URL: at(base, "/series/1", nil)}, &body{})
	return outcomes(), err
}

func TestBearerLoginReusesTheToken(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		line, d, base, s := doorLine(t)
		for range 2 {
			if o, err := getOnce(t, line, keyed("k", ""), base); err != nil || len(o) != 1 || o[0].Kind != metadata.OK {
				t.Fatalf("Get = %+v, %v; want one OK", o, err)
			}
		}
		if want := []string{"/login", "/series/1", "/series/1"}; !slices.Equal(s.paths(), want) {
			t.Fatalf("sent %v, want one login and two calls on its token", s.paths())
		}
		if got := s.reqs[0]; got.Method != http.MethodPost || got.Header.Get("Content-Type") != "application/json" || got.URL.RawQuery != "" {
			t.Fatalf("the login went as %s %s with %q, want a JSON POST with nothing on the query", got.Method, got.URL, got.Header.Get("Content-Type"))
		}
		if l := d.loggedIn(); l[0] != (credentials{Key: "k"}) {
			t.Fatalf("logged in with %+v, want key k and no PIN", l[0])
		}
		time.Sleep(time.Hour)
		if _, err := getOnce(t, line, keyed("k", ""), base); err != nil {
			t.Fatal(err)
		}
		if want := []string{"/login", "/series/1", "/series/1", "/login", "/series/1"}; !slices.Equal(s.paths(), want) {
			t.Fatalf("an hour on, sent %v, want a fresh login before the expired token is sent", s.paths())
		}
	})
}

func TestLoginWaitsOnTheBucket(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newDoor(map[string]string{"k": ""})
		lines, base, s := serve(t, d.handle)
		p := gate(base)
		p.Rate, p.Burst = 1, 1
		if _, err := getOnce(t, lines.Line(p), keyed("k", ""), base); err != nil {
			t.Fatal(err)
		}
		if s.count() != 2 || s.at[1].Sub(s.at[0]) != time.Second {
			t.Fatalf("sent at %v, want the call a second after the login at one request a second", s.at)
		}
	})
}

func TestA401RelogsInOnceAndRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		line, d, base, s := doorLine(t)
		if _, err := getOnce(t, line, keyed("k", ""), base); err != nil {
			t.Fatal(err)
		}
		d.expire()
		if o, err := getOnce(t, line, keyed("k", ""), base); err != nil || len(o) != 1 || o[0].Kind != metadata.OK {
			t.Fatalf("Get after the token expired = %+v, %v; want one OK", o, err)
		}
		if want := []string{"/login", "/series/1", "/series/1", "/login", "/series/1"}; !slices.Equal(s.paths(), want) {
			t.Fatalf("sent %v, want the 401 followed by one login and the call again", s.paths())
		}
	})
}

func TestASecond401IsRefused(t *testing.T) {
	t.Run("on a cached token", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			line, d, base, s := doorLine(t)
			if _, err := getOnce(t, line, keyed("k", ""), base); err != nil {
				t.Fatal(err)
			}
			d.set(func(d *door) { d.refuse = true })
			o, err := getOnce(t, line, keyed("k", ""), base)
			if se := sourceError(t, err, works.ErrRefused); se.Words != "refused the key: 401 Unauthorized" {
				t.Fatalf("words = %q", se.Words)
			}
			if len(o) != 1 || o[0].Kind != metadata.Refused {
				t.Fatalf("outcomes = %+v, want one Refused", o)
			}
			if want := []string{"/login", "/series/1", "/series/1", "/login", "/series/1"}; !slices.Equal(s.paths(), want) {
				t.Fatalf("sent %v, want one login after the first 401 and no try after the second", s.paths())
			}
		})
	})
	t.Run("on a token this call minted", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			line, d, base, s := doorLine(t)
			d.set(func(d *door) { d.refuse = true })
			if o, err := getOnce(t, line, keyed("k", ""), base); !errors.Is(err, works.ErrRefused) || len(o) != 1 || o[0].Kind != metadata.Refused {
				t.Fatalf("Get = %+v, %v; want one Refused", o, err)
			}
			if want := []string{"/login", "/series/1"}; !slices.Equal(s.paths(), want) {
				t.Fatalf("sent %v, want no second login for a token minted for this call", s.paths())
			}
		})
	})
}

func TestLoginOutcomeIsRecordedOnce(t *testing.T) {
	for _, c := range []struct {
		name      string
		key, pin  string
		login     int
		kind      metadata.OutcomeKind
		err       error
		words     string
		requested []string
	}{
		{"a refused key", "typo", "", 0, metadata.Refused, works.ErrRefused, "refused the key: 401 Unauthorized", []string{"/login"}},
		{"a refused PIN", "k", "0000", 0, metadata.PinRefused, works.ErrRefused, "refused the PIN", []string{"/login"}},
		{"no key", "", "", 0, metadata.Refused, works.ErrRefused, "can't be called without a key", nil},
		{"a login that fails each try", "k", "", http.StatusBadGateway, metadata.Unreachable, works.ErrUnavailable, "responded with 502 Bad Gateway", []string{"/login", "/login", "/login"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				line, d, base, s := doorLine(t)
				d.set(func(d *door) { d.login = c.login })
				o, err := getOnce(t, line, keyed(c.key, c.pin), base)
				if se := sourceError(t, err, c.err); se.Words != c.words {
					t.Fatalf("words = %q, want %q", se.Words, c.words)
				}
				if len(o) != 1 || o[0].Kind != c.kind || o[0].Provider != "gate" {
					t.Fatalf("outcomes = %+v, want one %d", o, c.kind)
				}
				if !slices.Equal(s.paths(), c.requested) {
					t.Fatalf("sent %v, want %v", s.paths(), c.requested)
				}
			})
		})
	}
}

func TestCallsOnAnExpiredTokenLogInOnce(t *testing.T) {
	for _, c := range []struct {
		name string
		end  func(d *door)
	}{
		{"expired by its time", func(*door) { time.Sleep(time.Hour) }},
		{"refused with a 401", (*door).expire},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				line, d, base, _ := doorLine(t)
				if _, err := getOnce(t, line, keyed("k", ""), base); err != nil {
					t.Fatal(err)
				}
				c.end(d)
				var calls sync.WaitGroup
				for range 8 {
					calls.Go(func() {
						if _, err := getOnce(t, line, keyed("k", ""), base); err != nil {
							t.Error(err)
						}
					})
				}
				calls.Wait()
				if n := len(d.loggedIn()); n != 2 {
					t.Fatalf("%d logins, want 2: the first call's, then one for all 8 calls on the ended token", n)
				}
			})
		})
	}
}
