//go:build online

package jellyfin_test

import (
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/jellyfin"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// TestJellyfinConforms runs the suite against the Jellyfin at
// TEST_ONLINE_JELLYFIN_URL with the API key TEST_ONLINE_JELLYFIN_KEY, in the
// library named TEST_ONLINE_JELLYFIN_LIBRARY whose folder
// TEST_ONLINE_JELLYFIN_FOLDER is the local folder TEST_ONLINE_JELLYFIN_DISK, each
// video a copy of TEST_ONLINE_JELLYFIN_SEED. TEST_ONLINE_JELLYFIN_ADMIN and
// TEST_ONLINE_JELLYFIN_PASSWORD, when set, sign in as an administrator. The
// run counts as green on the throwaway containers on Jackson's Mac (ruling
// J2), oldest and newest supported version each.
func TestJellyfinConforms(t *testing.T) {
	address := os.Getenv("TEST_ONLINE_JELLYFIN_URL")
	if address == "" {
		t.Skip("TEST_ONLINE_JELLYFIN_URL is not set; the online Jellyfin run is a manual run on throwaway containers: https://github.com/jatassi/SlipStream/issues/160")
	}
	folder, err := link.ParsePath(os.Getenv("TEST_ONLINE_JELLYFIN_FOLDER"))
	if err != nil {
		t.Fatal(err)
	}
	tg := conformance.Target{
		Address: address,
		Key:     os.Getenv("TEST_ONLINE_JELLYFIN_KEY"),
		HTTP:    &http.Client{Timeout: time.Minute},
		Library: os.Getenv("TEST_ONLINE_JELLYFIN_LIBRARY"),
		Folder:  folder,
		Disk:    conformance.DirDisk{Dir: os.Getenv("TEST_ONLINE_JELLYFIN_DISK"), Seed: os.Getenv("TEST_ONLINE_JELLYFIN_SEED")},
		Settle:  2 * time.Minute,
		Bound:   5 * time.Minute,
		Log:     slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
	if user := os.Getenv("TEST_ONLINE_JELLYFIN_ADMIN"); user != "" {
		tg.Admin = &conformance.Admin{User: user, Password: os.Getenv("TEST_ONLINE_JELLYFIN_PASSWORD")}
	}
	conformance.Run(t, tg)
}
