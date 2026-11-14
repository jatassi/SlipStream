//go:build online

package emby_test

import (
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/emby"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// TestEmbyConforms runs the suite against the Emby at
// TEST_ONLINE_EMBY_URL with the API key TEST_ONLINE_EMBY_KEY, in the
// library named TEST_ONLINE_EMBY_LIBRARY whose folder
// TEST_ONLINE_EMBY_FOLDER is the local folder TEST_ONLINE_EMBY_DISK, each
// video a copy of TEST_ONLINE_EMBY_SEED. The run counts as green on the throwaway containers on Jackson's Mac (ruling
// J2), oldest and newest supported version each.
func TestEmbyConforms(t *testing.T) {
	address := os.Getenv("TEST_ONLINE_EMBY_URL")
	if address == "" {
		t.Skip("TEST_ONLINE_EMBY_URL is not set; the online Emby run is a manual run on throwaway containers: https://github.com/jatassi/SlipStream/issues/160")
	}
	folder, err := link.ParsePath(os.Getenv("TEST_ONLINE_EMBY_FOLDER"))
	if err != nil {
		t.Fatal(err)
	}
	tg := conformance.Target{
		Address: address,
		Key:     os.Getenv("TEST_ONLINE_EMBY_KEY"),
		HTTP:    &http.Client{Timeout: time.Minute},
		Library: os.Getenv("TEST_ONLINE_EMBY_LIBRARY"),
		Folder:  folder,
		Disk:    conformance.DirDisk{Dir: os.Getenv("TEST_ONLINE_EMBY_DISK"), Seed: os.Getenv("TEST_ONLINE_EMBY_SEED")},
		Settle:  2 * time.Minute,
		Bound:   5 * time.Minute,
		Log:     slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
	conformance.Run(t, tg)
}
