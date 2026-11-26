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

// rules are what every move of the Table keeps to.
var rules = []struct {
	broken string
	breaks func(tr access.Transition) bool
}{
	{"rows of its way never hold one of its states", func(tr access.Transition) bool {
		return !holds(tr.Way, tr.From) || !holds(tr.Way, tr.To)
	}},
	{"only MakeAccount and Created are written before their remote call", func(tr access.Transition) bool {
		return tr.To == access.Making && tr.When == access.Done && tr.On != access.MakeAccount && tr.On != access.Created
	}},
	{"a failed move loses its problem", func(tr access.Transition) bool {
		return (tr.When == access.Failed || tr.When == access.Refused) && tr.Mark&access.KeepProblem == 0
	}},
	{"a Leave keeps a row only as removing or making with no Account", func(tr access.Transition) bool {
		kept := tr.To == access.Removing || tr.To == access.Making
		return tr.On == access.Leave && tr.To != access.None && (!kept || tr.Mark&access.ClearAccount == 0)
	}},
}

func holds(way access.Way, s access.State) bool {
	return s == access.None || slices.Contains(stored[way], s)
}

func TestTheTableIsClosed(t *testing.T) {
	type key struct {
		from access.State
		on   access.Event
		when access.Outcome
		way  access.Way
	}
	seen := map[key]bool{}
	for _, tr := range access.Table {
		if k := (key{tr.From, tr.On, tr.When, tr.Way}); seen[k] {
			t.Errorf("%+v: two moves for one state, event, outcome and way", tr)
		} else {
			seen[k] = true
		}
		for _, r := range rules {
			if r.breaks(tr) {
				t.Errorf("%+v: %s", tr, r.broken)
			}
		}
	}
	for way, states := range stored {
		for _, s := range states {
			if !access.Starts(s, access.Leave, way) {
				t.Errorf("a %s row in %q can't leave when its Account is deleted", way, s)
			}
			if !slices.Contains(access.States(), s) {
				t.Errorf("%q isn't a stored state", s)
			}
		}
	}
}

// everyRow is each state of each way with every combination of the other
// facts.
func everyRow() []access.Facts {
	var out []access.Facts
	for _, way := range access.Ways() {
		for _, s := range access.States() {
			for bits := range 16 {
				out = append(out, access.Facts{State: s, Way: way, PlexHome: bits&1 != 0,
					PlexInUse: bits&2 != 0, SharesWork: bits&4 != 0, HomeWorks: bits&8 != 0})
			}
		}
	}
	return out
}

func TestActionsFollowTheTable(t *testing.T) {
	for _, f := range everyRow() {
		for _, by := range []access.By{access.ByAdmin, access.ByMember} {
			for _, on := range access.Actions(f, by) {
				if !slices.ContainsFunc(access.Table, func(tr access.Transition) bool {
					return tr.From == f.State && tr.On == on && tr.Way == f.Way && tr.By&by != 0
				}) {
					t.Errorf("Actions(%+v, %d) offers %s, which the table doesn't start", f, by, on)
				}
			}
		}
	}

	plex := func(s access.State) access.Facts {
		return access.Facts{State: s, Way: access.WayPlexTV, SharesWork: true, HomeWorks: true}
	}
	users := access.Facts{State: access.Offered, Way: access.WayUsers}
	for _, c := range []struct {
		name string
		f    access.Facts
		by   access.By
		want []access.Event
	}{
		{"a waiting Plex Home offer", access.Facts{State: access.Waiting, Way: access.WayPlexTV, PlexHome: true, SharesWork: true, HomeWorks: true}, access.ByAdmin,
			[]access.Event{access.ShareHome, access.ShareFriend, access.MarkAdded, access.Remove}},
		{"a waiting Member", plex(access.Waiting), access.ByMember, []access.Event{access.AnotherPlex}},
		{"a waiting Member whose Plex account is in use", access.Facts{State: access.Waiting, Way: access.WayPlexTV, PlexInUse: true}, access.ByMember, []access.Event{}},
		{"an offered Plex row", plex(access.Offered), access.ByAdmin, []access.Event{access.Remove}},
		{"a shared row to its Member", plex(access.Shared), access.ByMember, []access.Event{access.Check}},
		{"a shared row to the Admin", plex(access.Shared), access.ByAdmin, []access.Event{access.Check, access.Remove}},
		{"an active Plex row", plex(access.Active), access.ByAdmin, []access.Event{access.Remove}},
		{"an offered Jellyfin row to its Member", users, access.ByMember, []access.Event{access.MakeAccount}},
		{"an offered Jellyfin row to the Admin", users, access.ByAdmin, []access.Event{access.Link, access.Remove}},
		{"an account being made", access.Facts{State: access.Making, Way: access.WayUsers}, access.ByAdmin, []access.Event{}},
		{"a removed account", access.Facts{State: access.Removed, Way: access.WayUsers}, access.ByAdmin, []access.Event{access.Offer}},
		{"a removal that failed", access.Facts{State: access.Removing, Way: access.WayUsers}, access.ByAdmin, []access.Event{access.RetryRemove}},
	} {
		if got := access.Actions(c.f, c.by); !slices.Equal(got, c.want) {
			t.Errorf("%s: Actions = %v, want %v", c.name, got, c.want)
		}
	}
}
