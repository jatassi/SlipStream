package mediaservers

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
)

// ErrMemberAddressOnPlex is an Address for Members on a Plex server, whose
// apps connect through the Member's Plex account.
var ErrMemberAddressOnPlex = errors.New("a Plex server takes no Address for Members")

func (p Patch) members() bool {
	return p.MembersSee != nil || p.MemberLibraries != nil || p.MemberAddress != nil
}

// membersChanged tells the Admins' access lists of a saved patch that
// changed what Members may be offered.
func (m *MediaServers) membersChanged(ctx context.Context, p Patch) {
	if p.members() {
		m.accessChanged(ctx, identity.AccountID{})
	}
}

// saveMembers writes the Members fields of p onto srv, the server as the
// rest of p leaves it. The first Show to Members with no Libraries for
// Members picks the libraries behind the shown Path mappings.
func (m *MediaServers) saveMembers(ctx context.Context, q *db.Queries, srv db.MediaserversServer, p Patch) error {
	if !p.members() {
		return nil
	}
	see, addr := srv.MembersSee, srv.MemberAddress
	if p.MembersSee != nil {
		see = *p.MembersSee
	}
	if p.MemberAddress != nil {
		addr = p.MemberAddress.To
	}
	if addr != nil && m.wayOf(srv.Kind).way() == access.WayPlexTV {
		return ErrMemberAddressOnPlex
	}
	libs, err := q.ListMemberLibraries(ctx, srv.ID)
	if err != nil {
		return err
	}
	picked := choices(libs)
	switch {
	case p.MemberLibraries != nil:
		picked = *p.MemberLibraries
	case see && len(picked) == 0:
		shown, err := q.ListShownLibraries(ctx, srv.ID)
		if err != nil {
			return err
		}
		for _, l := range shown {
			picked = append(picked, LibraryChoice{Key: *l.LibraryKey, Name: *l.LibraryName})
		}
	}
	if see && len(picked) == 0 {
		return ErrNoMemberLibraries
	}
	if err := q.DeleteMemberLibraries(ctx, srv.ID); err != nil {
		return err
	}
	for _, l := range picked {
		if err := q.InsertMemberLibrary(ctx, db.InsertMemberLibraryParams{ServerID: srv.ID, LibraryKey: l.Key, Name: l.Name}); err != nil {
			return err
		}
	}
	return q.SetMembersFields(ctx, db.SetMembersFieldsParams{ID: srv.ID, MembersSee: see, MemberAddress: addr})
}

// serverMembers is each server's Members fields as its view shows them.
type serverMembers struct {
	libraries map[link.ServerID][]LibraryChoice
	// watching is how many Members can watch on each server: their access
	// is active.
	watching map[link.ServerID]int
}

func readServerMembers(ctx context.Context, q *db.Queries) (serverMembers, error) {
	sm := serverMembers{libraries: map[link.ServerID][]LibraryChoice{}, watching: map[link.ServerID]int{}}
	libs, err := q.ListAllMemberLibraries(ctx)
	if err != nil {
		return sm, err
	}
	for _, l := range libs {
		sm.libraries[l.ServerID] = append(sm.libraries[l.ServerID], LibraryChoice{Key: l.LibraryKey, Name: l.Name})
	}
	counts, err := q.CountMembers(ctx)
	for _, c := range counts {
		sm.watching[c.ServerID] = int(c.Members)
	}
	return sm, err
}

func (sm serverMembers) fill(v *ServerView, row db.MediaserversServer) {
	v.MembersSee, v.MemberAddress, v.MemberCount = row.MembersSee, deref(row.MemberAddress), sm.watching[row.ID]
	v.MemberLibraries = append([]LibraryChoice{}, sm.libraries[row.ID]...)
}
