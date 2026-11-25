package mstest

import (
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/internal/mediaservers"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestOnlyTheReachingNodeServesAServersMembers saves Den, which home
// reaches: home lists Den's users, and seedbox, a Worker whose intent
// doesn't hold Den, refuses.
func TestOnlyTheReachingNodeServesAServersMembers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := New(t)
		const host = "192.168.1.20:8096"
		w.Jellyfin(t, host, jellyfin.Server{
			Name: "Den", Version: "12.2.0", ID: "ddc0b5b7f2824180aae9f2f245dd00ca", Keys: []string{"den-key"},
			Users: []jellyfin.User{{Name: "ada", Password: "ada-password", Admin: true}},
			Root:  "/media", Libraries: []jellyfin.Library{{Name: "Shelves", Type: "homevideos", Folders: []string{"/media/Shelves"}}},
		})
		v, err := w.Service.Add(t.Context(), mediaservers.Draft{Kind: "jellyfin", Name: "Den", Address: "http://" + host, Reaching: w.Home, APIKey: "den-key"})
		if err != nil {
			t.Fatal(err)
		}
		id, err := storage.ParseUUID(v.ID)
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		list := &msv1.MembersRequest{ServerId: id.Bytes(), Op: &msv1.MembersRequest_ListUsers{ListUsers: &msv1.ListUsers{}}}

		res, err := w.links.service(w.Home).Members(t.Context(), list)
		if users := res.GetUsers(); err != nil || len(users) != 1 || users[0].GetName() != "ada" {
			t.Errorf("home listed Den's users as %v, %v; want ada", users, err)
		}
		if _, err := w.links.service(w.Seedbox).Members(t.Context(), list); rpc.CodeOf(err) != rpc.CodeFailedPrecondition {
			t.Errorf("seedbox listed Den's users: %v, want FailedPrecondition", err)
		}
	})
}
