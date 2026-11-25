package scenario

import (
	"net/url"
	"slices"

	"github.com/jatassi/slipstream/internal/role"
)

// MediaServer is a Plex, Jellyfin or Emby the world adds through the API
// as an Admin would, reached from the Node that lists it. Its fake plays
// one library per Content Type of the roots it holds, unless a root names
// a library of its own, every folder under
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
	// RunsOn is the Node the server runs on, which the world saves as its
	// Runs on; empty saves none.
	RunsOn string `yaml:"runs_on,omitempty"`
	// Remote is Plex's Remote Access or Jellyfin's remote connections.
	Remote bool `yaml:"remote,omitempty"`
	// Custom are Plex's custom server access URLs, which plex.tv also lists
	// as connections.
	Custom []string `yaml:"custom,omitempty"`
	// Published are the addresses the server publishes beside each Node's
	// front: Plex's plex.tv connections, Jellyfin's published server
	// addresses, Emby's WAN and remote addresses. A Plex URL here or in
	// Custom with no port names the port the server listens on, since the
	// world draws every port.
	Published []string `yaml:"published,omitempty"`
	// Mapping is where Plex's own check of its public address left it.
	Mapping string `jsonschema:"enum=mapped,enum=failed,enum=pending" yaml:"mapping,omitempty"`
}

// MediaServerRoot is one library root a server holds.
type MediaServerRoot struct {
	Folder string `jsonschema:"required" yaml:"folder"`
	// Hidden saves the root's mapping as not shown, so the server gets no
	// changes for it.
	Hidden bool `yaml:"hidden,omitempty"`
	// Library is the name of the library or section that holds the root,
	// beside the one its Content Type's other roots share. Roots naming
	// one library share it.
	Library string `yaml:"library,omitempty"`
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
	s.checkPublishing(add, m)
	types := map[string]string{}
	for _, h := range m.Roots {
		i := slices.IndexFunc(s.Library.Roots, func(r LibraryRoot) bool { return r.Folder == h.Folder })
		if i < 0 {
			add("media server %s holds root %s, which the library doesn't name", m.Name, h.Folder)
			continue
		}
		if h.Library == "" {
			continue
		}
		if typ, ok := types[h.Library]; ok && typ != s.Library.Roots[i].ContentType {
			add("media server %s puts roots of %s and %s in library %s; a library holds one Content Type", m.Name, typ, s.Library.Roots[i].ContentType, h.Library)
		}
		types[h.Library] = s.Library.Roots[i].ContentType
	}
	if len(s.HeldRoots(m)) == 0 {
		add("media server %s holds no library root", m.Name)
	}
}

// checkPublishing checks where m runs and what it publishes.
func (s Scenario) checkPublishing(add func(format string, args ...any), m MediaServerAt) {
	if m.Kind != "plex" && (len(m.Custom) > 0 || m.Mapping != "") {
		add("media server %s sets custom or mapping, which only Plex takes", m.Name)
	}
	if m.RunsOn != "" && !slices.ContainsFunc(s.Nodes, func(n Node) bool { return n.Name == m.RunsOn }) {
		add("media server %s runs on %s, which isn't a Node", m.Name, m.RunsOn)
	}
	for _, u := range slices.Concat(m.Custom, m.Published) {
		if p, err := url.Parse(u); err != nil || (p.Scheme != "http" && p.Scheme != "https") || p.Hostname() == "" {
			add("media server %s publishes %q, which isn't an http or https URL with a host", m.Name, u)
		}
	}
}
