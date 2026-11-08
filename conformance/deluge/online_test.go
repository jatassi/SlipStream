//go:build online

package deluge_test

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	conformance "github.com/jatassi/slipstream/conformance/deluge"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// TestDelugeConforms runs the suite against the Deluge web UI at
// TEST_ONLINE_DELUGE_URL, signing in with TEST_ONLINE_DELUGE_PASSWORD,
// connecting to the daemon at TEST_ONLINE_DELUGE_DAEMON (host:port) when
// it is set, else to the only one, and saving to TEST_ONLINE_DELUGE_FOLDER.
func TestDelugeConforms(t *testing.T) {
	address := os.Getenv("TEST_ONLINE_DELUGE_URL")
	if address == "" {
		t.Skip("TEST_ONLINE_DELUGE_URL is not set until the online job runs Deluge's container image: https://github.com/jatassi/SlipStream/issues/155")
	}
	var daemon *clients.Daemon
	if hostport := os.Getenv("TEST_ONLINE_DELUGE_DAEMON"); hostport != "" {
		host, port, err := net.SplitHostPort(hostport)
		if err != nil {
			t.Fatal(err)
		}
		n, err := strconv.Atoi(port)
		if err != nil {
			t.Fatal(err)
		}
		daemon = &clients.Daemon{Host: host, Port: n}
	}
	conformance.Run(t, conformance.Target{
		Address:  address,
		Password: os.Getenv("TEST_ONLINE_DELUGE_PASSWORD"),
		Daemon:   daemon,
		HTTP:     &http.Client{Timeout: time.Minute},
		Folder:   os.Getenv("TEST_ONLINE_DELUGE_FOLDER"),
		Log:      slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
}
