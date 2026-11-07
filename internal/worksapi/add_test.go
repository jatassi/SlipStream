package worksapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// defaults is the fixture type's three add defaults.
type defaults struct {
	profile, root string
	monitor       bool
}

func (s *server) defaults(t *testing.T) defaults {
	t.Helper()
	var d defaults
	profiles, err := s.goals.ListProfiles(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range profiles {
		if p.Default && p.ContentType == contenttypetest.ID {
			d.profile = p.Name
		}
	}
	err = s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		all, err := roots.All(ctx, r)
		for _, root := range all {
			if root.Default {
				d.root = root.Path
			}
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	code, body := s.call(t, http.MethodGet, "/api/v1/content-types/fixture/settings", "")
	var settings struct {
		Settings []struct {
			Key   string
			Value json.RawMessage
		}
	}
	if err := json.Unmarshal([]byte(body), &settings); code != http.StatusOK || err != nil {
		t.Fatalf("getContentTypeSettings = %d %s, %v", code, body, err)
	}
	for _, f := range settings.Settings {
		if f.Key == "fixture.add.monitor" {
			d.monitor = string(f.Value) == "true"
		}
	}
	return d
}

type added struct {
	Created bool `json:"created"`
	Work    struct {
		ID    string `json:"id"`
		Items []struct {
			Monitored bool `json:"monitored"`
		} `json:"items"`
		Profile struct {
			Name string `json:"name"`
		} `json:"profile"`
	} `json:"work"`
}

func (s *server) add(t *testing.T, body string) added {
	t.Helper()
	code, reply := s.call(t, http.MethodPost, "/api/v1/works", body)
	var a added
	if err := json.Unmarshal([]byte(reply), &a); code != http.StatusOK || err != nil {
		t.Fatalf("addWork %s = %d %s, %v", body, code, reply, err)
	}
	return a
}

// remembering is an add of key that remembers, with each choice given
// when it isn't empty.
func remembering(key, profile string, root roots.Root, monitor string) string {
	body := `{"contentType":"fixture","key":"` + key + `","remember":true`
	if profile != "" {
		body += `,"profileId":"` + profile + `"`
	}
	if !root.ID.IsZero() {
		body += `,"rootId":"` + root.ID.String() + `"`
	}
	if monitor != "" {
		body += `,"monitor":` + monitor
	}
	return body + "}"
}

type world struct {
	s              *server
	small, fixture targets.Profile
	first, alt     roots.Root
}

// newWorld is a fixture type with its default Profile and a Small one, and
// two roots, first its default. It holds records w1 to w5.
func newWorld(t *testing.T) world {
	t.Helper()
	s := serve(t)
	profiles, err := s.goals.ListProfiles(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	small, err := s.goals.CreateProfile(t.Context(), identity.Admin{}, targets.ProfileSpec{Name: "Small", ContentType: contenttypetest.ID})
	if err != nil {
		t.Fatal(err)
	}
	node := role.NodeID{UUID: storage.NewUUID()}
	first := rootstest.Root(t, s.lib, node, contenttypetest.ID, t.TempDir())
	alt := rootstest.Root(t, s.lib, node, contenttypetest.ID, t.TempDir())
	for _, key := range []works.Key{"w1", "w2", "w3", "w4", "w5"} {
		s.fx.Put(key, record())
	}
	s.targetsChanged.take()
	s.rootsChanged.take()
	return world{s: s, small: small, fixture: profiles[0], first: first, alt: alt}
}

// moved fails unless the adds since the last call published one
// targets.changed when profile and roots.changed naming exactly rootIDs.
func (w world) moved(t *testing.T, step string, profile bool, rootIDs ...roots.ID) {
	t.Helper()
	sentTargets, sentRoots := w.s.targetsChanged.take(), w.s.rootsChanged.take()
	if want := map[bool]int{true: 1}[profile]; len(sentTargets) != want {
		t.Errorf("%s published targets.changed %d times, want %d", step, len(sentTargets), want)
	}
	var named []string
	for _, c := range sentRoots {
		named = append(named, c.RootIDs...)
	}
	var want []string
	for _, id := range rootIDs {
		want = append(want, id.String())
	}
	slices.Sort(named)
	slices.Sort(want)
	if !slices.Equal(named, want) {
		t.Errorf("%s published roots.changed naming %v, want %v", step, named, want)
	}
}

func TestRememberMovesExactlyTheChangedDefaults(t *testing.T) {
	w := newWorld(t)
	s := w.s
	if got, want := s.defaults(t), (defaults{profile: "fixture", root: w.first.Path, monitor: true}); got != want {
		t.Fatalf("defaults at start = %+v, want %+v", got, want)
	}

	a := s.add(t, remembering("w1", w.small.ID.String(), w.alt, "false"))
	if got, want := s.defaults(t), (defaults{profile: "Small", root: w.alt.Path, monitor: false}); !a.Created || got != want {
		t.Fatalf("after a remembered add with every choice changed: created %t, defaults %+v, want %+v", a.Created, got, want)
	}
	w.moved(t, "changing every default", true, w.alt.ID, w.first.ID)

	s.add(t, remembering("w2", w.small.ID.String(), w.alt, "false"))
	w.moved(t, "choosing every default as it is", false)

	a = s.add(t, remembering("w3", w.small.ID.String(), w.first, ""))
	if got, want := s.defaults(t), (defaults{profile: "Small", root: w.first.Path, monitor: false}); got != want || a.Work.Items[0].Monitored {
		t.Fatalf("after a remembered add changing only the root: defaults %+v, want %+v; monitored %t, want the default off", got, want, a.Work.Items[0].Monitored)
	}
	w.moved(t, "changing only the root", false, w.first.ID, w.alt.ID)

	a = s.add(t, remembering("w1", w.fixture.ID.String(), w.alt, "true"))
	if got, want := s.defaults(t), (defaults{profile: "Small", root: w.first.Path, monitor: false}); a.Created || got != want || a.Work.Profile.Name != "Small" {
		t.Fatalf("a remembered add of a Work already there: created %t, wants from %s, defaults %+v; want created false, Small, %+v", a.Created, a.Work.Profile.Name, got, want)
	}
	w.moved(t, "adding a Work already there", false)
}

func TestAnAddThatDoesntRememberMovesNoDefault(t *testing.T) {
	w := newWorld(t)
	s := w.s
	a := s.add(t, `{"contentType":"fixture","key":"w1","profileId":"`+w.small.ID.String()+`","rootId":"`+w.alt.ID.String()+`","monitor":false}`)
	if got, want := s.defaults(t), (defaults{profile: "fixture", root: w.first.Path, monitor: true}); !a.Created || a.Work.Profile.Name != "Small" || a.Work.Items[0].Monitored || got != want {
		t.Fatalf("an add without remember: created %t, wants from %s, monitored %t, defaults %+v; want a Small unmonitored Work and %+v", a.Created, a.Work.Profile.Name, a.Work.Items[0].Monitored, got, want)
	}
	w.moved(t, "an add without remember", false)
}

func TestAMonitorLeftOutTakesTheTypesAddMonitor(t *testing.T) {
	w := newWorld(t)
	s := w.s
	if a := s.add(t, `{"contentType":"fixture","key":"w1"}`); !a.Work.Items[0].Monitored {
		t.Fatal("an add with no monitor while fixture.add.monitor is on left its Item unmonitored")
	}
	expect(t, s, http.MethodPatch, "/api/v1/content-types/fixture/settings", `{"values":{"fixture.add.monitor":false}}`, 200,
		`{"settings":[`+addMonitorField(false, false)+`,{"key":"fixture.depth","label":"Boxes deep","source":"database","applies":"live","isDefault":true,"pendingRestart":false,"kind":"count","value":3,"running":3,"default":3,"min":1,"max":9,"unit":{"one":"box","many":"boxes"}}]}`)
	if a := s.add(t, `{"contentType":"fixture","key":"w2"}`); a.Work.Items[0].Monitored {
		t.Fatal("an add with no monitor while fixture.add.monitor is off monitored its Item")
	}
	if a := s.add(t, `{"contentType":"fixture","key":"w3","monitor":true}`); !a.Work.Items[0].Monitored {
		t.Fatal("an add with monitor true while fixture.add.monitor is off left its Item unmonitored")
	}
}

func TestARacingAddAnswersTheWorkTheOtherMade(t *testing.T) {
	w := newWorld(t)
	s := w.s
	letGo := s.fx.Hold(2)
	choices := map[string]string{w.small.ID.String(): "Small", w.fixture.ID.String(): "fixture"}
	type answer struct {
		profile string
		got     added
	}
	answers := make(chan answer, 2)
	for _, profile := range []string{w.small.ID.String(), w.fixture.ID.String()} {
		go func() {
			code, reply := s.call(t, http.MethodPost, "/api/v1/works", remembering("w1", profile, roots.Root{}, ""))
			var a added
			if err := json.Unmarshal([]byte(reply), &a); code != http.StatusOK || err != nil {
				t.Errorf("racing addWork = %d %s, %v", code, reply, err)
			}
			answers <- answer{choices[profile], a}
		}()
	}
	letGo()
	one, two := <-answers, <-answers
	if one.got.Created == two.got.Created || one.got.Work.ID != two.got.Work.ID {
		t.Fatalf("racing adds answered %+v and %+v, want one Work, created once", one.got, two.got)
	}
	made := one
	if two.got.Created {
		made = two
	}
	if got := s.defaults(t).profile; got != made.profile || one.got.Work.Profile.Name != made.profile || two.got.Work.Profile.Name != made.profile {
		t.Fatalf("after racing adds the default is %s and they want from %s and %s; want the Profile of the add that made the Work, %s",
			got, one.got.Work.Profile.Name, two.got.Work.Profile.Name, made.profile)
	}
	w.moved(t, "racing adds", made.profile == "Small")
}
