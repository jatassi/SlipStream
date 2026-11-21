package transfers_test

import (
	"bytes"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/zeebo/blake3"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

const backupName = "slipstream-20261114T040012Z-2.1.0-scheduled-7k3m.tar.zst.age"

// backupCopy is a copy of one Backup from the Backups folder on sender to
// a Backup destination on receiver.
type backupCopy struct {
	t                *testing.T
	id               storage.UUID
	sender, receiver *end
	from, to         string
	name             string
	file             *tv1.File
	data             []byte
}

func newBackupCopy(t *testing.T, sender, receiver *end, name string) *backupCopy {
	t.Helper()
	c := &backupCopy{
		t: t, id: storage.NewUUID(), sender: sender, receiver: receiver, name: name,
		from: filepath.Join(sender.dir, "backups"), to: filepath.Join(receiver.dir, "destination"),
		data: make([]byte, 2*chunk+99),
	}
	_, _ = rand.NewChaCha8([32]byte{9}).Read(c.data)
	for _, dir := range []string{c.from, c.to} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(c.from, backupName), c.data, 0o600); err != nil {
		t.Fatal(err)
	}
	c.file = &tv1.File{Size: int64(len(c.data)), Source: &tv1.Source{Ref: &tv1.Source_Backup{Backup: &tv1.BackupPlace{Folder: c.from, Name: name}}}}
	return c
}

// run gives both ends an intent that copies the Backup, each listing the
// folders given.
func (c *backupCopy) run(senderFolders, receiverFolders []*tv1.BackupFolder) {
	active := func(side tv1.Side, peer *end) *tv1.Active {
		return &tv1.Active{
			TransferId: c.id.Bytes(), Side: side, Route: tv1.Route_ROUTE_RECEIVER_DIALS, Files: []*tv1.File{c.file},
			Peer: &tv1.Peer{Key: peer.ln.Key[:], Addresses: peer.addrs()},
		}
	}
	send, recv := active(tv1.Side_SIDE_SEND, c.receiver), active(tv1.Side_SIDE_RECEIVE, c.sender)
	recv.Destination = &tv1.Destination{Kind: &tv1.Destination_Backup{Backup: &tv1.BackupPlace{Folder: c.to, Name: c.name}}}
	c.sender.h.set(&tv1.Intent{Active: []*tv1.Active{send}, BackupFolders: senderFolders}, role.Live)
	c.receiver.h.set(&tv1.Intent{Active: []*tv1.Active{recv}, Keep: []*tv1.Kept{{TransferId: c.id.Bytes(), Positions: []uint32{0}}}, BackupFolders: receiverFolders}, role.Live)
}

func (c *backupCopy) arrived(rs []*tv1.Result) bool {
	for _, r := range rs {
		if v := r.GetVerified(); v != nil && bytes.Equal(v.GetTransferId(), c.id.Bytes()) {
			return true
		}
	}
	return false
}

// folderFacts is the last facts' Backup folder at folder.
func folderFacts(fs []*tv1.Facts, folder string) *tv1.BackupFolderFacts {
	if len(fs) == 0 {
		return nil
	}
	for _, f := range fs[len(fs)-1].GetBackupFolders() {
		if f.GetFolder() == folder {
			return f
		}
	}
	return nil
}

// TestAWorkerReceivesOnlyCiphertextAndNames: a copy lands in the Backup
// destination under its Backup name with the very bytes of the Backups
// folder's file, and nothing else is left beside it but the hidden
// staging folder. The Worker reports the names it holds, answers a keyed
// check with the keyed BLAKE3 of those bytes, and deletes a Backup only
// by its name.
func TestAWorkerReceivesOnlyCiphertextAndNames(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		controller, w, _ := trio(t)
		c := newBackupCopy(t, controller, w[0], backupName)
		controller.start()
		w[0].start()
		c.run([]*tv1.BackupFolder{{Folder: c.from}}, []*tv1.BackupFolder{{Folder: c.to}})
		w[0].h.await(t, "the copy verified", func(rs []*tv1.Result, _ []*tv1.Facts) bool { return c.arrived(rs) })

		got, err := os.ReadFile(filepath.Join(c.to, backupName))
		if err != nil || !bytes.Equal(got, c.data) {
			t.Fatalf("the destination holds %d bytes (%v), want the Backup's %d", len(got), err, len(c.data))
		}
		entries, err := os.ReadDir(c.to)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		if !slices.Equal(names, []string{storing.StagingDir, backupName}) {
			t.Errorf("the destination holds %v", names)
		}
		w[0].h.await(t, "the copy listed", func(_ []*tv1.Result, fs []*tv1.Facts) bool {
			return slices.Equal(folderFacts(fs, c.to).GetNames(), []string{backupName})
		})

		key := bytes.Repeat([]byte{7}, 32)
		w[0].h.set(&tv1.Intent{BackupFolders: []*tv1.BackupFolder{{Folder: c.to, CheckKey: key, Check: backupName}}}, role.Live)
		h, err := blake3.NewKeyed(key)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(h, bytes.NewReader(c.data))
		w[0].h.await(t, "the check answered", func(_ []*tv1.Result, fs []*tv1.Facts) bool {
			check := folderFacts(fs, c.to).GetCheck()
			return bytes.Equal(check.GetKey(), key) && bytes.Equal(check.GetSum(), h.Sum(nil))
		})

		outside := filepath.Join(w[0].dir, backupName)
		if err := os.WriteFile(outside, c.data, 0o600); err != nil {
			t.Fatal(err)
		}
		w[0].h.set(&tv1.Intent{BackupFolders: []*tv1.BackupFolder{{Folder: c.to, Delete: []string{backupName, "../" + backupName}}}}, role.Live)
		w[0].h.await(t, "the copy deleted", func(_ []*tv1.Result, fs []*tv1.Facts) bool {
			f := folderFacts(fs, c.to)
			return f != nil && len(f.GetNames()) == 0
		})
		if _, err := os.Stat(outside); err != nil {
			t.Fatal("a delete by a name with a separator reached outside the folder:", err)
		}
	})
}

// TestANodeOpensOnlyAValidNameInAListedFolder: a sender asked for a path
// dressed as a Backup name, a name outside the pattern or a Backup in a
// folder its intent doesn't list answers that it has no such file, and a
// receiver doesn't stage into a folder its intent doesn't list.
func TestANodeOpensOnlyAValidNameInAListedFolder(t *testing.T) {
	for _, tc := range []struct {
		name, file string
		listed     bool
	}{
		{"a planted path", "../library.key", true},
		{"a name outside the pattern", "library.db", true},
		{"a folder the intent doesn't list", backupName, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				controller, w, _ := trio(t)
				c := newBackupCopy(t, w[0], controller, tc.file)
				if err := os.WriteFile(filepath.Join(w[0].dir, "library.key"), []byte("secret"), 0o600); err != nil {
					t.Fatal(err)
				}
				controller.start()
				w[0].start()
				var listed []*tv1.BackupFolder
				if tc.listed {
					listed = []*tv1.BackupFolder{{Folder: c.from}}
				}
				c.name = backupName
				c.run(listed, []*tv1.BackupFolder{{Folder: c.to}})
				if f := controller.h.awaitFailed(t, c.id); f.GetCode() != tv1.Failed_CODE_SOURCE_GONE {
					t.Fatalf("the copy failed %s, want source gone", f.GetCode())
				}
				if _, err := os.Stat(filepath.Join(c.to, backupName)); err == nil {
					t.Fatal("a file arrived")
				}
			})
		})
	}
	t.Run("a receiving folder the intent doesn't list", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			controller, w, _ := trio(t)
			c := newBackupCopy(t, controller, w[0], backupName)
			controller.start()
			w[0].start()
			c.run([]*tv1.BackupFolder{{Folder: c.from}}, nil)
			w[0].h.await(t, "the receiver waits", func(_ []*tv1.Result, fs []*tv1.Facts) bool {
				return lastWait(fs, c.id) == tv1.Wait_KIND_ROOT_NOT_READY
			})
			if _, err := os.Stat(filepath.Join(c.to, storing.StagingDir)); err == nil {
				t.Fatal("the receiver staged in a folder its intent doesn't list")
			}
		})
	})
}
