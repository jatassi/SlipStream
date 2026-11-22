package downloading_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

func TestADownloadFolderStagesAndArrivesInItsClientsFolder(t *testing.T) {
	inEachTransmission(t, func(t *testing.T, version string) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		c := n.startComposite(version, 1000)
		c.send(true)
		external := [16]byte(c.client.GetId())
		nobody := [16]byte(storage.NewUUID().Bytes())
		var hash bencode.InfoHash
		hash[0] = 0xab
		transfer := storage.NewUUID()
		rel, err := storing.ParseRel("Heat (1995)/Heat.mkv")
		if err != nil {
			t.Fatal(err)
		}

		for _, k := range []struct {
			name           string
			client         *[16]byte
			stage, arrives string
		}{
			{"the built-in client", nil, filepath.Join(n.dir, storing.StagingDir, transfer.String()), filepath.Join(n.dir, "Movies", hash.String(), "Heat (1995)", "Heat.mkv")},
			{"Transmission", &external, filepath.Join(c.daemon.Dir, storing.StagingDir, transfer.String()), filepath.Join(c.daemon.Dir, "Movies", "Heat (1995)", "Heat.mkv")},
		} {
			got, err := c.StageDownload(t.Context(), k.client, "Movies", hash, transfer)
			if err != nil || got != k.stage {
				t.Errorf("%s: StageDownload = %q, %v; want %q", k.name, got, err, k.stage)
			}
			if fi, err := os.Stat(k.stage); err != nil || !fi.IsDir() {
				t.Errorf("%s: the staging folder isn't there: %v", k.name, err)
			}
			got, err = c.Arriving(storing.Arriving{Client: k.client, ContentType: "Movies", InfoHash: hash, Path: rel})
			if err != nil || got != k.arrives {
				t.Errorf("%s: Arriving = %q, %v; want %q", k.name, got, err, k.arrives)
			}
		}
		if _, err := c.StageDownload(t.Context(), &nobody, "Movies", hash, transfer); err == nil {
			t.Error("StageDownload for a client the Node runs no actor for succeeded")
		}
	})
}
