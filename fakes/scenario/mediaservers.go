package scenario

import (
	"slices"

	"github.com/jatassi/slipstream/internal/role"
)

// MediaServer is a Plex, Jellyfin or Emby the world adds through the API
// as an Admin would, reached from the Node that lists it. Its fake plays
// one library per Content Type of the roots it holds, every folder under
// the server's own media folder: /media, or D:\Media on Windows. The world
// types the reaching Node's front as its address and the Controller's
// front as its fallback, then saves the Path mappings the server's
// libraries pre-fill.
type MediaServer struct {
	Kind string `jsonschema:"required,enum=plex,enum=jellyfin,enum=emby" yaml:"kind"`
	// Version is a version the kind's fake plays, its newest when empty.
	Version string `yaml:"version,omitempty"`
	Name    string `jsonschema:"required" yaml:"name"`
	// Windows names the server's folders as a Windows server does.
	Windows bool `yaml:"windows,omitempty"`
	// SignIn is how the world gets Jellyfin's or Emby's key: key types the
	// key the fake holds, and admin signs in as its administrator so
	// SlipStream creates one. Plex always signs in with the plex.tv PIN,
	// which the world approves on fake-plex-tv.
	SignIn string `jsonschema:"enum=key,enum=admin" yaml:"sign_in,omitempty"`
	// NoFallback leaves the fallback address empty, so the Controller never
	// reaches the server itself.
	NoFallback bool `yaml:"no_fallback,omitempty"`
	// Roots are the library roots the server holds, by folder; empty is
	// every root on the reaching Node. The world saves each other root as
	// not shown.
	Roots []MediaServerRoot `yaml:"roots,omitempty"`
	// Users are Jellyfin or Emby users beside the administrator, each with
	// watch data of their own.
	Users []string `yaml:"users,omitempty"`
}

// MediaServerRoot is one library root a server holds.
type MediaServerRoot struct {
	Folder string `jsonschema:"required" yaml:"folder"`
	// Hidden saves the root's mapping as not shown, so the server gets no
	// changes for it.
	Hidden bool `yaml:"hidden,omitempty"`
}

// MediaServerKinds are the kinds a Scenario's server names.
var MediaServerKinds = []string{"plex", "jellyfin", "emby"}

// MediaServerAt is a Scenario's server with the Node that reaches it.
type MediaServerAt struct {
	Node string
	MediaServer
}

// MediaServer is the Scenario's server of kind, false when it names none.
func (s Scenario) MediaServer(kind string) (MediaServerAt, bool) {
	for _, n := range s.Nodes {
		for _, m := range n.MediaServers {
			if m.Kind == kind {
				return MediaServerAt{Node: n.Name, MediaServer: m}, true
			}
		}
	}
	return MediaServerAt{}, false
}

// HeldRoots are the library roots m holds: those it names, or every root
// on its reaching Node.
func (s Scenario) HeldRoots(m MediaServerAt) []LibraryRoot {
	var out []LibraryRoot
	for _, r := range s.Library.Roots {
		named := slices.ContainsFunc(m.Roots, func(h MediaServerRoot) bool { return h.Folder == r.Folder })
		if named || (len(m.Roots) == 0 && s.RootNode(r) == m.Node) {
			out = append(out, r)
		}
	}
	return out
}

// mediaLink is whether n holds Media Server link once set up: the
// Controller always does.
func (n Node) mediaLink() bool {
	return n.Setup == SetupController || (n.Setup == SetupJoin && slices.Contains(n.Roles, role.MediaServerLink))
}

func (s Scenario) checkMediaServers(add func(format string, args ...any)) {
	var kinds []string
	for _, n := range s.Nodes {
		if len(n.MediaServers) > 0 && !n.mediaLink() {
			add("node %s reaches media servers, but doesn't hold media-server-link", n.Name)
		}
		for _, m := range n.MediaServers {
			if !slices.Contains(MediaServerKinds, m.Kind) {
				add("node %s media server kind %q is not one of %v", n.Name, m.Kind, MediaServerKinds)
				continue
			}
			if slices.Contains(kinds, m.Kind) {
				add("node %s reaches a second %s; the world plays at most one of each kind", n.Name, m.Kind)
			}
			kinds = append(kinds, m.Kind)
			if s.Library == nil {
				add("node %s reaches %s, and no library adds it", n.Name, m.Name)
				continue
			}
			s.checkMediaServer(add, MediaServerAt{Node: n.Name, MediaServer: m})
		}
	}
}

func (s Scenario) checkMediaServer(add func(format string, args ...any), m MediaServerAt) {
	if m.Kind == "plex" && m.SignIn != "" {
		add("media server %s signs in with %s, but Plex signs in only with the plex.tv PIN", m.Name, m.SignIn)
	}
	if m.Kind == "plex" && len(m.Users) > 0 {
		add("media server %s lists users, which only Jellyfin and Emby take", m.Name)
	}
	for _, h := range m.Roots {
		if !slices.ContainsFunc(s.Library.Roots, func(r LibraryRoot) bool { return r.Folder == h.Folder }) {
			add("media server %s holds root %s, which the library doesn't name", m.Name, h.Folder)
		}
	}
	if len(s.HeldRoots(m)) == 0 {
		add("media server %s holds no library root", m.Name)
	}
}
