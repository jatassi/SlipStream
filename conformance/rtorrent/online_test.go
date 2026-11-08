//go:build online

package rtorrent_test

import (
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/rtorrent"
)

// TestRTorrentConforms runs the suite against the rTorrent behind nginx
// at TEST_ONLINE_RTORRENT_URL, signing in with TEST_ONLINE_RTORRENT_USERNAME
// and TEST_ONLINE_RTORRENT_PASSWORD and saving to
// TEST_ONLINE_RTORRENT_FOLDER.
func TestRTorrentConforms(t *testing.T) {
	address := os.Getenv("TEST_ONLINE_RTORRENT_URL")
	if address == "" {
		t.Skip("TEST_ONLINE_RTORRENT_URL is not set until the online job runs rTorrent's container image: https://github.com/jatassi/SlipStream/issues/155")
	}
	conformance.Run(t, conformance.Target{
		Address:  address,
		Username: os.Getenv("TEST_ONLINE_RTORRENT_USERNAME"),
		Password: os.Getenv("TEST_ONLINE_RTORRENT_PASSWORD"),
		HTTP:     &http.Client{Timeout: time.Minute},
		Folder:   os.Getenv("TEST_ONLINE_RTORRENT_FOLDER"),
		Log:      slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
}
