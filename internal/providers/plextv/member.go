package plextv

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/providers/plex"
)

// Member is plex.tv as one Member's device: their client identifier and
// no key, so it makes only legacy PINs and calls only a Member's routes.
type Member struct{ t transport }

func NewMember(e Endpoints, h plex.Headers, c *http.Client) *Member {
	return &Member{t: transport{e: e, h: h, http: c}}
}

// CreateLegacyPin is a new strong PIN with no key, whose approval gives a
// legacy token.
func (m *Member) CreateLegacyPin(ctx context.Context) (Pin, error) {
	var out pinBody
	err := m.t.do(ctx, request{route: routes[legacyPin]}, &out)
	return Pin{ID: out.ID, Code: out.Code, ExpiresAt: out.ExpiresAt}, err
}

// AuthURL is the page where the Member approves code, which opens forward
// again with no query string.
func (m *Member) AuthURL(code, forward string) string { return m.t.authURL(code, forward) }

// PollLegacyPin is the Member's token once they approved the PIN, and
// false while it waits. A PIN past its life is an error Expired reads.
func (m *Member) PollLegacyPin(ctx context.Context, id int64) (MemberToken, bool, error) {
	var out pinBody
	if err := m.t.do(ctx, request{route: routes[pollLegacyPin], vars: pinVars(id)}, &out); err != nil {
		return MemberToken{}, false, err
	}
	if out.AuthToken == nil {
		return MemberToken{}, false, nil
	}
	return MemberToken{s: *out.AuthToken}, true, nil
}

func (m *Member) User(ctx context.Context, t MemberToken) (User, error) { return m.t.user(ctx, t.s) }

// SignOut ends the token, and only it: the Member's other Plex sign-ins
// keep working. A token plex.tv no longer knows is signed out already.
func (m *Member) SignOut(ctx context.Context, t MemberToken) error {
	err := m.t.do(ctx, request{route: routes[signOut], token: t.s}, nil)
	if status(err) == http.StatusNotFound {
		return nil
	}
	return err
}

// Accept finds share among the Member's pending invites by its id and
// accepts it. False when it isn't pending: the Admin's FindShare says
// whether it is accepted already.
func (m *Member) Accept(ctx context.Context, t MemberToken, share ShareID) (bool, error) {
	var out []struct {
		SharedServers []struct {
			ID wireID `json:"id"`
		} `json:"sharedServers"`
	}
	if err := m.t.do(ctx, request{route: routes[pendingReceived], token: t.s}, &out); err != nil {
		return false, err
	}
	for _, inv := range out {
		for _, s := range inv.SharedServers {
			if ShareID(s.ID) == share {
				return true, m.t.do(ctx, request{route: routes[accept], vars: []string{"{share}", string(share)}, token: t.s}, nil)
			}
		}
	}
	return false, nil
}
