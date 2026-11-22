//go:build online

package qbittorrent_test

import (
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jatassi/slipstream/conformance/internal/torrentfile"
	conformance "github.com/jatassi/slipstream/conformance/qbittorrent"
)

// TestQBittorrentConforms runs the suite against the qBittorrent at
// TEST_ONLINE_QBITTORRENT_URL, signing in with TEST_ONLINE_QBITTORRENT_API_KEY
// when it is set, else with TEST_ONLINE_QBITTORRENT_USERNAME and
// TEST_ONLINE_QBITTORRENT_PASSWORD, and saving to
// TEST_ONLINE_QBITTORRENT_FOLDER, which this machine sees at
// TEST_ONLINE_QBITTORRENT_NODE_FOLDER.
func TestQBittorrentConforms(t *testing.T) {
	address := os.Getenv("TEST_ONLINE_QBITTORRENT_URL")
	if address == "" {
		t.Skip("TEST_ONLINE_QBITTORRENT_URL is not set until the online job runs qBittorrent's container image: https://github.com/jatassi/SlipStream/issues/155")
	}
	seeded, interval := torrentfile.Online(t, os.Getenv("TEST_ONLINE_TRACKER_URL"), os.Getenv("TEST_ONLINE_TRACKER_INTERVAL"))
	conformance.Run(t, conformance.Target{
		Address:    address,
		Username:   os.Getenv("TEST_ONLINE_QBITTORRENT_USERNAME"),
		Password:   os.Getenv("TEST_ONLINE_QBITTORRENT_PASSWORD"),
		APIKey:     os.Getenv("TEST_ONLINE_QBITTORRENT_API_KEY"),
		HTTP:       &http.Client{Timeout: time.Minute},
		Folder:     os.Getenv("TEST_ONLINE_QBITTORRENT_FOLDER"),
		NodeFolder: os.Getenv("TEST_ONLINE_QBITTORRENT_NODE_FOLDER"),
		Seeded:     seeded,
		Interval:   interval,
		Log:        slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
}
