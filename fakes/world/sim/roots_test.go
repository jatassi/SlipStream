package sim_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
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
		const sample = 5*time.Minute + 10*time.Second + time.Second // a sampling interval, then the stat deadline
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
