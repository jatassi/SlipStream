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

func tooManyAnswer(until time.Time) answer {
	return waitProblem("too_many_attempts", "Too many failed attempts from this address. Wait before trying again.", until, 900)
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
		notice := n.mail.Wait(t, "admin@example.com")
		synctest.Wait()
		if notice.Subject != "Your SlipStream PIN was turned off" || !strings.HasPrefix(notice.Text, "Someone typed the wrong PIN for admin@example.com 10 times in a row") {
			t.Fatalf("notice = %q %q", notice.Subject, notice.Text)
		}
		if sent := n.mail.Sent(); len(sent) != 1 {
			t.Fatalf("sent %d emails, want only the Admin's one notice", len(sent))
		}
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

// TestEachAddressCountsItsFailures fails for another email each time, so
// no Account waits.
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
		blocked := tooManyAnswer(until)
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
		want(t, "that client again", n.signInFrom("10.1.2.3:1", "admin@example.com", "1234", forwarded("198.51.100.7")), tooManyAnswer(until))
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
		want(t, "the peer that forged it", n.signInFrom("192.0.2.51:1", "admin@example.com", "1234", forwarded("198.51.100.9")), tooManyAnswer(until))
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
		if w := n.patchSettings(res.APIKey, `{"sessionLifetime":3600}`); w.Code != http.StatusOK {
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

func outOfRange(detail string) string {
	return fmt.Sprintf(`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":%q}`, detail) + "\n"
}

func TestEverySettingHasAFloorAndACeiling(t *testing.T) {
	n := newNode(t)
	res, _ := n.claimed()
	w := n.do(call{method: http.MethodGet, target: "/api/v1/identity/settings", headers: map[string]string{"X-Api-Key": res.APIKey}})
	var fields map[string]struct {
		Value     int64
		IsDefault bool
	}
	if err := json.Unmarshal(w.Body.Bytes(), &fields); err != nil || w.Code != http.StatusOK {
		t.Fatalf("getIdentitySettings = %d %s", w.Code, w.Body)
	}
	for _, s := range []struct {
		field                   string
		value, floor, ceiling   int64
		companion, outsideRange string
	}{
		{"pinWaitAfter", 5, 3, 10, `,"pinOffAfter":20`, "Between 3 and 10 wrong PINs."},
		{"pinWait", 900, 60, 86400, "", "Between 1 min and 1 day."},
		{"pinOffAfter", 10, 4, 20, `,"pinWaitAfter":3`, "Between 4 and 20 wrong PINs."},
		{"addressFailures", 30, 10, 1000, "", "Between 10 and 1000 attempts."},
		{"addressWindow", 900, 60, 86400, "", "Between 1 min and 1 day."},
		{"reconfirmWindow", 900, 60, 86400, "", "Between 1 min and 1 day."},
		{"sessionLifetime", 2592000, 3600, 31536000, "", "Between 1 h and 365 days."},
		{"signupLinkLifetime", 604800, 60, 2592000, "", "Between 1 min and 30 days."},
		{"resetLinkLifetime", 86400, 60, 604800, "", "Between 1 min and 7 days."},
		{"emailLinkLifetime", 86400, 60, 604800, "", "Between 1 min and 7 days."},
	} {
		if got := fields[s.field]; got.Value != s.value || !got.IsDefault {
			t.Errorf("%s = %+v, want the default %d", s.field, got, s.value)
		}
		for _, v := range []int64{s.floor, s.ceiling} {
			w := n.patchSettings(res.APIKey, fmt.Sprintf(`{%q:%d%s}`, s.field, v, s.companion))
			saved := map[string]struct {
				Key       string
				Value     int64
				IsDefault bool
			}{}
			if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil || w.Code != http.StatusOK ||
				saved[s.field].Key != "identity."+s.field || saved[s.field].Value != v || saved[s.field].IsDefault {
				t.Errorf("%s = %d: %d %s, want it saved", s.field, v, w.Code, w.Body)
			}
		}
		for _, v := range []int64{s.floor - 1, s.ceiling + 1} {
			expect(t, n.patchSettings(res.APIKey, fmt.Sprintf(`{%q:%d%s}`, s.field, v, s.companion)), 422, outOfRange(s.outsideRange))
		}
	}
}

func TestThePINTurnsOffOnlyAfterTheWait(t *testing.T) {
	n := newNode(t)
	res, _ := n.claimed()
	expect(t, n.patchSettings(res.APIKey, `{"pinWaitAfter":2}`), 422, outOfRange("Between 3 and 9 wrong PINs."))
	expect(t, n.patchSettings(res.APIKey, `{"pinOffAfter":3}`), 422, outOfRange("Between 6 and 20 wrong PINs."))
	expect(t, n.patchSettings(res.APIKey, `{"pinWaitAfter":6,"pinOffAfter":6}`), 422, outOfRange("Between 7 and 20 wrong PINs."))
	expect(t, n.patchSettings(res.APIKey, `{"pinWaitAfter":10}`), 422, outOfRange("Between 3 and 9 wrong PINs."))
	w := n.patchSettings(res.APIKey, `{"pinWaitAfter":8}`)
	var got struct {
		PINWaitAfter, PINOffAfter struct{ Value, Min, Max int64 }
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != http.StatusOK {
		t.Fatalf("updateIdentitySettings = %d %s", w.Code, w.Body)
	}
	if want := (struct{ Value, Min, Max int64 }{10, 9, 20}); got.PINOffAfter != want {
		t.Errorf("pinOffAfter = %+v beside a wait after 8, want %+v", got.PINOffAfter, want)
	}
	if want := (struct{ Value, Min, Max int64 }{8, 3, 9}); got.PINWaitAfter != want {
		t.Errorf("pinWaitAfter = %+v beside the PIN turning off after 10, want %+v", got.PINWaitAfter, want)
	}
	expect(t, n.patchSettings(res.APIKey, `{"pinOffAfter":8}`), 422, outOfRange("Between 9 and 20 wrong PINs."))
	if w := n.patchSettings(res.APIKey, `{"pinWaitAfter":null,"pinOffAfter":null}`); w.Code != http.StatusOK || strings.Contains(w.Body.String(), `"isDefault":false`) {
		t.Fatalf("resetting both = %d %s, want every setting at its default", w.Code, w.Body)
	}
}

func TestTheAddressIsAnOriginWithAName(t *testing.T) {
	n := newNode(t)
	res, _ := n.claimed()
	apiKey := map[string]string{"X-Api-Key": res.APIKey}
	put := func(body string) *httptest.ResponseRecorder {
		return n.do(call{method: http.MethodPut, target: "/api/v1/identity/address", host: "media.example.com:8484", body: body, headers: apiKey})
	}
	expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/identity/address", host: "media.example.com:8484", headers: apiKey}), 200,
		`{"address":"","suggestedAddress":"http://media.example.com:8484"}`+"\n")
	expect(t, put(`{"address":"HTTPS://Media.Example.com:443/"}`), 200,
		`{"address":"https://media.example.com","suggestedAddress":"http://media.example.com:8484"}`+"\n")
	expect(t, put(`{"address":"http://192.168.1.4:8484"}`), 422,
		problem(422, "address_ip", "The address must be a name, not an IP address"))
	expect(t, put(`{"address":"ftp://media.example.com"}`), 422,
		`{"status":422,"code":"invalid","title":"Some fields are invalid","errors":[{"location":"body.address","message":"expected http:// or https:// then a host name and an optional port"}]}`+"\n")
	expect(t, n.do(call{method: http.MethodGet, target: "/api/v1/identity/address", host: "media.example.com:8484", headers: apiKey}), 200,
		`{"address":"https://media.example.com","suggestedAddress":"http://media.example.com:8484"}`+"\n")
}
