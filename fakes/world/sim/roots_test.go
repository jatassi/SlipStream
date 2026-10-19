package sim_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storing"
)

type rootRow struct {
	ID       string `json:"id"`
	NodeID   string `json:"nodeId"`
	Path     string `json:"path"`
	State    string `json:"state"`
	VolumeID string `json:"volumeId"`
}

func rootAt(t *testing.T, a *worldtest.API, path string) rootRow {
	t.Helper()
	reply := a.Admin("home", http.MethodGet, "/api/v1/roots", "")
	var list struct{ Roots []rootRow }
	if err := json.Unmarshal([]byte(reply.Body), &list); reply.Status != http.StatusOK || err != nil {
		t.Fatalf("listRoots = %d %s, %v", reply.Status, reply.Body, err)
	}
	for _, r := range list.Roots {
		if r.Path == path {
			return r
		}
	}
	t.Fatalf("listRoots has no root at %s: %s", path, reply.Body)
	return rootRow{}
}

// TestAVolumeReportsItsScenarioFigures puts a Movies root on seedbox with
// 2.5 GiB free of 10, and seedbox reports the root on that volume.
func TestAVolumeReportsItsScenarioFigures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := home(t)
		s.Nodes[1].Roles = append(s.Nodes[1].Roles, "storage")
		s.Library.Roots = append(s.Library.Roots, scenario.LibraryRoot{
			ContentType: "movie", Folder: "films", Node: "seedbox", Volume: &scenario.LibraryVolume{TotalGiB: 10, FreeGiB: 2.5},
		})
		w := sim.Launch(t, s, sim.WithWalk())
		synctest.Wait()
		a := w.API()
		folder := filepath.Join(w.Manifest().MediaDir, "films")
		root := rootAt(t, a, folder)
		if want := "sim:" + folder; root.NodeID != a.Node("seedbox").NodeID || root.State != "ready" || root.VolumeID != want {
			t.Fatalf("the films root is %+v; want ready on seedbox on volume %s", root, want)
		}
		reply := a.Admin("home", http.MethodGet, "/api/v1/nodes/"+root.NodeID+"/folders?path="+url.QueryEscape(folder), "")
		var listing struct {
			Volume struct {
				ID          string
				Total, Free int64
			}
		}
		if err := json.Unmarshal([]byte(reply.Body), &listing); reply.Status != http.StatusOK || err != nil {
			t.Fatalf("browse the films root = %d %s, %v", reply.Status, reply.Body, err)
		}
		if v := listing.Volume; v.ID != root.VolumeID || v.Total != 10<<30 || v.Free != 5<<29 {
			t.Fatalf("the films root's volume reads %+v; want 2.5 GiB free of 10", v)
		}
	})
}

// TestAHungShareReadsUnresponsive hangs the movies root's folder past the
// stat deadline: the next sample reads it unresponsive without stalling,
// and once the share answers the one after reads it ready again.
func TestAHungShareReadsUnresponsive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithWalk())
		synctest.Wait()
		a := w.API()
		folder := filepath.Join(w.Manifest().MediaDir, "movies")
		if root := rootAt(t, a, folder); root.State != "ready" {
			t.Fatalf("the movies root is %+v before the fault, want ready", root)
		}
		sample := roots.SampleEvery.Default + storing.DefaultTimings().Stat + time.Second
		lift := w.Disk("home", folder, fault.Hung{})
		time.Sleep(sample)
		synctest.Wait()
		root := rootAt(t, a, folder)
		if root.State != "unresponsive" {
			t.Fatalf("the movies root is %+v with its share hung, want unresponsive", root)
		}
		reply := a.Admin("home", http.MethodGet, "/api/v1/nodes/"+root.NodeID+"/folders?path="+url.QueryEscape(folder), "")
		var problem struct {
			Code   string
			Detail struct{ Kind string }
		}
		_ = json.Unmarshal([]byte(reply.Body), &problem)
		if reply.Status != http.StatusUnprocessableEntity || problem.Code != "folder_unreadable" {
			t.Fatalf("browse the hung folder = %d %s; want 422 folder_unreadable", reply.Status, reply.Body)
		}
		lift()
		time.Sleep(sample)
		synctest.Wait()
		if root := rootAt(t, a, folder); root.State != "ready" {
			t.Fatalf("the movies root is %+v once its share answers, want ready", root)
		}
	})
}

// TestAnImportIntoAnUnmountedRootWaitsUntilItIsBack unmounts the movies
// root just as the Grab imports: the import waits, the root reads
// unmounted, and once the disk is back the next sample lets the Grab
// import.
func TestAnImportIntoAnUnmountedRootWaitsUntilItIsBack(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reached, unmounted := make(chan struct{}), make(chan struct{})
		var held atomic.Bool
		w := sim.Launch(t, home(t), sim.WithWalk(), sim.Edit(func(_ string, e *node.Env) {
			e.Test.BeforeStep = func(_ context.Context, s grabs.Stage) {
				if s == grabs.Importing && held.CompareAndSwap(false, true) {
					close(reached)
					<-unmounted
				}
			}
		}))
		search(t, w, addOrchard(t, w))
		<-reached
		folder := filepath.Join(w.Manifest().MediaDir, "movies")
		lift := w.Disk("home", folder, fault.Unmounted{})
		close(unmounted)
		time.Sleep(time.Minute)
		synctest.Wait()
		if gs := grabsListed(t, w); len(gs) != 1 || gs[0].Stage != "importing" || gs[0].Failure != nil {
			t.Fatalf("the Grabs with the root unmounted are %+v, want one waiting at importing", gs)
		}
		if root := rootAt(t, w.API(), folder); root.State != "unmounted" {
			t.Fatalf("the movies root is %+v while its disk is away, want unmounted", root)
		}
		lift()
		time.Sleep(5*time.Minute + 10*time.Second + time.Second)
		expectImported(t, await(t, w))
		expectPlaced(t, w)
	})
}

// TestAWorkOnAnOfflineNodesRootDoesntSearch puts the movies root on
// seedbox and cuts seedbox off: a search of a movie is refused, naming
// seedbox.
func TestAWorkOnAnOfflineNodesRootDoesntSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := home(t)
		s.Nodes[1].Roles = append(s.Nodes[1].Roles, "storage")
		s.Library.Roots = []scenario.LibraryRoot{{ContentType: "movie", Folder: "movies", Node: "seedbox"}}
		w := sim.Launch(t, s, sim.WithWalk())
		synctest.Wait()
		work := addOrchard(t, w)
		flip(t, w.Path("home", "seedbox"), fault.Stall{})
		link := node.Defaults(contenttype.Set{}, install.Dev).Timings.Link
		time.Sleep(link.Keepalive + link.WriteTimeout)
		synctest.Wait()
		worldtest.Expect(t, "searchWork", w.API().Admin("home", http.MethodPost, "/api/v1/works/"+work+"/search", ""), worldtest.Reply{
			Status: http.StatusConflict,
			Body: `{"status":409,"code":"root_unreachable","title":"The Work's library root is on a Node that takes no new files now",` +
				`"detail":"seedbox is offline, so the library root there takes no new files."}` + "\n",
		})
	})
}

// TestASeedboxWithoutStorageBrowsesItsFolders browses seedbox, which holds
// no Storage, and has it judge a downloads folder but refuse to judge a
// root.
func TestASeedboxWithoutStorageBrowsesItsFolders(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t))
		synctest.Wait()
		a := w.API()
		seedbox := a.Node("seedbox").NodeID
		data := w.Node("seedbox").DataDir().Path()
		reply := a.Admin("home", http.MethodGet, "/api/v1/nodes/"+seedbox+"/folders?path="+url.QueryEscape(data), "")
		var listing struct{ Path string }
		if err := json.Unmarshal([]byte(reply.Body), &listing); reply.Status != http.StatusOK || err != nil || listing.Path != data {
			t.Fatalf("browse seedbox's data folder = %d %s, %v", reply.Status, reply.Body, err)
		}
		check := "/api/v1/nodes/" + seedbox + "/folders/check"
		reply = a.Admin("home", http.MethodPost, check, `{"purpose":"downloads","path":`+strconv.Quote(data)+`}`)
		if reply.Status != http.StatusOK || !strings.HasPrefix(reply.Body, `{"usable":true,`) {
			t.Fatalf("check seedbox's data folder for downloads = %d %s, want usable", reply.Status, reply.Body)
		}
		reply = a.Admin("home", http.MethodPost, check, `{"purpose":"root","path":`+strconv.Quote(data)+`}`)
		if reply.Status != http.StatusConflict || !strings.Contains(reply.Body, `"code":"node_not_storage"`) {
			t.Fatalf("check seedbox's data folder as a root = %d %s, want node_not_storage", reply.Status, reply.Body)
		}
	})
}
