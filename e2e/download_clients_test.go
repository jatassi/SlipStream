package e2e

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

var clientOrchard = magnetRelease{"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW", "9000001", "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv"}

// TestExternalTransmissionImports grabs The Quiet Orchard on the
// external-clients world, where only its Transmission can download the
// release, at 4.1, the version Jackson runs, and at 4.0. The Grab imports a
// hard link of Transmission's download, the Torrent names Transmission, and
// doctor finds the address the world typed reaching Transmission's front.
func TestExternalTransmissionImports(t *testing.T) {
	t.Parallel()
	for _, version := range []string{"4.1", "4.0"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			s, err := scenario.Load("external-clients")
			if err != nil {
				t.Fatal(err)
			}
			s.Nodes[0].DownloadClients[0].Version = version
			run := worldtest.LaunchScenario(t, s)
			m := run.Manifest()
			a := api(t, run)

			g := endedGrab(t, a, grabbed(t, a, clientOrchard))
			if g.Stage != "imported" || g.InfoHash != infoHashOf(t, m, clientOrchard.name) {
				t.Fatalf("the Grab ended %s with %+v after %v, want imported", g.Stage, g.Failure, g.stages())
			}
			holdsOnly(t, m, clientOrchard.placed)
			placed := filepath.Join(m.MediaDir, filepath.FromSlash(clientOrchard.placed))
			if _, err := world.MatchRelease(m, placed, clientOrchard.name); err != nil {
				t.Fatal(err)
			}

			var torrents struct {
				Rows []struct {
					ClientKind string `json:"clientKind"`
					ClientName string `json:"clientName"`
				} `json:"rows"`
			}
			decode(t, "listTorrents", a.Admin("home", http.MethodGet, "/api/v1/torrents", ""), &torrents)
			if len(torrents.Rows) != 1 || torrents.Rows[0].ClientKind != "transmission" || torrents.Rows[0].ClientName != "Transmission" {
				t.Fatalf("listTorrents = %+v, want the one Torrent in Transmission", torrents.Rows)
			}
			expectLinkedFrom(t, transmissionFolder(t, a), clientOrchard.name+".mkv", placed)
			doctor(t, run)
		})
	}
}

// transmissionFolder is the folder Transmission downloads into, as home
// sees it: the Node side of its one Client folder.
func transmissionFolder(t *testing.T, a *worldtest.API) string {
	t.Helper()
	var list struct {
		Cards []struct {
			Externals []struct {
				ClientFolders []struct {
					Node string `json:"node"`
				} `json:"clientFolders"`
			} `json:"externals"`
		} `json:"cards"`
	}
	decode(t, "listDownloadClients", a.Admin("home", http.MethodGet, "/api/v1/download-clients", ""), &list)
	if len(list.Cards) != 1 || len(list.Cards[0].Externals) != 1 || len(list.Cards[0].Externals[0].ClientFolders) != 1 {
		t.Fatalf("listDownloadClients = %+v, want home's card with Transmission and its one Client folder", list.Cards)
	}
	return list.Cards[0].Externals[0].ClientFolders[0].Node
}

// expectLinkedFrom checks placed is the file named name under folder, hard
// linked rather than copied.
func expectLinkedFrom(t *testing.T, folder, name, placed string) {
	t.Helper()
	var downloaded os.FileInfo
	err := filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.Name() == name {
			downloaded, err = os.Stat(path)
		}
		return err //nolint:wrapcheck // the test reports it as is
	})
	if err != nil || downloaded == nil {
		t.Fatalf("%s holds no %s: %v", folder, name, err)
	}
	linked, err := os.Stat(placed)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(downloaded, linked) {
		t.Fatalf("%s isn't %s hard linked: the import copied it", placed, name)
	}
}
