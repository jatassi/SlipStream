//go:build online

package qbittorrent_test

import (
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/qbittorrent"
)

// TestQBittorrentConforms runs the suite against the qBittorrent at
// TEST_ONLINE_QBITTORRENT_URL, signing in with TEST_ONLINE_QBITTORRENT_API_KEY
// when it is set, else with TEST_ONLINE_QBITTORRENT_USERNAME and
// TEST_ONLINE_QBITTORRENT_PASSWORD, and saving to
// TEST_ONLINE_QBITTORRENT_FOLDER.
func TestQBittorrentConforms(t *testing.T) {
	address := os.Getenv("TEST_ONLINE_QBITTORRENT_URL")
	if address == "" {
		t.Skip("TEST_ONLINE_QBITTORRENT_URL is not set until the online job runs qBittorrent's container image: https://github.com/jatassi/SlipStream/issues/155")
	}
	conformance.Run(t, conformance.Target{
		Address:  address,
		Username: os.Getenv("TEST_ONLINE_QBITTORRENT_USERNAME"),
		Password: os.Getenv("TEST_ONLINE_QBITTORRENT_PASSWORD"),
		APIKey:   os.Getenv("TEST_ONLINE_QBITTORRENT_API_KEY"),
		HTTP:     &http.Client{Timeout: time.Minute},
		Folder:   os.Getenv("TEST_ONLINE_QBITTORRENT_FOLDER"),
		Log:      slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
}
