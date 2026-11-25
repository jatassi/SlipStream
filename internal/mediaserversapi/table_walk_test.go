package mediaserversapi_test

import (
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
)

// walkRig is a Plex server and Den, both shown to Members, and a Member.
type walkRig struct {
	plexRig
	servers map[access.Way]string
	member  mstest.Member
}

func newWalkRig(t *testing.T) walkRig {
	t.Helper()
	r := newPlexRig(t)
	r.signIn(t)
	plexServer := r.addPlex(t, r.w.Home.String()).ID
	r.w.Jellyfin(t, host, den(floorVer))
	denServer := r.add(t)
	r.show(t, plexServer)
	r.show(t, denServer)
	return walkRig{plexRig: r, servers: map[access.Way]string{access.WayPlexTV: plexServer, access.WayUsers: denServer}, member: r.w.Member(t, "sam")}
}

// row is the Member's one row on server, false for none.
func (r walkRig) row(t *testing.T, way access.Way) (accessOut, bool) {
	t.Helper()
	for _, row := range r.accessOf(t, r.member.ID).Rows {
		if row.Server.ID == r.servers[way] {
			return row, true
		}
	}
	return accessOut{}, false
}

// events start each event as its caller does: an Admin through the API,
// identity's Account delete through the Granter.
var events = map[access.Event]func(t *testing.T, r walkRig, way access.Way){
	access.Offer: func(t *testing.T, r walkRig, way access.Way) {
		if status, body := r.offer(t, r.member.ID, r.servers[way], way == access.WayPlexTV); status != http.StatusOK {
			t.Fatalf("offer = %d %s", status, body)
		}
	},
	access.Remove: func(t *testing.T, r walkRig, way access.Way) {
		row, _ := r.row(t, way)
		if status, body := r.o.CallSignedIn(t, http.MethodDelete, servers+"/access/"+row.ID, ""); status != http.StatusOK {
			t.Fatalf("remove = %d %s", status, body)
		}
	},
	access.Leave: func(t *testing.T, r walkRig, _ access.Way) {
		done, err := r.w.Moves().Granter().Leaving(t.Context(), r.member.ID)
		if err != nil {
			t.Fatal(err)
		}
		done()
	},
}

// TestTheTableWalk drives Table entries at the API seam: each case brings
// a row to the entry's From, starts its event and reads the row back in
// the entry's To. An offer asks for Plex Home on Plex, so a Plex row
// offered as a friend comes back as Plex Home.
func TestTheTableWalk(t *testing.T) {
	for _, c := range []struct {
		from access.State
		on   access.Event
		when access.Outcome
		way  access.Way
	}{
		{access.None, access.Offer, access.NoIdentity, access.WayPlexTV},
		{access.None, access.Offer, access.Done, access.WayUsers},
		{access.Offered, access.Offer, access.Done, access.WayPlexTV},
		{access.Offered, access.Remove, access.Done, access.WayPlexTV},
		{access.Offered, access.Remove, access.Done, access.WayUsers},
		{access.Offered, access.Leave, access.Done, access.WayPlexTV},
		{access.Offered, access.Leave, access.Done, access.WayUsers},
	} {
		entry := access.Must(c.from, c.on, c.when, c.way)
		t.Run(string(c.way)+" "+string(c.from)+" "+string(c.on), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newWalkRig(t)
				if c.from == access.Offered {
					if status, body := r.offer(t, r.member.ID, r.servers[c.way], false); status != http.StatusOK {
						t.Fatalf("offer = %d %s", status, body)
					}
				}
				events[c.on](t, r, c.way)
				row, ok := r.row(t, c.way)
				switch {
				case entry.To == access.None && ok:
					t.Fatalf("the row is %s, want it deleted", row.State)
				case entry.To == access.None:
				case !ok || row.State != entry.To:
					t.Fatalf("the row is %+v, want %s", row, entry.To)
				case c.way == access.WayPlexTV && (row.Plex == nil || !row.Plex.PlexHome):
					t.Errorf("the Plex row is %+v, want it offered as Plex Home", row.Plex)
				}
			})
		})
	}
}
