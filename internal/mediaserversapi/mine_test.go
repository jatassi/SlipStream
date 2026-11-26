package mediaserversapi_test

import (
	"net/http"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
)

type myAccessOut struct {
	Rows []struct {
		ID      string         `json:"id"`
		State   access.State   `json:"state"`
		Pending bool           `json:"pending"`
		Actions []access.Event `json:"actions"`
	} `json:"rows"`
}

// TestEachMemberReadsOnlyTheirOwnAccess offers Den to Sam and not to Kit:
// Sam reads the offer with what to do next, Kit reads nothing, and the
// Admin can still offer Den to Kit only.
func TestEachMemberReadsOnlyTheirOwnAccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		r.w.Jellyfin(t, host, den(floorVer))
		server := r.add(t)
		r.show(t, server)
		sam, kit := r.w.Member(t, "sam"), r.w.Member(t, "kit")
		if status, body := r.offer(t, sam.ID, server, false); status != http.StatusOK {
			t.Fatalf("offer Den to Sam = %d %s", status, body)
		}

		mine := func(m mstest.Member) myAccessOut {
			status, body := r.o.CallAs(t, http.MethodGet, "/api/v1/account/media-servers", "", m.Session)
			if status != http.StatusOK {
				t.Fatalf("list my access = %d %s", status, body)
			}
			return decode[myAccessOut](t, body)
		}
		got := mine(sam)
		if len(got.Rows) != 1 || got.Rows[0].State != access.Offered || !got.Rows[0].Pending ||
			!slices.Equal(got.Rows[0].Actions, []access.Event{access.MakeAccount}) {
			t.Errorf("Sam reads %+v, want Den offered with makeAccount", got.Rows)
		}
		if got := mine(kit); len(got.Rows) != 0 {
			t.Errorf("Kit reads %+v, want nothing", got.Rows)
		}
		for _, c := range []struct {
			who       mstest.Member
			rows      int
			offerable int
		}{{sam, 1, 0}, {kit, 0, 1}} {
			l := r.accessOf(t, c.who.ID)
			if len(l.Rows) != c.rows || len(l.Offerable) != c.offerable {
				t.Errorf("the Admin lists %+v for %s, want %d rows and %d servers to offer", l, c.who.ID, c.rows, c.offerable)
			}
		}
	})
}
