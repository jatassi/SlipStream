package storing

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"

	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestStagingIsATransfersFolderOnAReadyRoot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		id, transfer := storage.NewUUID(), storage.NewUUID()
		folder := marked(t, id)
		s := newStore(newTestDisk(t))
		running(t, s, intentOf(rootAt(id, folder, false))).next(t)
		want := filepath.Join(folder, StagingDir, transfer.String())
		for range 2 {
			got, err := s.Staging(t.Context(), id, 1, transfer)
			if err != nil || got != want {
				t.Fatalf("Staging = %q, %v; want %q", got, err, want)
			}
		}
		if info, err := os.Stat(want); err != nil || !info.IsDir() {
			t.Errorf("the staging folder reads %v, %v; want a folder", info, err)
		}
	})
}

func TestStagingRefusesARootThatTakesNoBytes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDisk(t)
		ready, unmarked, unwritable := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
		readyAt, unmarkedAt, unwritableAt := marked(t, ready), t.TempDir(), marked(t, unwritable)
		d.flag(d.readOnly, unwritableAt)
		intent := intentOf(rootAt(ready, readyAt, false), rootAt(unmarked, unmarkedAt, false), rootAt(unwritable, unwritableAt, false))
		s := newStore(d)
		h := running(t, s, intent)
		h.next(t)
		for _, c := range []struct {
			name  string
			root  storage.UUID
			place int64
			want  NotReady
		}{
			{"a root the Store doesn't hold", storage.NewUUID(), 1, NotReady{Unknown: true}},
			{"a root at another place", ready, 2, NotReady{Unknown: true}},
			{"a folder without its marker", unmarked, 1, NotReady{State: Unmounted}},
			{"a folder that takes no writes", unwritable, 1, NotReady{State: Unwritable}},
		} {
			_, err := s.Staging(t.Context(), c.root, c.place, storage.NewUUID())
			if nr := notReady(t, err); *nr != c.want {
				t.Errorf("Staging on %s = %+v, want %+v", c.name, *nr, c.want)
			}
		}
		h.send(role.Snapshot[*storagev1.Intent]{Intent: intent, Standing: role.Live, Draining: true})
		h.next(t)
		_, err := s.Staging(t.Context(), ready, 1, storage.NewUUID())
		if nr := notReady(t, err); !nr.Draining {
			t.Errorf("Staging while the Node drains Storage = %+v, want draining", *nr)
		}
		for _, folder := range []string{readyAt, unmarkedAt, unwritableAt} {
			if _, err := os.Stat(filepath.Join(folder, StagingDir)); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("a refused Staging made %s's staging folder: %v", folder, err)
			}
		}
	})
}
