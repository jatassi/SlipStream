package clients_test

import (
	"errors"
	"testing"

	"github.com/jatassi/slipstream/internal/torrent/clients"
)

func TestParseLabelFollowsEachKindsRule(t *testing.T) {
	cases := []struct {
		raw  string
		kind clients.Kind
		want error
	}{
		{"Movies, 4K", clients.QBittorrent, nil},
		{"Movies 4K", clients.RTorrent, nil},
		{"movies-4k", clients.Transmission, nil},
		{"movies,4k", clients.Transmission, clients.ErrLabelComma},
		{"movies_4k-2", clients.Deluge, nil},
		{"Movies", clients.Deluge, clients.ErrLabelWord},
		{"movies 4k", clients.Deluge, clients.ErrLabelWord},
		{"", clients.QBittorrent, clients.ErrLabelEmpty},
		{"", clients.Deluge, clients.ErrLabelEmpty},
		{"movies\n", clients.QBittorrent, clients.ErrLabelControl},
		{"mo\tvies", clients.Transmission, clients.ErrLabelControl},
	}
	for _, c := range cases {
		l, err := clients.ParseLabel(c.raw, c.kind)
		if !errors.Is(err, c.want) {
			t.Errorf("ParseLabel(%q, %s) = %v, want %v", c.raw, c.kind, err, c.want)
			continue
		}
		if err == nil && l.String() != c.raw {
			t.Errorf("ParseLabel(%q, %s) = %q, want it unchanged", c.raw, c.kind, l)
		}
	}
}
