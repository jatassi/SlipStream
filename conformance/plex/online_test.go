//go:build online

package plex_test

import (
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/plex"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// TestPlexConforms runs the suite against the Plex at
// TEST_ONLINE_PLEX_URL with its server token TEST_ONLINE_PLEX_TOKEN, in
// the section TEST_ONLINE_PLEX_SECTION whose Location
// TEST_ONLINE_PLEX_FOLDER is the local folder TEST_ONLINE_PLEX_DISK, each
// video a copy of TEST_ONLINE_PLEX_SEED. Plex Pass, the plex.tv sign-in and
// Windows paths are recorded manual runs under recorded/ until then.
func TestPlexConforms(t *testing.T) {
	address := os.Getenv("TEST_ONLINE_PLEX_URL")
	if address == "" {
		t.Skip("TEST_ONLINE_PLEX_URL is not set; the online Plex run is a recorded manual run on Jackson's server: https://github.com/jatassi/SlipStream/issues/160")
	}
	folder, err := link.ParsePath(os.Getenv("TEST_ONLINE_PLEX_FOLDER"))
	if err != nil {
		t.Fatal(err)
	}
	conformance.Run(t, conformance.Target{
		Address: address,
		Token:   os.Getenv("TEST_ONLINE_PLEX_TOKEN"),
		HTTP:    &http.Client{Timeout: time.Minute},
		Section: os.Getenv("TEST_ONLINE_PLEX_SECTION"),
		Folder:  folder,
		Disk:    conformance.DirDisk{Dir: os.Getenv("TEST_ONLINE_PLEX_DISK"), Seed: os.Getenv("TEST_ONLINE_PLEX_SEED")},
		Settle:  5 * time.Second,
		Bound:   2 * time.Minute,
		Log:     slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
}
