package sim_test

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/health"
)

type healthItem struct {
	Key       string
	Level     string
	Subject   string
	Text      string
	Since     time.Time
	ChangedAt time.Time
	Unsure    bool
	Action    *struct{ Kind, Href, Label string }
}

type healthReport struct {
	Categories []struct {
		ID     string
		Unread string
		Items  []healthItem
	}
}

func (r healthReport) items(t *testing.T, id string) map[string]healthItem {
	t.Helper()
	for _, c := range r.Categories {
		if c.ID == id {
			if c.Unread != "" {
				t.Fatalf("Health couldn't read %s: %s", id, c.Unread)
			}
			out := map[string]healthItem{}
			for _, it := range c.Items {
				out[it.Key] = it
			}
			return out
		}
	}
	t.Fatalf("the Health report has no category %s", id)
	return nil
}

func healthItems(t *testing.T, a *worldtest.API, id string) map[string]healthItem {
	t.Helper()
	return decode[healthReport](t, "getSystemHealth", a.Admin("home", http.MethodGet, "/api/v1/system/health", "")).items(t, id)
}

type healthChecked struct {
	Report    healthReport
	Unreached []struct{ Key, Words string }
}

// checkStorage is Check now on the Storage item key, or on the whole
// category when key is "".
func checkStorage(t *testing.T, a *worldtest.API, key string) healthChecked {
	t.Helper()
	path := "/api/v1/system/health/storage/check"
	if key != "" {
		path = "/api/v1/system/health/storage/items/" + key + "/check"
	}
	return decode[healthChecked](t, "Check now on storage "+key, a.Admin("home", http.MethodPost, path, ""))
}

type healthVolumes struct {
	Volumes []struct {
		Mount        string
		WarnBytes    int64
		ErrorBytes   int64
		SkipsWarning bool
		Level        *string
	}
	NotReporting []struct {
		NodeID    string
		Mount     string
		WarnBelow *struct{ Amount int64 }
	}
}

func listHealthVolumes(t *testing.T, a *worldtest.API) healthVolumes {
	t.Helper()
	return decode[healthVolumes](t, "listHealthVolumes", a.Admin("home", http.MethodGet, "/api/v1/system/health/volumes", ""))
}

func patch(t *testing.T, a *worldtest.API, path, body string) {
	t.Helper()
	if r := a.Admin("home", http.MethodPatch, path, body); r.Status != http.StatusOK {
		t.Fatalf("PATCH %s %s = %d %s, want 200", path, body, r.Status, r.Body)
	}
}

// space plays a 500 GiB volume on folder of home's, with free GiB free,
// replacing the figures it played before; off lifts them.
type space struct {
	w      *sim.World
	folder string
	lift   func()
}

func (s *space) free(gib int64) {
	s.off()
	s.lift = s.w.Disk("home", s.folder, fault.Volume{Total: 500 << 30, Free: gib << 30})
}

func (s *space) off() {
	if s.lift != nil {
		s.lift()
		s.lift = nil
	}
}

type moviesOnHome struct {
	w      *sim.World
	a      *worldtest.API
	home   string
	folder string
	// volume is the Storage key of the movies root's volume while space
	// plays it.
	volume string
	space  *space
}

// launchMoviesOnHome launches home at the default Storage levels, which
// the figures space plays are judged by.
func launchMoviesOnHome(t *testing.T) moviesOnHome {
	t.Helper()
	s := home(t)
	s.Library.StorageLevels = nil
	w := sim.Launch(t, s, sim.WithWalk())
	synctest.Wait()
	a := w.API()
	m := moviesOnHome{w: w, a: a, home: a.Node("home").NodeID, folder: filepath.Join(w.Manifest().MediaDir, "movies")}
	m.volume = string(health.KeyOf("v", m.home, health.Opaque(m.folder)))
	m.space = &space{w: w, folder: m.folder}
	return m
}

// at plays free GiB on the movies volume, checks storage now, and answers
// the volume's item after.
func (m moviesOnHome) at(t *testing.T, gib int64) healthItem {
	t.Helper()
	m.space.free(gib)
	time.Sleep(time.Second)
	return checkStorage(t, m.a, "").Report.items(t, "storage")[m.volume]
}

// TestAVolumeWarnsThenFailsThenRecovers checks the movies volume as its
// free space falls past the default 100 GB warning and 20 GB error, then
// recovers: since holds through the worse step, Check now on the item
// samples it, and the item clears once space is back.
func TestAVolumeWarnsThenFailsThenRecovers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := launchMoviesOnHome(t)
		if it := m.at(t, 200); it.Level != "" {
			t.Fatalf("with 200 GB free the volume is %+v, want no item", it)
		}
		warned := m.at(t, 50)
		name := m.folder + " on home"
		if warned.Level != "warn" || warned.Subject != name || warned.Action == nil || warned.Action.Kind != "check" ||
			warned.Text != "Free up space on "+name+", or add a library root on another volume. It has 50 GB left." {
			t.Fatalf("with 50 GB free the volume is %+v, want a warning on %s offering Check now", warned, name)
		}
		m.space.free(10)
		time.Sleep(time.Second)
		failed := checkStorage(t, m.a, m.volume).Report.items(t, "storage")[m.volume]
		if failed.Level != "fail" || !failed.Since.Equal(warned.Since) || !failed.ChangedAt.After(warned.ChangedAt) {
			t.Fatalf("Check now on the volume with 10 GB free = %+v, want an error since %s, changed after %s", failed, warned.Since, warned.ChangedAt)
		}
		m.space.free(200)
		if it, ok := checkStorage(t, m.a, "").Report.items(t, "storage")[m.volume]; ok {
			t.Fatalf("with 200 GB free again the volume is %+v, want it cleared", it)
		}
	})
}

// TestAVolumesLevelsAreSetGloballyAndPerVolume judges 200 GB free of 500
// against a 50% global warning, then the volume's own 30% warning and
// 250 GB error, then resets both to the defaults.
func TestAVolumesLevelsAreSetGloballyAndPerVolume(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := launchMoviesOnHome(t)
		if it := m.at(t, 200); it.Level != "" {
			t.Fatalf("with 200 GB free the volume is %+v, want no item", it)
		}
		level := func() string { return healthItems(t, m.a, "storage")[m.volume].Level }
		patch(t, m.a, "/api/v1/system/health/settings", `{"warnBelow":{"amount":50,"unit":"percent"}}`)
		if got := level(); got != "warn" {
			t.Fatalf("with 40%% free under a 50%% global warning the volume reads %q, want warn", got)
		}
		volume := `{"nodeId":"` + m.home + `","mount":` + strconv.Quote(m.folder) + `,`
		patch(t, m.a, "/api/v1/system/health/volumes", volume+`"warnBelow":{"amount":30,"unit":"percent"}}`)
		if got := level(); got != "" {
			t.Fatalf("with 40%% free under its own 30%% warning the volume reads %q, want nothing", got)
		}
		patch(t, m.a, "/api/v1/system/health/volumes", volume+`"errorBelow":{"amount":`+bytesOf(250)+`,"unit":"bytes"}}`)
		if got := level(); got != "fail" {
			t.Fatalf("with 200 GB free under its own 250 GB error the volume reads %q, want fail", got)
		}
		volumes := listHealthVolumes(t, m.a).Volumes
		movies := false
		for _, v := range volumes {
			movies = movies || v.Mount == m.folder && v.SkipsWarning && v.WarnBytes == 150<<30 && v.ErrorBytes == 250<<30
		}
		if !movies {
			t.Fatalf("listHealthVolumes = %+v, want the movies volume skipping its 150 GB warning for its 250 GB error", volumes)
		}
		patch(t, m.a, "/api/v1/system/health/volumes", volume+`"warnBelow":null,"errorBelow":null}`)
		if got := level(); got != "warn" {
			t.Fatalf("with its own levels reset the volume reads %q, want the global 50%% warning", got)
		}
		patch(t, m.a, "/api/v1/system/health/settings", `{"warnBelow":null}`)
		if got := level(); got != "" {
			t.Fatalf("with every level reset the volume reads %q, want nothing", got)
		}
	})
}

// TestAVolumesOwnLevelsOutliveItsFigures gives the movies volume its own
// warning, then lifts its figures so the Node reports another volume for
// the root: the levels list as not reporting, apply again once the volume
// is back under its mount, and go with Remove. The simulated volume's id
// is its mount, so a remount here can't change the id; the levels are keyed
// by mount alone.
func TestAVolumesOwnLevelsOutliveItsFigures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := launchMoviesOnHome(t)
		m.at(t, 200)
		volume := `{"nodeId":"` + m.home + `","mount":` + strconv.Quote(m.folder) + `,`
		patch(t, m.a, "/api/v1/system/health/volumes", volume+`"warnBelow":{"amount":`+bytesOf(300)+`,"unit":"bytes"}}`)
		if got := healthItems(t, m.a, "storage")[m.volume].Level; got != "warn" {
			t.Fatalf("with 200 GB free under its own 300 GB warning the volume reads %q, want warn", got)
		}
		gone := func(when string) {
			t.Helper()
			m.space.off()
			checkStorage(t, m.a, "")
			not := listHealthVolumes(t, m.a).NotReporting
			if len(not) != 1 || not[0].NodeID != m.home || not[0].Mount != m.folder || not[0].WarnBelow == nil || not[0].WarnBelow.Amount != 300<<30 {
				t.Fatalf("%s the overrides not reporting are %+v, want the movies volume's 300 GB warning", when, not)
			}
		}
		gone("with the figures lifted")
		if got := m.at(t, 200).Level; got != "warn" {
			t.Fatalf("back under its mount the volume reads %q, want its own warning again", got)
		}
		gone("lifted again")
		if r := m.a.Admin("home", http.MethodDelete, "/api/v1/system/health/volumes?nodeId="+m.home+"&mount="+url.QueryEscape(m.folder), ""); r.Status != http.StatusNoContent {
			t.Fatalf("deleteHealthVolume = %d %s, want 204", r.Status, r.Body)
		}
		if not := listHealthVolumes(t, m.a).NotReporting; len(not) != 0 {
			t.Fatalf("after Remove the overrides not reporting are %+v, want none", not)
		}
	})
}

// TestAnUnmountedRootIsAStorageErrorThroughARestart unmounts the movies
// root: Check now raises its error, a Controller restart keeps its since,
// and once the disk is back Check now on the item clears it.
func TestAnUnmountedRootIsAStorageErrorThroughARestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := launchMoviesOnHome(t)
		root := string(health.KeyOf("r", rootAt(t, m.a, m.folder).ID))
		lift := m.w.Disk("home", m.folder, fault.Unmounted{})
		raised := checkStorage(t, m.a, "").Report.items(t, "storage")[root]
		if raised.Level != "fail" || raised.Subject != "Movies library root on home" || raised.Action == nil || raised.Action.Kind != "check" ||
			raised.Text != "The folder is there but looks unmounted. Plug in or mount the disk." {
			t.Fatalf("the unmounted root's item is %+v, want an error on the Movies library root offering Check now", raised)
		}

		h := m.w.Node("home")
		h.Stop()
		time.Sleep(time.Minute)
		h.Start()
		synctest.Wait()
		m.a = m.w.API()
		if after := checkStorage(t, m.a, "").Report.items(t, "storage")[root]; after.Level != "fail" || after.Unsure || !after.Since.Equal(raised.Since) {
			t.Fatalf("after a Controller restart the root's item is %+v, want the error since %s", after, raised.Since)
		}

		lift()
		if it, ok := checkStorage(t, m.a, root).Report.items(t, "storage")[root]; ok {
			t.Fatalf("Check now on the root with its disk back left %+v, want it cleared", it)
		}
	})
}

// TestARootOnAnOfflineNodeRaisesOnlyTheNodesItem puts the movies root on
// seedbox and stops seedbox past a minute's grace: Nodes holds seedbox's
// error and Storage nothing, and Check now on Storage answers seedbox
// unreached.
func TestARootOnAnOfflineNodeRaisesOnlyTheNodesItem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := home(t)
		s.Nodes[1].Roles = append(s.Nodes[1].Roles, "storage")
		s.Library.Roots = []scenario.LibraryRoot{{ContentType: "movie", Folder: "movies", Node: "seedbox", Volume: &scenario.LibraryVolume{TotalGiB: 500, FreeGiB: 200}}}
		w := sim.Launch(t, s, sim.WithWalk())
		synctest.Wait()
		a := w.API()
		patch(t, a, "/api/v1/nodes/settings", `{"offlineGrace":60}`)
		box := a.Node("seedbox").NodeID
		w.Node("seedbox").Stop()
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		if it := healthItems(t, a, "nodes")[box]; it.Level != "fail" {
			t.Fatalf("past the grace seedbox's Nodes item is %+v, want an error", it)
		}
		checked := checkStorage(t, a, "")
		if items := checked.Report.items(t, "storage"); len(items) != 0 {
			t.Fatalf("with seedbox offline Storage holds %+v, want nothing", items)
		}
		if len(checked.Unreached) != 1 || !strings.HasPrefix(checked.Unreached[0].Words, "seedbox is offline, so its figures are as of ") {
			t.Fatalf("Check now on Storage with seedbox offline left %+v unreached, want seedbox", checked.Unreached)
		}
	})
}

func bytesOf(gib int64) string { return strconv.FormatInt(gib<<30, 10) }
