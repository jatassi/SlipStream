package clients_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/torrent/clients"
)

func TestEveryKindHasFactsTheFormCanUse(t *testing.T) {
	floors := map[clients.Kind]string{
		clients.QBittorrent: "4.6", clients.Transmission: "4.0", clients.Deluge: "2.1", clients.RTorrent: "0.9.8",
	}
	for _, kind := range clients.Kinds {
		f := kind.Facts()
		if f.Name == "" {
			t.Fatalf("%s has no facts", kind)
		}
		if got := f.Floor.String(); got != floors[kind] {
			t.Errorf("%s's floor = %s, want %s", f.Name, got, floors[kind])
		}
		if len(f.Acts) != 4 {
			t.Errorf("%s supports %d actions, want pause, resume, recheck and announce", f.Name, len(f.Acts))
		}
		if _, err := clients.ParseLabel(clients.DefaultLabel, kind); err != nil {
			t.Errorf("%s refuses the default label: %v", f.Name, err)
		}
	}
}

func TestOnlyRTorrentLeavesDataToTheNode(t *testing.T) {
	for _, kind := range clients.Kinds {
		if got, want := kind.Facts().DeletesData, kind != clients.RTorrent; got != want {
			t.Errorf("%s DeletesData = %v, want %v", kind, got, want)
		}
	}
}

func TestOnlyQBittorrentAndTransmissionStopPollingAtARefusedSignIn(t *testing.T) {
	for _, kind := range clients.Kinds {
		want := kind == clients.QBittorrent || kind == clients.Transmission
		if got := kind.Facts().Lockout; got != want {
			t.Errorf("%s Lockout = %v, want %v", kind, got, want)
		}
	}
}
