package sim_test

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/passkey/passkeytest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/role"
)

// serving is the home Scenario with Serving on seedbox too.
func serving(t *testing.T) scenario.Scenario {
	t.Helper()
	s := home(t)
	for i, n := range s.Nodes {
		if n.Name == "seedbox" {
			s.Nodes[i].Roles = append(s.Nodes[i].Roles, role.Serving)
		}
	}
	return s
}

// entry is a Node's HTTP listener as a browser reaches it, by its
// .localhost name.
type entry struct {
	t      *testing.T
	w      *sim.World
	name   string
	origin string
}

func entryOf(t *testing.T, w *sim.World, name string) entry {
	t.Helper()
	port := w.Node(name).Ready().Listeners.HTTP.Port()
	return entry{t: t, w: w, name: name, origin: "http://" + name + ".localhost:" + strconv.Itoa(int(port))}
}

type reply struct {
	status  int
	body    string
	session string
}

// send calls path at the entry from the world's client at 127.0.0.1, as a
// page of origin does, with session's cookie unless it is empty.
func (e entry) send(method, path, body, session, origin string) *http.Response {
	e.t.Helper()
	var r io.Reader = http.NoBody
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(e.t.Context(), method, "http://"+e.w.Node(e.name).Ready().Listeners.HTTP.String()+path, r)
	if err != nil {
		e.t.Fatal(err)
	}
	req.Host = strings.TrimPrefix(e.origin, "http://")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		req.Header.Set("Origin", origin)
	}
	if session != "" {
		req.Header.Set("Cookie", identity.CookieName+"="+session)
	}
	res, err := e.w.Client().Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	return res
}

func (e entry) call(method, path, body, session string) reply {
	e.t.Helper()
	res := e.send(method, path, body, session, e.origin)
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		e.t.Fatal(err)
	}
	r := reply{status: res.StatusCode, body: string(data)}
	for _, c := range res.Cookies() {
		if c.Name == identity.CookieName {
			r.session = c.Value
		}
	}
	return r
}

func (e entry) signIn() reply {
	e.t.Helper()
	return e.call(http.MethodPost, "/api/v1/session", `{"email":"ada@example.test","pin":"2468"}`, "")
}

type system struct {
	DataDir string `json:"dataDir"`
}

func dataDir(t *testing.T, r reply) string {
	t.Helper()
	var s system
	if r.status != http.StatusOK || json.Unmarshal([]byte(r.body), &s) != nil {
		t.Fatalf("getSystem = %d %s", r.status, r.body)
	}
	return s.DataDir
}

// TestAServingWorkerServesTheControllersApp signs in at seedbox's address,
// which holds Serving: the session is home's, getSystem reports home, and
// the live stream carries home's events through seedbox. With home
// stopped, seedbox answers the API 503 controller_unreachable, serves its
// own app, and getServing says what it holds, never what anyone asked for.
func TestAServingWorkerServesTheControllersApp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, serving(t))
		synctest.Wait()
		box := entryOf(t, w, "seedbox")
		signedIn := box.signIn()
		if signedIn.status != http.StatusOK || signedIn.session == "" {
			t.Fatalf("signing in at seedbox's address = %d %s", signedIn.status, signedIn.body)
		}
		if got, want := dataDir(t, box.call(http.MethodGet, "/api/v1/system", "", signedIn.session)), w.Node("home").DataDir().Path(); got != want {
			t.Fatalf("getSystem at seedbox's address reports %s, want home's %s", got, want)
		}

		stream := box.send(http.MethodGet, "/api/v1/events", "", signedIn.session, "")
		defer func() { _ = stream.Body.Close() }()
		events := bufio.NewReader(stream.Body)
		if got := nextEvent(t, events); got != "live.resync" {
			t.Fatalf("the live stream at seedbox's address opened with %s, want live.resync", got)
		}
		if again := entryOf(t, w, "home").signIn(); again.status != http.StatusOK {
			t.Fatalf("signing in at home = %d %s", again.status, again.body)
		}
		if got := nextEvent(t, events); got != "identity.sessions-changed" {
			t.Fatalf("after a sign-in at home, the live stream at seedbox's address read %s", got)
		}

		w.Node("home").Stop()
		stopped := time.Now()
		synctest.Wait()
		time.Sleep(time.Minute)
		if r := box.call(http.MethodGet, "/api/v1/system", "", signedIn.session); r.status != http.StatusServiceUnavailable ||
			r.body != `{"status":503,"code":"controller_unreachable","title":"The Controller can't be reached"}`+"\n" {
			t.Fatalf("getSystem at seedbox's address with home stopped = %d %s", r.status, r.body)
		}
		if r := box.call(http.MethodGet, "/movies", "", ""); r.status != http.StatusNotFound || !strings.HasPrefix(r.body, "the web app isn't embedded") {
			t.Fatalf("a navigation at seedbox's address with home stopped = %d %s, want seedbox's own app", r.status, r.body)
		}
		r := box.call(http.MethodGet, "/api/v1/serving", "", "")
		want := `{"name":"seedbox","fingerprint":"` + key(t, w, "seedbox").Fingerprint() + `","controller":"home","since":"` +
			stopped.UTC().Format(time.RFC3339Nano) + `","holding":["Searching","Downloading"],"seeding":0}` + "\n"
		if r.status != http.StatusOK || r.body != want {
			t.Fatalf("getServing with home stopped = %d %s, want 200 %s", r.status, r.body, want)
		}
	})
}

func nextEvent(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	event := ""
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("reading the live stream: %v", err)
		}
		if name, ok := strings.CutPrefix(line, "event: "); ok {
			event = strings.TrimSpace(name)
		}
		if line == "\n" && event != "" {
			return event
		}
	}
}

// TestAMismatchedServingWorkerStillServes restarts seedbox on another
// build, as during an update: its Serve streams still reach home.
func TestAMismatchedServingWorkerStillServes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, serving(t))
		synctest.Wait()
		seedbox := w.Node("seedbox")
		seedbox.Stop()
		seedbox.Start(func(e *node.Env) { e.Version = "0.0.0-older" })
		synctest.Wait()
		box := entryOf(t, w, "seedbox")
		signedIn := box.signIn()
		if signedIn.status != http.StatusOK {
			t.Fatalf("signing in at a mismatched seedbox's address = %d %s", signedIn.status, signedIn.body)
		}
		if got, want := dataDir(t, box.call(http.MethodGet, "/api/v1/system", "", signedIn.session)), w.Node("home").DataDir().Path(); got != want {
			t.Fatalf("getSystem at a mismatched seedbox's address reports %s, want home's %s", got, want)
		}
	})
}

// TestUntickingServingHandsTheAddressBack takes Serving off seedbox: its
// address answers as a Worker's again.
func TestUntickingServingHandsTheAddressBack(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, serving(t))
		synctest.Wait()
		box := entryOf(t, w, "seedbox")
		if r := box.call(http.MethodGet, "/api/v1/status", "", ""); !strings.Contains(r.body, `"claim":"controller"`) {
			t.Fatalf("getStatus at a Serving seedbox's address = %d %s, want home's", r.status, r.body)
		}
		a := w.API()
		a.SetRoles("home", a.Node("seedbox").NodeID, "searching", "downloading")
		synctest.Wait()
		if r := box.call(http.MethodGet, "/api/v1/status", "", ""); !strings.Contains(r.body, `"claim":"worker"`) {
			t.Fatalf("getStatus at seedbox's address once it stopped serving = %d %s, want seedbox's own", r.status, r.body)
		}
	})
}

// TestAPasskeyWorksAtTheAddressItWasMadeAt is the first-PR fact "passkey
// origin". The Worker passes Host and Origin through, so a ceremony at
// seedbox's address runs for seedbox's origin: a passkey added there signs
// in there. A passkey added at home's address is bound to home's RP ID, so
// the browser has none to offer at seedbox's, while the PIN signs in at
// both. The origin check reads the Host the Worker received.
func TestAPasskeyWorksAtTheAddressItWasMadeAt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, serving(t))
		synctest.Wait()
		box, home := entryOf(t, w, "seedbox"), entryOf(t, w, "home")
		atBox, atHome := box.signIn(), home.signIn()
		if atBox.status != http.StatusOK || atHome.status != http.StatusOK {
			t.Fatalf("signing in by PIN: at seedbox's address %d, at home's %d", atBox.status, atHome.status)
		}

		boxKey, homeKey := &passkeytest.Authenticator{}, &passkeytest.Authenticator{}
		box.addPasskey(atBox.session, boxKey)
		home.addPasskey(atHome.session, homeKey)
		if r := box.passkeySignIn(boxKey); r.status != http.StatusOK || r.session == "" {
			t.Fatalf("the passkey added at seedbox's address, signing in there = %d %s", r.status, r.body)
		}
		if r := box.call(http.MethodGet, "/api/v1/session", "", box.passkeySignIn(boxKey).session); r.status != http.StatusOK {
			t.Fatalf("the passkey's session at seedbox's address = %d %s", r.status, r.body)
		}
		s := box.startAssertion()
		if _, err := homeKey.Assert(box.origin, s.Options); err == nil {
			t.Fatal("the passkey added at home's address was offered at seedbox's")
		}

		cross := box.send(http.MethodPost, "/api/v1/session", `{"email":"ada@example.test","pin":"2468"}`, "", home.origin)
		_ = cross.Body.Close()
		if cross.StatusCode != http.StatusForbidden {
			t.Fatalf("a sign-in at seedbox's address from home's page = %d, want 403", cross.StatusCode)
		}
	})
}

type ceremony struct {
	Ceremony string          `json:"ceremony"`
	Options  json.RawMessage `json:"options"`
}

func (e entry) ceremony(path, session string) ceremony {
	e.t.Helper()
	r := e.call(http.MethodPost, path, "", session)
	var c ceremony
	if r.status != http.StatusOK || json.Unmarshal([]byte(r.body), &c) != nil {
		e.t.Fatalf("POST %s at %s = %d %s", path, e.origin, r.status, r.body)
	}
	return c
}

func (e entry) addPasskey(session string, key *passkeytest.Authenticator) {
	e.t.Helper()
	c := e.ceremony("/api/v1/passkeys/registration", session)
	response, err := key.Register(e.origin, c.Options)
	if err != nil {
		e.t.Fatal(err)
	}
	if r := e.call(http.MethodPost, "/api/v1/passkeys/registration/"+c.Ceremony, string(response), session); r.status != http.StatusCreated ||
		!strings.Contains(r.body, `"host":"`+e.name+`.localhost"`) {
		e.t.Fatalf("adding a passkey at %s = %d %s", e.origin, r.status, r.body)
	}
}

func (e entry) startAssertion() ceremony {
	e.t.Helper()
	return e.ceremony("/api/v1/passkeys/assertion", "")
}

func (e entry) passkeySignIn(key *passkeytest.Authenticator) reply {
	e.t.Helper()
	c := e.startAssertion()
	response, err := key.Assert(e.origin, c.Options)
	if err != nil {
		e.t.Fatalf("asserting at %s: %v", e.origin, err)
	}
	return e.call(http.MethodPost, "/api/v1/passkeys/assertion/"+c.Ceremony, string(response), "")
}
