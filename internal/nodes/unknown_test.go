package nodes_test

import (
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// TestUnknownNodesAreBoundedByCountAndAge hears four unknown Nodes with
// room for three: the oldest drops, a Node heard again moves to the front,
// and one not heard for longer than UnknownFor leaves the list.
func TestUnknownNodesAreBoundedByCountAndAge(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		timings := nodes.DefaultTimings()
		timings.UnknownNodes, timings.UnknownFor = 3, time.Hour
		svc, err := nodes.Open(t.Context(), nodes.Config{
			Library: storagetest.Library(t, nodes.Schema(), role.LibrarySchema()), Key: key(9), Name: "home", Link: unlinked{},
			Owners: nodestest.Unowned(), Timings: timings, Log: discard,
		})
		if err != nil {
			t.Fatal(err)
		}
		hear := func(b byte) {
			svc.Unknown(key(b), link.NodeInfo{OS: "linux", Arch: "amd64"}, netip.MustParseAddrPort(fmt.Sprintf("203.0.113.%d:8485", b)))
			time.Sleep(time.Minute)
		}
		heard := func() string {
			var out []string
			for _, u := range svc.Unknowns() {
				out = append(out, u.Address.Addr().String())
			}
			return strings.Join(out, " ")
		}
		for _, b := range []byte{1, 2, 3, 4} {
			hear(b)
		}
		if got, want := heard(), "203.0.113.4 203.0.113.3 203.0.113.2"; got != want {
			t.Errorf("after four unknown Nodes with room for three, Unknowns = %q, want %q", got, want)
		}
		hear(2)
		if got, want := heard(), "203.0.113.2 203.0.113.4 203.0.113.3"; got != want {
			t.Errorf("after 203.0.113.2 is heard again, Unknowns = %q, want %q", got, want)
		}
		time.Sleep(time.Hour - 2*time.Minute)
		if got, want := heard(), "203.0.113.2"; got != want {
			t.Errorf("an hour on, Unknowns = %q, want only the Node heard last, %q", got, want)
		}
	})
}

func key(b byte) link.Key {
	var k link.Key
	for i := range k {
		k[i] = b
	}
	return k
}
