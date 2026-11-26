package setup

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/internal/node"
)

// Access drives each Member's access to the state the Scenario names, as
// the Admin and the Member would. The Admin offers each server. A Member
// with a Plex row past offered signs in to Plex on plex.tv's page, which
// leaves it waiting; for shared the Member then signs SlipStream out on
// plex.tv, so the Admin's share stays unaccepted, and for active the
// share accepts with the Member's sign-in. A Jellyfin or Emby row past
// offered is the user the Member makes, which the Admin then removes for
// removed. It fails unless every row reads its state at the end.
func Access(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, claims Claims, accounts []Account, w MediaWorld) error {
	if !slices.ContainsFunc(accounts, func(a Account) bool { return len(a.Access) > 0 }) {
		return nil
	}
	controller := controllerURL(s, ready)
	admin, err := nodeapi.SignIn(ctx, c, controller, s.Admin.Email, s.Admin.PIN)
	if err != nil {
		return err
	}
	for _, a := range accounts {
		if len(a.Access) == 0 {
			continue
		}
		if err := memberAccess(ctx, c, s, controller, claims.APIKey, admin, a, w); err != nil {
			return fmt.Errorf("member %s: %w", a.Email, err)
		}
	}
	return nodeapi.SignOut(ctx, c, controller, admin)
}

func memberAccess(ctx context.Context, c *http.Client, s scenario.Scenario, controller, apiKey string, admin *http.Cookie, a Account, w MediaWorld) error {
	offerable, err := nodeapi.ListAccess(ctx, c, controller, apiKey, a.ID)
	if err != nil {
		return err
	}
	rows := make([]string, len(a.Access))
	for i, want := range a.Access {
		at := slices.IndexFunc(offerable.Offerable, func(srv nodeapi.AccessServer) bool { return srv.Name == want.Server })
		if at < 0 {
			return fmt.Errorf("%s isn't offerable", want.Server)
		}
		if rows[i], err = nodeapi.OfferMediaServer(ctx, c, controller, apiKey, a.ID, offerable.Offerable[at].ID); err != nil {
			return err
		}
	}
	member, err := nodeapi.SignIn(ctx, c, controller, a.Email, a.PIN)
	if err != nil {
		return err
	}
	for i, want := range a.Access {
		if want.State == scenario.AccessOffered {
			continue
		}
		srv, _ := s.MediaServerNamed(want.Server)
		if srv.Kind == "plex" {
			err = plexAccess(ctx, c, controller, admin, member, a, rows[i], want.State, w)
		} else {
			err = usersAccess(ctx, c, controller, apiKey, member, a, rows[i], want.State)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", want.Server, err)
		}
	}
	if err := nodeapi.SignOut(ctx, c, controller, member); err != nil {
		return err
	}
	return readsBack(ctx, c, controller, apiKey, a, rows)
}

func plexAccess(ctx context.Context, c *http.Client, controller string, admin, member *http.Cookie, a Account, row string, state scenario.AccessState, w MediaWorld) error {
	if err := memberPlexSignIn(ctx, c, controller, member, a.Plex.Username, w); err != nil {
		return err
	}
	if state == scenario.AccessWaiting {
		return nil
	}
	if state == scenario.AccessShared {
		if err := w.Cue(ctx, plextv.SignOutMember.Name, map[string]string{"account": a.Plex.Username}); err != nil {
			return err
		}
	}
	return nodeapi.ShareAccess(ctx, c, controller, admin, row)
}

// memberPlexSignIn signs the Member in to Plex as the plex.tv account
// username, approving the PIN on plex.tv as that account would, and
// waits on the Member's event stream until SlipStream signed in with it.
func memberPlexSignIn(ctx context.Context, c *http.Client, controller string, member *http.Cookie, username string, w MediaWorld) error {
	pin, err := nodeapi.StartMemberPlexSignIn(ctx, c, controller, member)
	if err != nil {
		return err
	}
	if err := approve(ctx, w, pin.AuthURL, username); err != nil {
		return err
	}
	var last nodeapi.MemberPlexSignIn
	err = nodeapi.AwaitEvent(ctx, c, controller, member, "media-servers.my-access-changed", func() (bool, error) {
		var read error
		last, read = nodeapi.MemberPlexSignInState(ctx, c, controller, member, pin.ID)
		return last.State != "waiting", read
	})
	if err != nil {
		return err
	}
	if last.State != "signed_in" {
		return fmt.Errorf("the Plex sign-in as %s reads %s %s", username, last.State, last.Refusal)
	}
	return nil
}

func usersAccess(ctx context.Context, c *http.Client, controller, apiKey string, member *http.Cookie, a Account, row string, state scenario.AccessState) error {
	if err := nodeapi.MakeMediaServerAccount(ctx, c, controller, member, row, a.Username(), a.PIN); err != nil {
		return err
	}
	if state == scenario.AccessRemoved {
		return nodeapi.RemoveAccess(ctx, c, controller, apiKey, row)
	}
	return nil
}

func readsBack(ctx context.Context, c *http.Client, controller, apiKey string, a Account, rows []string) error {
	got, err := nodeapi.ListAccess(ctx, c, controller, apiKey, a.ID)
	if err != nil {
		return err
	}
	for i, want := range a.Access {
		at := slices.IndexFunc(got.Rows, func(r nodeapi.AccessRow) bool { return r.ID == rows[i] })
		if at < 0 || got.Rows[at].State != string(want.State) {
			return fmt.Errorf("access to %s reads %+v, want %s", want.Server, got.Rows, want.State)
		}
	}
	return nil
}
