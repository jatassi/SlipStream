package scenario

import (
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// DownloadClient is an external Download Client the world adds to its
// Node through the API, as an Admin would, at that Node's front for the
// fake of its kind. Its faults are fault rules on that front, such as a
// refuse for a client that isn't running, and the fake's cues.
type DownloadClient struct {
	Kind string `jsonschema:"required,enum=qbittorrent,enum=transmission,enum=deluge,enum=rtorrent" yaml:"kind"`
	// Version is a version the kind's fake plays, such as 4.1.
	Version string `jsonschema:"required" yaml:"version"`
	// Name is the client's name in SlipStream, the kind's name when empty.
	Name string `yaml:"name,omitempty"`
	// Username and Password are the client's sign-in, which the world
	// types; both empty is a client with sign-in off.
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	// Label is the client's label setting, slipstream when empty.
	Label string `yaml:"label,omitempty"`
	// Folder is the client's own name for its download folder, such as
	// /downloads or D:\Torrents; empty is the folder as its Node sees it.
	Folder string `yaml:"folder,omitempty"`
	// Pair has the world add the Client folder pairing Folder with the
	// folder as its Node sees it.
	Pair        bool          `yaml:"pair,omitempty"`
	ImportDelay time.Duration `yaml:"import_delay,omitempty"`
	// GrabsTo has the world set its Node's New Grabs go to this client.
	GrabsTo bool `yaml:"grabs_to,omitempty"`
}

func (n Node) checkClients(add func(format string, args ...any)) {
	if len(n.DownloadClients) > 0 && !n.downloads() {
		add("node %s has download clients, but doesn't hold downloading", n.Name)
	}
	grabsTo := 0
	for i, c := range n.DownloadClients {
		switch {
		case !slices.Contains(clients.Kinds, clients.Kind(c.Kind)):
			add("node %s download client kind %q is not one of %v", n.Name, c.Kind, clients.Kinds)
		case slices.ContainsFunc(n.DownloadClients[:i], func(o DownloadClient) bool { return o.Kind == c.Kind }):
			add("node %s has two %s download clients; the world adds at most one of each kind", n.Name, c.Kind)
		}
		if c.Username != "" && clients.Kind(c.Kind).Facts().SignIn.Username == clients.NotTaken {
			add("node %s %s download client has a username, which %s doesn't take", n.Name, c.Kind, clients.Kind(c.Kind).Facts().Name)
		}
		if c.Version == "" {
			add("node %s %s download client names no version", n.Name, c.Kind)
		}
		if c.Pair && c.Folder == "" {
			add("node %s %s download client pairs its folder, but names none", n.Name, c.Kind)
		}
		if c.ImportDelay < 0 {
			add("node %s %s download client import_delay %s is negative", n.Name, c.Kind, c.ImportDelay)
		}
		if c.GrabsTo {
			grabsTo++
		}
	}
	if grabsTo > 1 {
		add("node %s sends its New Grabs to %d download clients; at most one", n.Name, grabsTo)
	}
}

// downloads reports whether n holds Downloading once set up: a join or
// controller Node listing it, and a controller Node that lists no Roles.
func (n Node) downloads() bool {
	return (n.Setup == SetupController && len(n.Roles) == 0) || (n.Setup != SetupNone && slices.Contains(n.Roles, role.Downloading))
}
