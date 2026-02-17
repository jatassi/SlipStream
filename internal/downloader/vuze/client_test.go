package vuze

import (
	"testing"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

func TestClient_Type(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{Host: "localhost", Port: 9091})
	if client.Type() != types.ClientTypeVuze {
		t.Errorf("expected type %s, got %s", types.ClientTypeVuze, client.Type())
	}
}

func TestClient_Protocol(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{Host: "localhost", Port: 9091})
	if client.Protocol() != types.ProtocolTorrent {
		t.Errorf("expected protocol %s, got %s", types.ProtocolTorrent, client.Protocol())
	}
}
