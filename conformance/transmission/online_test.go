//go:build online

package transmission_test

import (
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/transmission"
)

// TestTransmissionConforms runs the suite against the Transmission at
// TEST_ONLINE_TRANSMISSION_URL, signing in with
// TEST_ONLINE_TRANSMISSION_USERNAME and TEST_ONLINE_TRANSMISSION_PASSWORD
// and saving to TEST_ONLINE_TRANSMISSION_FOLDER.
func TestTransmissionConforms(t *testing.T) {
	address := os.Getenv("TEST_ONLINE_TRANSMISSION_URL")
	if address == "" {
		t.Skip("TEST_ONLINE_TRANSMISSION_URL is not set until the online job runs Transmission's container image: https://github.com/jatassi/SlipStream/issues/155")
	}
	conformance.Run(t, conformance.Target{
		Address:  address,
		Username: os.Getenv("TEST_ONLINE_TRANSMISSION_USERNAME"),
		Password: os.Getenv("TEST_ONLINE_TRANSMISSION_PASSWORD"),
		HTTP:     &http.Client{Timeout: time.Minute},
		Folder:   os.Getenv("TEST_ONLINE_TRANSMISSION_FOLDER"),
		Log:      slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
}
