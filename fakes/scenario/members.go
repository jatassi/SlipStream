package scenario

import (
	"slices"
	"strings"
)

// PlexAccount is a plex.tv account fake-plex-tv holds.
type PlexAccount struct {
	Username string `jsonschema:"required" yaml:"username"`
	Email    string `jsonschema:"required" yaml:"email"`
}

// AccessState is where the world leaves a Member's access to a server.
type AccessState string

const (
	// AccessOffered is offered by the Admin, nothing done since.
	AccessOffered AccessState = "offered"
	// AccessWaiting is a Plex server the Member signed in to Plex for,
	// waiting on the Admin's share.
	AccessWaiting AccessState = "waiting"
	// AccessShared is shared on Plex and not accepted: the Member signed
	// SlipStream out on plex.tv before the share, so it can't accept it.
	AccessShared AccessState = "shared"
	// AccessActive is a share the Member's sign-in accepted, or a Jellyfin
	// or Emby user the Member made.
	AccessActive AccessState = "active"
	// AccessRemoved is a Jellyfin or Emby user the Admin removed, which
	// the server keeps turned off.
	AccessRemoved AccessState = "removed"
)

// AccessStates are the states a Scenario's access takes.
var AccessStates = []AccessState{AccessOffered, AccessWaiting, AccessShared, AccessActive, AccessRemoved}

// MemberAccess is a Member's access to one Media Server, by its name. A
// Jellyfin or Emby user the Member makes is named the local part of their
// email, with their PIN as its password.
type MemberAccess struct {
	Server string      `jsonschema:"required" yaml:"server"`
	State  AccessState `jsonschema:"required,enum=offered,enum=waiting,enum=shared,enum=active,enum=removed" yaml:"state"`
}

// plexStates are the states only Plex takes, and usersStates those only
// Jellyfin and Emby take.
var (
	plexStates  = []AccessState{AccessWaiting, AccessShared}
	usersStates = []AccessState{AccessRemoved}
)

// Username is the user a Member makes on a Jellyfin or Emby server.
func (m Member) Username() string {
	name, _, _ := strings.Cut(m.Email, "@")
	return name
}

func (s Scenario) checkAccess(add func(format string, args ...any)) {
	usernames := map[string]bool{}
	plexAccount := func(a PlexAccount) {
		name := strings.ToLower(a.Username)
		if usernames[name] {
			add("plex account %s is listed twice", a.Username)
		}
		usernames[name] = true
	}
	for _, a := range s.PlexAccounts {
		plexAccount(a)
	}
	for _, m := range s.Members {
		if m.Plex != nil {
			plexAccount(*m.Plex)
		}
		for i, a := range m.Access {
			s.checkMemberAccess(add, m, a)
			if slices.ContainsFunc(m.Access[:i], func(o MemberAccess) bool { return o.Server == a.Server }) {
				add("member %s lists access to %s twice", m.Email, a.Server)
			}
		}
	}
}

func (s Scenario) checkMemberAccess(add func(format string, args ...any), m Member, a MemberAccess) {
	if !slices.Contains(AccessStates, a.State) {
		add("member %s access to %s is %q, not one of %v", m.Email, a.Server, a.State, AccessStates)
		return
	}
	srv, ok := s.MediaServerNamed(a.Server)
	if !ok {
		add("member %s has access to %s, which no node reaches", m.Email, a.Server)
		return
	}
	if !srv.ShowToMembers {
		add("member %s has access to %s, which isn't shown to members", m.Email, a.Server)
	}
	plex := srv.Kind == "plex"
	switch {
	case plex && slices.Contains(usersStates, a.State):
		add("member %s access to %s is %s, which only Jellyfin and Emby take", m.Email, a.Server, a.State)
	case !plex && slices.Contains(plexStates, a.State):
		add("member %s access to %s is %s, which only Plex takes", m.Email, a.Server, a.State)
	case plex && a.State != AccessOffered && m.Plex == nil:
		add("member %s access to %s is %s, so plex is required", m.Email, a.Server, a.State)
	}
}

// MediaServerNamed is the Scenario's server named name, false when no
// Node reaches one.
func (s Scenario) MediaServerNamed(name string) (MediaServer, bool) {
	for _, n := range s.Nodes {
		if i := slices.IndexFunc(n.MediaServers, func(m MediaServer) bool { return m.Name == name }); i >= 0 {
			return n.MediaServers[i], true
		}
	}
	return MediaServer{}, false
}
