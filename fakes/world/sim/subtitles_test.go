package sim_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/role"
)

const (
	amber  = "Amber Signal"
	brick  = "Brick Lane Echo"
	cinder = "Cinder Bloom"
	lamps  = "The Lamplighters"
)

// english is a Wanted Language that wants English subtitles whatever the
// audio, hearing impaired as hi says.
func english(hi string) string {
	return `{"language":"en","forced":false,"hearingImpaired":"` + hi + `","audio":"always"}`
}

type subWorld struct {
	t *testing.T
	w *sim.World
	// at is the Controller, which answers the API.
	at string
}

// launchSubtitles launches the subtitles Scenario after edit, with SubDL
// and SubSource keyed.
func launchSubtitles(t *testing.T, edit func(*scenario.Scenario)) subWorld {
	t.Helper()
	s, err := scenario.Load("subtitles")
	if err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		edit(&s)
	}
	sw := subWorld{t: t, w: sim.Launch(t, s, sim.WithWalk()), at: s.Nodes[0].Name}
	sw.key("subdl")
	sw.key("subsource")
	return sw
}

func (s subWorld) api() *worldtest.API { return s.w.API() }

func (s subWorld) call(method, route, body string) worldtest.Reply {
	return s.api().Admin(s.at, method, route, body)
}

func (s subWorld) key(provider string) {
	s.t.Helper()
	if r := s.call(http.MethodPut, "/api/v1/subtitles/providers/"+provider+"/key", `{"key":"sim-`+provider+`"}`); r.Status != http.StatusOK {
		s.t.Fatalf("putting a %s key = %d %s, want 200", provider, r.Status, r.Body)
	}
}

// want puts the Wanted Languages on the Profile named profile.
func (s subWorld) want(profile string, languages ...string) {
	s.t.Helper()
	listed := simtest.Decode[struct {
		Profiles []map[string]json.RawMessage `json:"profiles"`
	}](s.t, "listProfiles", s.call(http.MethodGet, "/api/v1/profiles", ""))
	for _, p := range listed.Profiles {
		var name, id string
		_ = json.Unmarshal(p["name"], &name)
		_ = json.Unmarshal(p["id"], &id)
		if name != profile {
			continue
		}
		body := `{"name":` + string(p["name"]) + `,"contentType":` + string(p["contentType"]) + `,"targets":` + string(p["targets"]) +
			`,"editions":` + string(p["editions"]) + `,"languages":[` + strings.Join(languages, ",") + `]}`
		if r := s.call(http.MethodPut, "/api/v1/profiles/"+id+"?confirm=0&confirmSubtitles=0", body); r.Status != http.StatusOK {
			s.t.Fatalf("putting %s's languages = %d %s, want 200", profile, r.Status, r.Body)
		}
		return
	}
	s.t.Fatalf("no Profile is named %s", profile)
}

type subFile struct {
	FileID     string `json:"fileId"`
	Path       string `json:"path"`
	Companions []struct {
		Name string `json:"name"`
	} `json:"companions"`
	Wants []struct {
		Want  string `json:"want"`
		State struct {
			Kind    string  `json:"kind"`
			Missing *string `json:"missing"`
			Sidecar *struct {
				Name     string  `json:"name"`
				Source   string  `json:"source"`
				Provider *string `json:"provider"`
				Fit      *struct {
					Match int `json:"match"`
				} `json:"fit"`
			} `json:"sidecar"`
		} `json:"state"`
	} `json:"wants"`
}

// english is the file's English want's state as "kind" and either why it
// is missing or its Sidecar's name, source and provider.
func (f subFile) english() string {
	for _, w := range f.Wants {
		if w.Want != "en" {
			continue
		}
		st := w.State
		switch {
		case st.Missing != nil:
			return st.Kind + " " + *st.Missing
		case st.Sidecar != nil && st.Sidecar.Provider != nil:
			return st.Kind + " " + st.Sidecar.Name + " " + st.Sidecar.Source + " " + *st.Sidecar.Provider
		case st.Sidecar != nil:
			return st.Kind + " " + st.Sidecar.Name + " " + st.Sidecar.Source
		}
		return st.Kind
	}
	return "unwanted"
}

// work is the Work labeled label, once the library scan has added it.
func (s subWorld) work(label string) string {
	s.t.Helper()
	var id string
	simtest.Eventually(s.t, "the library lists "+label, func() bool {
		lib := simtest.Decode[struct {
			Rows []struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			} `json:"rows"`
		}](s.t, "getLibrary", s.call(http.MethodGet, "/api/v1/library", ""))
		for _, r := range lib.Rows {
			if r.Label == label {
				id = r.ID
			}
		}
		return id != ""
	})
	return id
}

// files are the Work's files with their subtitles, in path order.
func (s subWorld) files(label string) []subFile {
	s.t.Helper()
	return simtest.Decode[struct {
		Files []subFile `json:"files"`
	}](s.t, "getWorkSubtitles", s.call(http.MethodGet, "/api/v1/works/"+s.work(label)+"/subtitles", "")).Files
}

func (s subWorld) file(label string) subFile {
	s.t.Helper()
	return s.files(label)[0]
}

// englishOf is each of the Work's files' English want, in path order.
func (s subWorld) englishOf(label string) []string {
	s.t.Helper()
	var out []string
	for _, f := range s.files(label) {
		out = append(out, f.english())
	}
	return out
}

// await waits in fake time until every file of the Work reads want.
func (s subWorld) await(label, want string) {
	s.t.Helper()
	var got []string
	for range simtest.Settled / time.Second {
		if got = s.englishOf(label); !slices.ContainsFunc(got, func(g string) bool { return g != want }) {
			return
		}
		time.Sleep(time.Second)
	}
	s.t.Fatalf("%s still reads %q, want %s", label, got, want)
}

// scanned waits until the Work's files' audio is scanned.
func (s subWorld) scanned(label string) {
	s.t.Helper()
	simtest.Eventually(s.t, label+"'s audio is scanned", func() bool {
		return !slices.ContainsFunc(s.englishOf(label), func(got string) bool { return got == "missing map_due" })
	})
}

func (s subWorld) searchNow(label string) {
	s.t.Helper()
	for _, f := range s.files(label) {
		if r := s.call(http.MethodPost, "/api/v1/files/"+f.FileID+"/subtitles/en/search", ""); r.Status != http.StatusAccepted {
			s.t.Fatalf("searching %s now = %d %s, want 202", f.Path, r.Status, r.Body)
		}
	}
}

// sidecars are the subtitle files on disk beside the Work's videos, by
// name.
func (s subWorld) sidecars(folder string) []string {
	s.t.Helper()
	var out []string
	err := filepath.WalkDir(filepath.Join(s.w.Manifest().MediaDir, folder), func(p string, d os.DirEntry, err error) error {
		if err == nil && filepath.Ext(p) == ".srt" {
			out = append(out, d.Name())
		}
		return err
	})
	if err != nil {
		s.t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

type candidates struct {
	State      string `json:"state"`
	Candidates []struct {
		Key      string `json:"key"`
		Provider string `json:"provider"`
		Tried    *struct {
			Outcome string `json:"outcome"`
		} `json:"tried"`
	} `json:"candidates"`
	Outcomes []struct {
		Provider string `json:"provider"`
		Kind     string `json:"kind"`
	} `json:"outcomes"`
}

// manualSearch runs a Manual subtitle search on the file's English want
// and answers it once done.
func (s subWorld) manualSearch(f subFile) candidates {
	s.t.Helper()
	started := simtest.Decode[struct {
		SearchID string `json:"searchId"`
	}](s.t, "startManualSearch", accepted(s.call(http.MethodPost, "/api/v1/files/"+f.FileID+"/subtitles/en/candidates", "")))
	var c candidates
	simtest.Eventually(s.t, "the Manual subtitle search is done", func() bool {
		c = simtest.Decode[candidates](s.t, "readManualSearch", s.call(http.MethodGet, "/api/v1/files/"+f.FileID+"/subtitles/en/candidates/"+started.SearchID, ""))
		return c.State == "done"
	})
	return c
}

// accepted reads a 202 as the 200 simtest.Decode wants.
func accepted(r worldtest.Reply) worldtest.Reply {
	if r.Status == http.StatusAccepted {
		r.Status = http.StatusOK
	}
	return r
}

func (s subWorld) provider(id string) (state, until string) {
	s.t.Helper()
	list := simtest.Decode[struct {
		Providers []struct {
			ID    string `json:"id"`
			State struct {
				Kind  string  `json:"kind"`
				Until *string `json:"until"`
			} `json:"state"`
		} `json:"providers"`
	}](s.t, "listProviders", s.call(http.MethodGet, "/api/v1/subtitles/providers", ""))
	for _, p := range list.Providers {
		if p.ID == id {
			if p.State.Until != nil {
				return p.State.Kind, *p.State.Until
			}
			return p.State.Kind, ""
		}
	}
	s.t.Fatalf("no provider is %s", id)
	return "", ""
}

// TestSubtitleSearchKeepsTheThirdCandidateWhenTheFirstTwoFail ranks two
// SubDL subtitles that follow other speech above SubSource's that
// follows the film's: a search fetches down the list, tries each SubDL
// one, and keeps SubSource's, beside the video and with its timing match.
func TestSubtitleSearchKeepsTheThirdCandidateWhenTheFirstTwoFail(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := launchSubtitles(t, func(s *scenario.Scenario) {
			s.SubtitleProviders.Catalog = append(s.SubtitleProviders.Catalog, scenario.CatalogSubtitle{
				ID: "brick-other-speech-2", Provider: "subdl", Work: brick, Language: "en", Release: "Brick.Lane.Echo.2019.1080p.WEB-DL.H264-OTHER",
			})
		})
		s.want("Movies", english("allow"))
		s.scanned(brick)
		s.searchNow(brick)
		s.await(brick, "met en.srt provider subsource")

		if got := s.sidecars("movies/Brick Lane Echo (2019) {tmdb-9000202}"); !slices.Equal(got, []string{"Brick.Lane.Echo.2019.1080p.WEB-DL.H264-FKW.en.srt"}) {
			t.Fatalf("beside Brick Lane Echo's video sit %v, want its .en.srt alone", got)
		}
		var tried []string
		for _, c := range s.manualSearch(s.file(brick)).Candidates {
			if c.Tried != nil {
				tried = append(tried, c.Key+" "+c.Tried.Outcome)
			}
		}
		slices.Sort(tried)
		if want := []string{"subdl:brick-other-speech failed_timing", "subdl:brick-other-speech-2 failed_timing", "subsource:4851815 passed"}; !slices.Equal(tried, want) {
			t.Fatalf("the candidates tried before read %q, want both SubDL ones failed on timing and SubSource's passed", tried)
		}
	})
}

// TestGestdownRefreshingTheShowAnswers423ThenItsSeasonPackLands has
// Gestdown answer an episode search with 423 while it refreshes the show:
// the search is tried again on Gestdown's refresh schedule, not given up,
// and one season pack then lands each episode's subtitle beside it.
func TestGestdownRefreshingTheShowAnswers423ThenItsSeasonPackLands(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := launchSubtitles(t, nil)
		s.want("TV", english("allow"))
		s.scanned(lamps)
		s.w.Cue("gestdown", "gestdown-refreshing", map[string]int{"calls": 1})
		asked := time.Now()
		s.searchNow(lamps)
		s.await(lamps, "met en.srt provider gestdown")

		if waited := time.Since(asked); waited < 30*time.Second {
			t.Fatalf("the episodes met English %s after the search, want no sooner than Gestdown's 30 s refresh retry", waited)
		}
		if got, want := s.sidecars("tv"), []string{"The Lamplighters - S01E01.en.srt", "The Lamplighters - S01E02.en.srt", "The Lamplighters - S01E03.en.srt"}; !slices.Equal(got, want) {
			t.Fatalf("the season's folder holds %v, want %v", got, want)
		}
	})
}

// TestAKeyThatRunsOutPausesSubDLUntilMidnightAcrossARestart spends
// SubDL's daily downloads with SubSource off: a search of Cinder Bloom,
// whose one subtitle is SubDL's, leaves it missing because SubDL is
// paused, out until the next UTC midnight, which a restart of the
// Controller keeps. TestAKeyThatRunsOutPausesSubDLUntilItsResetAndSurvivesARestart
// in internal/subtitles fetches it after the reset.
func TestAKeyThatRunsOutPausesSubDLUntilMidnightAcrossARestart(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := launchSubtitles(t, nil)
		if r := s.call(http.MethodPatch, "/api/v1/subtitles/providers/subsource", `{"on":false}`); r.Status != http.StatusOK {
			t.Fatalf("turning SubSource off = %d %s, want 200", r.Status, r.Body)
		}
		s.want("Movies", english("allow"))
		s.scanned(cinder)
		s.w.Cue("subdl", "subdl-daily-limit", map[string]string{})
		s.searchNow(cinder)
		now := time.Now().UTC()
		midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
		simtest.Eventually(t, "SubDL reads out", func() bool { state, _ := s.provider("subdl"); return state == "out" })
		if _, until := s.provider("subdl"); until != midnight.Format(time.RFC3339) {
			t.Fatalf("SubDL is out until %s, want the next UTC midnight %s", until, midnight.Format(time.RFC3339))
		}
		s.await(cinder, "missing paused")

		s.w.Node(s.at).Stop()
		s.w.Node(s.at).Start()
		if state, until := s.provider("subdl"); state != "out" || until != midnight.Format(time.RFC3339) {
			t.Fatalf("after a restart SubDL reads %s until %s, want out until %s", state, until, midnight.Format(time.RFC3339))
		}
		s.await(cinder, "missing paused")
	})
}

// TestAnUpgradeCarriesItsSidecar imports Amber Signal's 1080p file by
// hand over the 720p one that has SubSource's subtitle. When the new
// file speaks as the old one did, the subtitle follows it to its new name
// and passes its timing again; when it speaks otherwise, the carried
// subtitle fails, leaves the disk, and the want is missing again.
func TestAnUpgradeCarriesItsSidecar(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		seed    uint64
		after   string
		carried bool
	}{
		{"the same speech", 11, "met en.srt provider subsource", true},
		{"other speech", 12, "missing never_searched", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				s := launchSubtitles(t, func(s *scenario.Scenario) {
					shelf := &s.Library.Folders[len(s.Library.Folders)-1].Files[0]
					shelf.Audio[0].Speech.Seed = tc.seed
				})
				s.want("Movies", english("allow"))
				s.scanned(amber)
				s.searchNow(amber)
				s.await(amber, "met en.srt provider subsource")

				s.importByHand("shelf/Amber.Signal.2020.1080p.WEB-DL.H264-FKW.mkv")
				simtest.Eventually(t, "the 1080p file replaced the 720p one", func() bool {
					return strings.HasSuffix(s.file(amber).Path, "1080p.mkv")
				})
				s.scanned(amber)
				s.await(amber, tc.after)
				var want []string
				if tc.carried {
					want = []string{strings.TrimSuffix(path.Base(s.file(amber).Path), ".mkv") + ".en.srt"}
				}
				simtest.Eventually(t, fmt.Sprintf("beside Amber Signal's 1080p video sit %q", want), func() bool {
					return slices.Equal(s.sidecars("movies/Amber Signal (2020) {tmdb-9000201}"), want)
				})
			})
		})
	}
}

// importByHand copies the library file at rel into a folder of the same
// name in the Controller's download folder, imports that folder by hand
// and confirms its Import.
func (s subWorld) importByHand(rel string) {
	s.t.Helper()
	downloads := s.w.Node(s.at).DataDir().Resolve("downloads")
	data, err := os.ReadFile(filepath.Join(s.w.Manifest().MediaDir, filepath.FromSlash(rel)))
	if err != nil {
		s.t.Fatal(err)
	}
	at := filepath.Join(downloads, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(at), 0o700); err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(at, data, 0o600); err != nil {
		s.t.Fatal(err)
	}
	body := `{"nodeId":"` + s.api().Node(s.at).NodeID + `","folder":` + strconv.Quote(filepath.Dir(at)) + `}`
	opened := simtest.Decode[struct {
		Imports []struct {
			ID string `json:"id"`
		} `json:"imports"`
	}](s.t, "importFolder", created(s.call(http.MethodPost, "/api/v1/imports", body)))
	for _, im := range opened.Imports {
		simtest.Eventually(s.t, "the Import waits to be confirmed or places its file", func() bool {
			r := s.call(http.MethodGet, "/api/v1/imports/"+im.ID, "")
			return strings.Contains(r.Body, `"outcome":"waiting"`) || strings.Contains(r.Body, `"outcome":"placed"`)
		})
		if r := s.call(http.MethodPost, "/api/v1/imports/confirm", `{"importIds":["`+im.ID+`"]}`); r.Status != http.StatusOK {
			s.t.Fatalf("confirmImports = %d %s, want 200", r.Status, r.Body)
		}
	}
}

func created(r worldtest.Reply) worldtest.Reply {
	if r.Status == http.StatusCreated {
		r.Status = http.StatusOK
	}
	return r
}

// TestASurveyAdoptsTheSubtitlesBesideAFile puts an .en.srt and an
// .en.hi.srt beside Brick Lane Echo and scans its root: English that
// prefers hearing impaired adopts the .en.hi.srt as found, without a
// search, and both files stay on disk.
func TestASurveyAdoptsTheSubtitlesBesideAFile(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := launchSubtitles(t, nil)
		folder := "movies/Brick Lane Echo (2019) {tmdb-9000202}"
		for _, name := range []string{"Brick.Lane.Echo.2019.1080p.WEB-DL.H264-FKW.en.srt", "Brick.Lane.Echo.2019.1080p.WEB-DL.H264-FKW.en.hi.srt"} {
			if err := os.WriteFile(filepath.Join(s.w.Manifest().MediaDir, filepath.FromSlash(folder), name), []byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		root := simtest.RootAt(t, s.api(), filepath.Join(s.w.Manifest().MediaDir, "movies"))
		if r := s.call(http.MethodPost, "/api/v1/roots/"+root.ID+"/scan", ""); r.Status != http.StatusAccepted {
			t.Fatalf("scanning the Movies root = %d %s, want 202", r.Status, r.Body)
		}
		simtest.Eventually(t, "the scan lists both companions", func() bool { return len(s.file(brick).Companions) == 2 })
		s.want("Movies", english("prefer"))
		s.await(brick, "met en.hi.srt found")
		if got := s.sidecars(folder); len(got) != 2 {
			t.Fatalf("beside Brick Lane Echo sit %v, want both subtitles kept", got)
		}
	})
}

// TestThreeNodesMapOnTheStorageNodeAndSearchOnTheSubtitlesHolder runs
// Storage, Subtitles and the Controller on three Nodes: the Storage Node
// scans the audio and keeps the Speech Map, the Subtitles holder fetches,
// and the Sidecar lands beside the video on the Storage Node's disk.
func TestThreeNodesMapOnTheStorageNodeAndSearchOnTheSubtitlesHolder(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := launchSubtitles(t, func(s *scenario.Scenario) {
			s.Nodes = []scenario.Node{
				{Name: "home", Setup: scenario.SetupController, Roles: []role.Name{role.Metadata, role.Notifying, role.Serving}},
				{Name: "nas", Setup: scenario.SetupJoin, Roles: []role.Name{role.Storage, role.Converting}},
				{Name: "subs", Setup: scenario.SetupJoin, Roles: []role.Name{role.Subtitles}},
			}
			for i := range s.Library.Roots {
				s.Library.Roots[i].Node = "nas"
			}
		})
		s.want("Movies", english("allow"))
		s.scanned(brick)
		s.searchNow(brick)
		s.await(brick, "met en.srt provider subsource")

		maps, err := os.ReadDir(s.w.Node("nas").DataDir().SpeechMapsDir())
		if err != nil || len(maps) == 0 {
			t.Fatalf("nas's speechmaps folder lists %d files, %v; want the films' maps", len(maps), err)
		}
		if _, err := os.Stat(s.w.Node("home").DataDir().SpeechMapsDir()); err == nil {
			if entries, _ := os.ReadDir(s.w.Node("home").DataDir().SpeechMapsDir()); len(entries) > 0 {
				t.Fatalf("the Controller keeps %d Speech Maps, want none: maps stay on the Node holding the file", len(entries))
			}
		}
		if got := s.sidecars("movies/Brick Lane Echo (2019) {tmdb-9000202}"); len(got) != 1 {
			t.Fatalf("beside Brick Lane Echo sit %v, want SubSource's subtitle", got)
		}
	})
}

// TestAStorageNodeThatDropsOffHoldsItsSearchUntilItIsBack stalls the path
// between the Controller and the Storage Node past the link's keepalive:
// a Search of Brick Lane Echo then waits, its Node asked at most once a
// poll rather than in a loop, and lands once the path heals.
func TestAStorageNodeThatDropsOffHoldsItsSearchUntilItIsBack(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := launchSubtitles(t, func(s *scenario.Scenario) {
			s.Nodes = []scenario.Node{
				{Name: "home", Setup: scenario.SetupController, Roles: []role.Name{role.Metadata, role.Notifying, role.Serving}},
				{Name: "nas", Setup: scenario.SetupJoin, Roles: []role.Name{role.Storage, role.Converting}},
				{Name: "subs", Setup: scenario.SetupJoin, Roles: []role.Name{role.Subtitles}},
			}
			for i := range s.Library.Roots {
				s.Library.Roots[i].Node = "nas"
			}
		})
		s.want("Movies", english("allow"))
		s.scanned(brick)
		simtest.Flip(t, s.w.Path("home", "nas"), fault.Stall{})
		link := node.Defaults(contenttype.Set{}, install.Dev).Timings.Link
		time.Sleep(link.Keepalive + link.WriteTimeout)
		synctest.Wait()
		s.searchNow(brick)
		time.Sleep(5 * time.Minute)
		synctest.Wait()
		if got := s.englishOf(brick); !slices.Equal(got, []string{"missing never_searched"}) {
			t.Fatalf("while nas is away Brick Lane Echo reads %q, want its search still to come", got)
		}
		var failed int
		for _, line := range s.w.Node("home").Log() {
			if strings.Contains(line, "A subtitle search failed") || strings.Contains(line, "An audio scan call failed") {
				failed++
			}
		}
		if failed > 5 {
			t.Fatalf("in 5 minutes with nas away the Controller logged %d failed calls to it, want at most one a minute", failed)
		}
		s.w.Path("home", "nas").Clear("")
		s.await(brick, "met en.srt provider subsource")
	})
}
