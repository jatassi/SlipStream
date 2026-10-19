package sim_test

import (
	"net/http"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node"
)

type volumeList struct {
	Nodes []struct {
		NodeID  string `json:"nodeId"`
		Live    bool   `json:"live"`
		Volumes []struct {
			ID        string    `json:"id"`
			Total     int64     `json:"total"`
			Free      int64     `json:"free"`
			SampledAt time.Time `json:"sampledAt"`
			RootIDs   []string  `json:"rootIds"`
		} `json:"volumes"`
	} `json:"nodes"`
}

func volumes(t *testing.T, a *worldtest.API) volumeList {
	t.Helper()
	return decode[volumeList](t, "listVolumes", a.Admin("home", http.MethodGet, "/api/v1/volumes", ""))
}

// TestAnOfflineNodesVolumesReadAsOf puts a Movies root on seedbox with
// 2.5 GiB free of 10 and cuts seedbox off: its volume still lists those
// figures as of their sample, not live.
func TestAnOfflineNodesVolumesReadAsOf(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := home(t)
		s.Nodes[1].Roles = append(s.Nodes[1].Roles, "storage")
		s.Library.Roots = []scenario.LibraryRoot{{ContentType: "movie", Folder: "movies", Node: "seedbox", Volume: &scenario.LibraryVolume{TotalGiB: 10, FreeGiB: 2.5}}}
		w := sim.Launch(t, s, sim.WithWalk())
		synctest.Wait()
		a := w.API()
		seedbox := a.Node("seedbox").NodeID
		before := volumes(t, a)
		if len(before.Nodes) != 1 || before.Nodes[0].NodeID != seedbox || !before.Nodes[0].Live || len(before.Nodes[0].Volumes) != 1 {
			t.Fatalf("listVolumes with seedbox online = %+v, want its one volume, live", before)
		}
		flip(t, w.Path("home", "seedbox"), fault.Stall{})
		link := node.Defaults(contenttype.Set{}, install.Dev).Timings.Link
		time.Sleep(link.Keepalive + link.WriteTimeout)
		synctest.Wait()
		after := volumes(t, a)
		if len(after.Nodes) != 1 || after.Nodes[0].Live || len(after.Nodes[0].Volumes) != 1 {
			t.Fatalf("listVolumes with seedbox offline = %+v, want its one volume, not live", after)
		}
		if v, was := after.Nodes[0].Volumes[0], before.Nodes[0].Volumes[0]; v.Total != 10<<30 || v.Free != 5<<29 || !v.SampledAt.Equal(was.SampledAt) {
			t.Fatalf("seedbox's volume offline reads %+v, want 2.5 GiB free of 10 as of %s", v, was.SampledAt)
		}
	})
}

// TestCheckStorageEveryReachesTheNode sets Check storage every to a
// minute: a volume whose figures change reports them within the minute
// and the stat deadline, not the default five minutes.
func TestCheckStorageEveryReachesTheNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithWalk())
		synctest.Wait()
		a := w.API()
		if r := a.Admin("home", http.MethodPatch, "/api/v1/roots/settings", `{"sampleEvery":60}`); r.Status != http.StatusOK {
			t.Fatalf("updateRootSettings = %d %s, want 200", r.Status, r.Body)
		}
		synctest.Wait()
		folder := filepath.Join(w.Manifest().MediaDir, "movies")
		w.Disk("home", folder, fault.Volume{Total: 10 << 30, Free: 2 << 30})
		time.Sleep(time.Minute + 10*time.Second + time.Second)
		synctest.Wait()
		list := volumes(t, a)
		if len(list.Nodes) != 1 || len(list.Nodes[0].Volumes) != 1 {
			t.Fatalf("listVolumes = %+v, want home's one volume", list)
		}
		if v := list.Nodes[0].Volumes[0]; v.ID != "sim:"+folder || v.Free != 2<<30 || v.Total != 10<<30 {
			t.Fatalf("a minute after the figures changed the volume reads %+v, want 2 GiB free of 10 on sim:%s", v, folder)
		}
	})
}
