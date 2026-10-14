package node

import (
	"errors"
	"path/filepath"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
)

// The Worker's source holds no payload yet, so its answer proves the
// Controller's stream reached it.
func TestTheControllerReceivesFromItsWorkersSource(t *testing.T) {
	net := fault.NewNetwork()
	homeEnv, homeHost := machine(t, net, "127.0.0.2", "home")
	boxEnv, boxHost := machine(t, net, "127.0.0.3", "box")
	synctest.Test(t, func(t *testing.T) {
		home, hc := startOn(t, homeEnv, homeHost)
		apiKey, code := claimedHome(t, home, hc)
		box, bc := startOn(t, boxEnv, boxHost)
		joined := joinBox(t, box, bc, boxEnv.DataDir, code, apiKey)
		synctest.Wait()
		boxID, err := storage.ParseUUID(joined.NodeID)
		if err != nil {
			t.Fatal(err)
		}
		err = home.half.Load().transfers.Receive(t.Context(), transfers.Job{
			ID:       transfers.ID{UUID: storage.NewUUID()},
			Source:   role.NodeID{UUID: boxID},
			InfoHash: bencode.InfoHash{1},
			Staging:  filepath.Join(t.TempDir(), "movies", ".slipstream-staging", "grab.part"),
		})
		if !errors.Is(err, transfers.ErrNotPresent) {
			t.Fatalf("receiving from box = %v, want ErrNotPresent", err)
		}
	})
}
