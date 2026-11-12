package sim_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/transmission"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1/storagev1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// TestTheLadderFaultsReachTheStore imports a Grab from Transmission into a
// movies root that can neither link nor clone, so the placed file is a
// copy of the download, then asks home's Store to place the download
// again under each ladder fault: a place without a copy needs one, a copy
// into a full volume and a clone that fills it need space, and with every
// fault lifted the place is a link.
func TestTheLadderFaultsReachTheStore(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w, r := suite{kind: transmission.Fake.Name, version: transmission.Versions[len(transmission.Versions)-1]}.launch(t, func(*scenario.DownloadClient) {})
		movies := filepath.Join(w.Manifest().MediaDir, "movies")
		liftLinks := w.Disk("home", movies, fault.NoLinks{})
		liftClones := w.Disk("home", movies, fault.NoClones{})
		simtest.Search(t, w, simtest.AddOrchard(t, w))
		expectImportedFrom(t, simtest.Await(t, w))
		expectPlaced(t, w)
		copied, err := os.Stat(filepath.Join(w.Manifest().MediaDir, filepath.FromSlash(placed)))
		if err != nil {
			t.Fatal(err)
		}
		if downloaded := r.downloaded(); downloaded == nil || os.SameFile(downloaded, copied) {
			t.Fatal("the placed file is the client's download linked, want a copy where the root can't link or clone")
		}

		place := storePlace(t, w, movies)
		var needsCopy *storing.NeedsCopyError
		if _, err := place("needs a copy.mkv", false); !errors.As(err, &needsCopy) || needsCopy.Size != sim.PayloadSize {
			t.Fatalf("a place without a copy where the root can't link or clone = %v, want a copy of %d bytes needed", err, sim.PayloadSize)
		}
		liftFull := w.Disk("home", movies, fault.Full{})
		if _, err := place("full.mkv", true); !errors.As(err, new(*storing.NeedsSpaceError)) {
			t.Fatalf("a copy into a full root = %v, want space needed", err)
		}
		liftFull()
		liftClones()
		liftCloneFull := w.Disk("home", movies, fault.CloneFull{})
		if _, err := place("clone full.mkv", false); !errors.As(err, new(*storing.NeedsSpaceError)) {
			t.Fatalf("a clone that fills the root = %v, want space needed", err)
		}
		liftCloneFull()
		liftLinks()
		if p, err := place("linked.mkv", false); err != nil || p.Step != storing.Link {
			t.Fatalf("a place with every fault lifted = %+v, %v; want a link", p, err)
		}
	})
}

// storePlace places the release's download under a name in the Work's
// folder of the root at path, through home's Store as the Controller
// calls it.
func storePlace(t *testing.T, w *sim.World, path string) func(name string, allowCopy bool) (storing.Placed, error) {
	t.Helper()
	root, err := storage.ParseUUID(simtest.RootAt(t, w.API(), path).ID)
	if err != nil {
		t.Fatal(err)
	}
	cluster := w.Node("home").Half().Cluster
	store := storing.Remote(func(id role.NodeID) storagev1connect.StorageServiceHandler {
		return storagev1connect.StorageService.On(cluster, id)
	})
	at := storing.At{Node: nodeID(t, w, "home"), Root: root}
	payload := storing.Payload{InfoHash: w.InfoHash(release)}
	return func(name string, allowCopy bool) (storing.Placed, error) {
		rel, err := storing.ParseRel("The Quiet Orchard (2021)/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return store.Place(t.Context(), at, payload, rel, allowCopy)
	}
}
