package identityapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
)

type answer struct {
	status     int
	body       string
	retryAfter string
}

func answerOf(w *httptest.ResponseRecorder) answer {
	return answer{status: w.Code, body: w.Body.String(), retryAfter: w.Header().Get("Retry-After")}
}

func (n *node) signInFrom(from, email, pin string, headers map[string]string) answer {
	n.t.Helper()
	return answerOf(n.do(call{method: http.MethodPost, target: "/api/v1/session", from: from, headers: headers,
		body: `{"email":"` + email + `","pin":"` + pin + `"}`}))
}

func waitProblem(code, title string, until time.Time, seconds int) answer {
	body := fmt.Sprintf(`{"status":429,"code":%q,"title":%q,"detail":%q}`, code, title, until.UTC().Format(time.RFC3339)) + "\n"
	return answer{status: 429, body: body, retryAfter: strconv.Itoa(seconds)}
}

var (
	wrongAnswer = answer{status: 401, body: pinWrong}
	offAnswer   = answer{status: 403, body: problem(403, "pin_off", "This PIN is turned off. Use a passkey or ask an Admin for a Reset link.")}
)

func pinWaitAnswer(until time.Time, seconds int) answer {
	return waitProblem("pin_wait", "Too many wrong PINs. Wait before trying again.", until, seconds)
}

func tooManyAnswer(until time.Time, seconds int) answer {
	return waitProblem("too_many_attempts", "Too many failed attempts from this address. Wait before trying again.", until, seconds)
}

func want(t *testing.T, step string, got, want answer) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %d %q Retry-After %q, want %d %q Retry-After %q", step, got.status, got.body, got.retryAfter, want.status, want.body, want.retryAfter)
	}
}

// TestFiveWrongPINsWaitAndTenTurnThePINOff walks one Account and one email
// with no Account through the same guesses side by side: both answer the
// same at every step, so the answers reveal nothing about who has an
// Account.
func TestFiveWrongPINsWaitAndTenTurnThePINOff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		n.claimed()
		both := func(step, pin string, w answer) {
			t.Helper()
			want(t, step+" (Account)", n.signInFrom("192.0.2.1:1", "admin@example.com", pin, nil), w)
			want(t, step+" (no Account)", n.signInFrom("192.0.2.2:1", "nobody@example.com", pin, nil), w)
		}
		for i := 1; i <= 5; i++ {
			both(fmt.Sprintf("wrong PIN %d", i), "9999", wrongAnswer)
		}
		until := time.Now().Add(15 * time.Minute)
		both("a wrong PIN during the wait", "9999", pinWaitAnswer(until, 900))
		want(t, "the right PIN during the wait", n.signInFrom("192.0.2.1:1", "admin@example.com", "1234", nil), pinWaitAnswer(until, 900))
		time.Sleep(5 * time.Minute)
		both("five minutes into the wait", "9999", pinWaitAnswer(until, 600))
		for i := 6; i <= 10; i++ {
			time.Sleep(time.Until(until))
			both(fmt.Sprintf("wrong PIN %d, once the wait passed", i), "9999", wrongAnswer)
			until = time.Now().Add(15 * time.Minute)
			if i < 10 {
				both(fmt.Sprintf("straight after wrong PIN %d", i), "9999", pinWaitAnswer(until, 900))
			}
		}
		time.Sleep(time.Hour)
		both("an hour after the tenth", "9999", offAnswer)
		want(t, "the right PIN once it is off", n.signInFrom("192.0.2.1:1", "admin@example.com", "1234", nil), offAnswer)
	})
}

func TestTheRightPINClearsTheCount(t *testing.T) {
	n := newNode(t)
	n.claimed()
	for range 4 {
		want(t, "a wrong PIN", n.signInFrom("192.0.2.1:1", "admin@example.com", "9999", nil), wrongAnswer)
	}
	if got := n.signInFrom("192.0.2.1:1", "admin@example.com", "1234", nil); got.status != http.StatusOK {
		t.Fatalf("the right PIN = %d %s, want 200", got.status, got.body)
	}
	for range 4 {
		want(t, "a wrong PIN after the right one", n.signInFrom("192.0.2.1:1", "admin@example.com", "9999", nil), wrongAnswer)
	}
	if got := n.signInFrom("192.0.2.1:1", "admin@example.com", "1234", nil); got.status != http.StatusOK {
		t.Fatalf("the right PIN after four more wrong = %d %s, want 200", got.status, got.body)
	}
}

// TestParallelWrongPINsAreCountedBeforeTheyAreHashed sends fifty wrong
// PINs for one Account at once. Each is counted before it queues for one
// of the two hash slots, so only the five before the wait are ever
// verified, far below the hard stop.
func TestParallelWrongPINsAreCountedBeforeTheyAreHashed(t *testing.T) {
	n := newNode(t)
	n.claimed()
	var mu sync.Mutex
	answers := map[string]int{}
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			got := n.signInFrom(fmt.Sprintf("192.0.2.%d:1", i+1), "admin@example.com", "9999", nil)
			var p struct{ Code string }
			_ = json.Unmarshal([]byte(got.body), &p)
			mu.Lock()
			answers[p.Code]++
			mu.Unlock()
		})
	}
	wg.Wait()
	if answers["pin_wrong"] != 5 || answers["pin_wait"] != 45 || len(answers) != 2 {
		t.Fatalf("answers = %v, want 5 pin_wrong and 45 pin_wait", answers)
	}
	w := n.signInFrom("192.0.2.99:1", "admin@example.com", "1234", nil)
	if w.status != http.StatusTooManyRequests {
		t.Fatalf("the right PIN after the burst = %d %s, want the wait", w.status, w.body)
	}
}

// TestEachAddressCountsItsFailures fails thirty times from one address,
// each time for another email so no Account waits. Successes never count,
// an IPv6 address counts with its /64, and the window lets go.
func TestEachAddressCountsItsFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		n.claimed()
		signedIn := func(step, from string) {
			t.Helper()
			if got := n.signInFrom(from, "admin@example.com", "1234", nil); got.status != http.StatusOK {
				t.Fatalf("%s: %d %s, want 200", step, got.status, got.body)
			}
		}
		fail := func(from string) {
			t.Helper()
			for i := range 30 {
				want(t, "a failure", n.signInFrom(from, fmt.Sprintf("guess%d@example.com", i), "9999", nil), wrongAnswer)
			}
		}
		for range 40 {
			signedIn("a success never counts", "[2001:db8:1:1::1]:1")
		}
		fail("[2001:db8:1:1::1]:1")
		until := time.Now().Add(15 * time.Minute)
		blocked := tooManyAnswer(until, 900)
		want(t, "the right PIN past the limit", n.signInFrom("[2001:db8:1:1::1]:1", "admin@example.com", "1234", nil), blocked)
		want(t, "another address in the /64", n.signInFrom("[2001:db8:1:1:ffff:ffff:ffff:ffff]:1", "admin@example.com", "1234", nil), blocked)
		signedIn("the next /64", "[2001:db8:1:2::1]:1")

		fail("192.0.2.10:1")
		want(t, "the IPv4 address past the limit", n.signInFrom("192.0.2.10:2", "admin@example.com", "1234", nil), blocked)
		signedIn("its neighbor", "192.0.2.11:1")

		time.Sleep(15 * time.Minute)
		signedIn("once the window passed", "[2001:db8:1:1::1]:1")
		signedIn("once the window passed", "192.0.2.10:1")
	})
}

// TestOnlyATrustedProxyNamesTheClient puts SlipStream behind a proxy at
// 10.0.0.0/8 that forwards the client's address and scheme.
func TestOnlyATrustedProxyNamesTheClient(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var proxy api.Proxy
		if err := proxy.UnmarshalText([]byte("10.0.0.0/8")); err != nil {
			t.Fatal(err)
		}
		n := newNodeWith(t, nodeOptions{proxies: []api.Proxy{proxy}})
		n.claimed()
		forwarded := func(client string) map[string]string {
			return map[string]string{"X-Forwarded-For": "203.0.113.200, " + client + ", 10.9.9.9", "X-Forwarded-Proto": "http, https"}
		}
		for i := range 30 {
			want(t, "a failure through the proxy", n.signInFrom("10.1.2.3:1", fmt.Sprintf("guess%d@example.com", i), "9999", forwarded("198.51.100.7")), wrongAnswer)
		}
		until := time.Now().Add(15 * time.Minute)
		want(t, "that client again", n.signInFrom("10.1.2.3:1", "admin@example.com", "1234", forwarded("198.51.100.7")), tooManyAnswer(until, 900))
		other := n.do(call{method: http.MethodPost, target: "/api/v1/session", from: "10.1.2.3:1", headers: forwarded("198.51.100.8"),
			body: `{"email":"admin@example.com","pin":"1234"}`})
		if other.Code != http.StatusOK || !strings.HasSuffix(other.Header().Get("Set-Cookie"), "; HttpOnly; Secure; SameSite=Lax") {
			t.Fatalf("another client through the proxy = %d %s, Set-Cookie %q; want 200 and a Secure cookie", other.Code, other.Body, other.Header().Get("Set-Cookie"))
		}
		direct := n.do(call{method: http.MethodPost, target: "/api/v1/session", from: "192.0.2.50:1", headers: forwarded("198.51.100.7"),
			body: `{"email":"admin@example.com","pin":"1234"}`})
		if direct.Code != http.StatusOK || strings.Contains(direct.Header().Get("Set-Cookie"), "Secure") {
			t.Fatalf("a peer that is no proxy claiming the blocked client = %d %s, Set-Cookie %q; want 200 and its own address", direct.Code, direct.Body, direct.Header().Get("Set-Cookie"))
		}
		for i := range 30 {
			forged := map[string]string{"X-Forwarded-For": fmt.Sprintf("198.51.100.%d", i)}
			want(t, "a failure with a forged header", n.signInFrom("192.0.2.51:1", fmt.Sprintf("guess%d@example.com", i), "9999", forged), wrongAnswer)
		}
		want(t, "the peer that forged it", n.signInFrom("192.0.2.51:1", "admin@example.com", "1234", forwarded("198.51.100.9")), tooManyAnswer(until, 900))
	})
}

func (n *node) patchSettings(apiKey, body string) *httptest.ResponseRecorder {
	n.t.Helper()
	return n.do(call{method: http.MethodPatch, target: "/api/v1/identity/settings", body: body, headers: map[string]string{"X-Api-Key": apiKey}})
}

func TestALifetimeChangeAppliesWithoutARestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t)
		res, a := n.claimed()
		time.Sleep(2 * time.Hour)
		if w := n.getSession(a); w.Code != http.StatusOK {
			t.Fatalf("getSession two hours on = %d %s", w.Code, w.Body)
		}
		if w := n.patchSettings(res.APIKey, `{"sessionLifetimeSeconds":3600}`); w.Code != http.StatusOK {
			t.Fatalf("updateIdentitySettings = %d %s", w.Code, w.Body)
		}
		time.Sleep(time.Hour + time.Second)
		expect(t, n.getSession(a), 401, signedOut)
		w := n.signIn("admin@example.com", "1234")
		if got := w.Header().Get("Set-Cookie"); !strings.Contains(got, "; Max-Age=3600;") {
			t.Fatalf("Set-Cookie = %q, want the new lifetime", got)
		}
		b := sessionOf(t, w)
		time.Sleep(59 * time.Minute)
		if w := n.getSession(b); w.Code != http.StatusOK {
			t.Fatalf("getSession inside the new lifetime = %d %s", w.Code, w.Body)
		}
	})
}

func TestEverySettingHasAFloorAndACeiling(t *testing.T) {
	n := newNode(t)
	res, _ := n.claimed()
	w := n.do(call{method: http.MethodGet, target: "/api/v1/identity/settings", headers: map[string]string{"X-Api-Key": res.APIKey}})
	var defaults map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &defaults); err != nil || w.Code != http.StatusOK {
		t.Fatalf("getIdentitySettings = %d %s", w.Code, w.Body)
	}
	for _, b := range identity.Bounds {
		if got := defaults[b.Field]; got != float64(b.Default) {
			t.Errorf("%s defaults to %v, want %d", b.Field, got, b.Default)
		}
	}
	companion := map[string]string{"pinWaitAfter": `,"pinOffAfter":20`, "pinOffAfter": `,"pinWaitAfter":3`}
	for _, b := range identity.Bounds {
		for v, ok := range map[int64]bool{b.Floor - 1: false, b.Floor: true, b.Ceiling: true, b.Ceiling + 1: false} {
			w := n.patchSettings(res.APIKey, fmt.Sprintf(`{%q:%d%s}`, b.Field, v, companion[b.Field]))
			switch {
			case ok && (w.Code != http.StatusOK || !strings.Contains(w.Body.String(), fmt.Sprintf(`%q:%d,`, b.Field, v))):
				t.Errorf("%s = %d: %d %s, want it saved", b.Field, v, w.Code, w.Body)
			case !ok && (w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(),
				fmt.Sprintf(`"code":"setting_out_of_range","title":"A setting is outside its range","errors":[{"location":"body.%s","message":"expected %d to %d"}`, b.Field, b.Floor, b.Ceiling))):
				t.Errorf("%s = %d: %d %s, want setting_out_of_range", b.Field, v, w.Code, w.Body)
			}
		}
	}
	expect(t, n.patchSettings(res.APIKey, `{"pinWaitAfter":6,"pinOffAfter":6}`), 422,
		`{"status":422,"code":"setting_out_of_range","title":"A setting is outside its range","errors":[{"location":"body.pinOffAfter","message":"expected more than pinWaitAfter"}]}`+"\n")
}

func TestTheAddressIsAnOriginWithAName(t *testing.T) {
	n := newNode(t)
	res, _ := n.claimed()
	address := func(w *httptest.ResponseRecorder) string {
		t.Helper()
		var s struct{ Address, SuggestedAddress string }
		if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil || w.Code != http.StatusOK {
			t.Fatalf("settings = %d %s", w.Code, w.Body)
		}
		return s.Address
	}
	get := n.do(call{method: http.MethodGet, target: "/api/v1/identity/settings", host: "media.example.com:8484", headers: map[string]string{"X-Api-Key": res.APIKey}})
	if address(get) != "" || !strings.Contains(get.Body.String(), `"suggestedAddress":"http://media.example.com:8484"`) {
		t.Fatalf("an unset address = %s, want it empty and this request's address suggested", get.Body)
	}
	if got := address(n.patchSettings(res.APIKey, `{"address":"HTTPS://Media.Example.com:443/"}`)); got != "https://media.example.com" {
		t.Fatalf("address = %q, want https://media.example.com", got)
	}
	expect(t, n.patchSettings(res.APIKey, `{"address":"http://192.168.1.4:8484"}`), 422,
		problem(422, "address_ip", "The address must be a name, not an IP address"))
	expect(t, n.patchSettings(res.APIKey, `{"address":"ftp://media.example.com"}`), 422,
		`{"status":422,"code":"invalid","title":"Some fields are invalid","errors":[{"location":"body.address","message":"expected http:// or https:// then a host name and an optional port"}]}`+"\n")
	if got := address(n.patchSettings(res.APIKey, `{"pinWaitAfter":4}`)); got != "https://media.example.com" {
		t.Fatalf("address after another change = %q, want it kept", got)
	}
}
