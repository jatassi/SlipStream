package access_test

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/mediaservers/access"
)

// stored is each way's states, as the schema's CHECKs allow them.
var stored = map[access.Way][]access.State{
	access.WayPlexTV: {access.Offered, access.Waiting, access.Shared, access.Active, access.Removing},
	access.WayUsers:  {access.Offered, access.Making, access.Active, access.Removing, access.Removed},
}

func TestTheTableIsClosed(t *testing.T) {
	type key struct {
		from access.State
		on   access.Event
		when access.Outcome
		way  access.Way
	}
	seen := map[key]bool{}
	leaves := map[access.Way][]access.State{}
	for _, tr := range access.Table {
		k := key{tr.From, tr.On, tr.When, tr.Way}
		if seen[k] {
			t.Errorf("%+v: two moves for one state, event, outcome and way", tr)
		}
		seen[k] = true
		for _, s := range []access.State{tr.From, tr.To} {
			if s != access.None && !slices.Contains(stored[tr.Way], s) {
				t.Errorf("%+v: %s rows never hold %q", tr, tr.Way, s)
			}
		}
		if tr.To == access.Making && tr.When == access.Done && tr.On != access.MakeAccount && tr.On != access.Created {
			t.Errorf("%+v: only MakeAccount and Created are written before their remote call", tr)
		}
		if (tr.When == access.Failed || tr.When == access.Refused) && tr.Mark&access.KeepProblem == 0 {
			t.Errorf("%+v: a failed move loses its problem", tr)
		}
		if tr.On == access.Leave {
			leaves[tr.Way] = append(leaves[tr.Way], tr.From)
			if tr.To != access.None && (tr.Mark&access.ClearAccount == 0 || (tr.To != access.Removing && tr.To != access.Making)) {
				t.Errorf("%+v: a Leave keeps a row only as removing or making with no Account", tr)
			}
		}
	}
	for way, states := range stored {
		for _, s := range states {
			if !slices.Contains(leaves[way], s) {
				t.Errorf("a %s row in %q can't leave when its Account is deleted", way, s)
			}
			if !slices.Contains(access.States(), s) {
				t.Errorf("%q isn't a stored state", s)
			}
		}
	}
}

func TestActionsFollowTheTable(t *testing.T) {
	people := []access.By{access.ByAdmin, access.ByMember}
	for _, way := range access.Ways() {
		for _, s := range access.States() {
			for _, by := range people {
				for bits := range 32 {
					f := access.Facts{State: s, Way: way, PlexHome: bits&1 != 0, MakesAccounts: bits&2 != 0,
						PlexInUse: bits&4 != 0, SharesWork: bits&8 != 0, HomeWorks: bits&16 != 0}
					for _, on := range access.Actions(f, by) {
						if !slices.ContainsFunc(access.Table, func(tr access.Transition) bool {
							return tr.From == s && tr.On == on && tr.Way == way && tr.By&by != 0
						}) {
							t.Errorf("Actions(%+v, %d) offers %s, which the table doesn't start", f, by, on)
						}
					}
				}
			}
		}
	}

	plex := func(s access.State) access.Facts {
		return access.Facts{State: s, Way: access.WayPlexTV, SharesWork: true, HomeWorks: true}
	}
	users := access.Facts{State: access.Offered, Way: access.WayUsers, MakesAccounts: true}
	for _, c := range []struct {
		name string
		f    access.Facts
		by   access.By
		want []access.Event
	}{
		{"a waiting friend", plex(access.Waiting), access.ByAdmin, []access.Event{access.ShareFriend, access.ShareHome, access.MarkAdded, access.Remove}},
		{"a waiting Plex Home offer", access.Facts{State: access.Waiting, Way: access.WayPlexTV, PlexHome: true, SharesWork: true, HomeWorks: true}, access.ByAdmin,
			[]access.Event{access.ShareHome, access.ShareFriend, access.MarkAdded, access.Remove}},
		{"no Plex Pass for the Home", access.Facts{State: access.Waiting, Way: access.WayPlexTV, SharesWork: true}, access.ByAdmin,
			[]access.Event{access.ShareFriend, access.MarkAdded, access.Remove}},
		{"shares refused to the Admin's sign-in", access.Facts{State: access.Waiting, Way: access.WayPlexTV}, access.ByAdmin,
			[]access.Event{access.MarkAdded, access.Remove}},
		{"a waiting Member", plex(access.Waiting), access.ByMember, []access.Event{access.AnotherPlex}},
		{"a waiting Member whose Plex account is in use", access.Facts{State: access.Waiting, Way: access.WayPlexTV, PlexInUse: true}, access.ByMember, []access.Event{}},
		{"an offered Plex row", plex(access.Offered), access.ByAdmin, []access.Event{access.Remove}},
		{"a shared row to its Member", plex(access.Shared), access.ByMember, []access.Event{access.Check}},
		{"a shared row to the Admin", plex(access.Shared), access.ByAdmin, []access.Event{access.Check, access.Remove}},
		{"an active Plex row", plex(access.Active), access.ByAdmin, []access.Event{access.Remove}},
		{"an offered Jellyfin row to its Member", users, access.ByMember, []access.Event{access.MakeAccount}},
		{"an offered row of a kind with linked users only", access.Facts{State: access.Offered, Way: access.WayUsers}, access.ByMember, []access.Event{}},
		{"an offered Jellyfin row to the Admin", users, access.ByAdmin, []access.Event{access.Link, access.Remove}},
		{"an account being made", access.Facts{State: access.Making, Way: access.WayUsers, MakesAccounts: true}, access.ByAdmin, []access.Event{}},
		{"a removed account", access.Facts{State: access.Removed, Way: access.WayUsers}, access.ByAdmin, []access.Event{access.Offer}},
		{"a removal that failed", access.Facts{State: access.Removing, Way: access.WayUsers}, access.ByAdmin, []access.Event{access.RetryRemove}},
	} {
		if got := access.Actions(c.f, c.by); !slices.Equal(got, c.want) {
			t.Errorf("%s: Actions = %v, want %v", c.name, got, c.want)
		}
	}
}
