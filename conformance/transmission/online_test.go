//go:build online

package transmission_test

import (
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/torrentfile"
	conformance "github.com/jatassi/slipstream/conformance/transmission"
)

// TestTransmissionConforms runs the suite against the Transmission at
// TEST_ONLINE_TRANSMISSION_URL, signing in with
// TEST_ONLINE_TRANSMISSION_USERNAME and TEST_ONLINE_TRANSMISSION_PASSWORD
// and saving to TEST_ONLINE_TRANSMISSION_FOLDER, which this machine sees at
// TEST_ONLINE_TRANSMISSION_NODE_FOLDER.
func TestTransmissionConforms(t *testing.T) {
	address := os.Getenv("TEST_ONLINE_TRANSMISSION_URL")
	if address == "" {
		t.Skip("TEST_ONLINE_TRANSMISSION_URL is not set until the online job runs Transmission's container image: https://github.com/jatassi/SlipStream/issues/697")
	}
	seeded, interval := torrentfile.Online(t, os.Getenv("TEST_ONLINE_TRACKER_URL"), os.Getenv("TEST_ONLINE_TRACKER_INTERVAL"))
	conformance.Run(t, conformance.Target{
		Address:    address,
		Username:   os.Getenv("TEST_ONLINE_TRANSMISSION_USERNAME"),
		Password:   os.Getenv("TEST_ONLINE_TRANSMISSION_PASSWORD"),
		HTTP:       &http.Client{Timeout: time.Minute},
		Folder:     os.Getenv("TEST_ONLINE_TRANSMISSION_FOLDER"),
		NodeFolder: os.Getenv("TEST_ONLINE_TRANSMISSION_NODE_FOLDER"),
		Seeded:     seeded,
		Interval:   interval,
		Log:        slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
}
