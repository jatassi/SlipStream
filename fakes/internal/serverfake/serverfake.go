// Package serverfake is what fake-plex, fake-jellyfin and fake-emby share
// with the world: the server a Scenario names for a kind, its folders as
// the server names them, the key and administrator the world signs in
// with, and the addresses an Admin typed for it.
package serverfake

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/storing"
)

// Library is one library the Scenario's server plays: its Content Type's
// folders as the server names them. Name is the library the Scenario
// names, "" for the one the Content Type's roots share.
type Library struct {
	ContentType string
	Name        string
	Folders     []string
}

// Played is the Scenario's server of a kind, with its media folder and
// libraries as the server names them.
type Played struct {
	scenario.MediaServerAt
	Version   string
	Root      string
	Libraries []Library
	Streamed  []Streamed
}

// Playing is the Scenario's server of kind, false when it names none.
// versions are the fake's, newest last.
func Playing(s scenario.Scenario, kind string, versions []string) (Played, bool, error) {
	m, ok := s.MediaServer(kind)
	if !ok {
		return Played{}, false, nil
	}
	p := Played{MediaServerAt: m, Version: m.Version, Root: "/media"}
	if m.Windows {
		p.Root = `D:\Media`
	}
	if p.Version == "" {
		p.Version = versions[len(versions)-1]
	}
	if !slices.Contains(versions, p.Version) {
		return Played{}, false, fmt.Errorf("%s %s plays version %s; fake-%s plays %v", kind, m.Name, p.Version, kind, versions)
	}
	for _, r := range s.HeldRoots(m) {
		name := ""
		if h := slices.IndexFunc(m.Roots, func(h scenario.MediaServerRoot) bool { return h.Folder == r.Folder }); h >= 0 {
			name = m.Roots[h].Library
		}
		i := slices.IndexFunc(p.Libraries, func(l Library) bool { return l.ContentType == r.ContentType && l.Name == name })
		if i < 0 {
			p.Libraries = append(p.Libraries, Library{ContentType: r.ContentType, Name: name})
			i = len(p.Libraries) - 1
		}
		p.Libraries[i].Folders = append(p.Libraries[i].Folders, p.Folder(r.Folder))
	}
	for _, f := range unrooted(s, m) {
		p.Libraries[0].Folders = append(p.Libraries[0].Folders, p.Folder(f))
	}
	files, err := streamed(s, p)
	if err != nil {
		return Played{}, false, err
	}
	p.Streamed = files
	return p, true, nil
}

// unrooted are the folders of m's played files that no root names, in
// the order m plays them.
func unrooted(s scenario.Scenario, m scenario.MediaServerAt) []string {
	var out []string
	for _, pl := range m.Plays {
		folder, _, _ := strings.Cut(pl.File, "/")
		if !slices.ContainsFunc(s.Library.Roots, func(r scenario.LibraryRoot) bool { return r.Folder == folder }) && !slices.Contains(out, folder) {
			out = append(out, folder)
		}
	}
	return out
}

// Path is the file of library.folders at file, a folder/path, as the
// server names it.
func (p Played) Path(file string) (link.ServerPath, error) {
	folder, rel, _ := strings.Cut(file, "/")
	at, err := link.ParsePath(p.Folder(folder))
	if err != nil {
		return link.ServerPath{}, err
	}
	r, err := storing.ParseRel(rel)
	if err != nil {
		return link.ServerPath{}, err
	}
	return at.Join(r), nil
}

// Folder is the library root folder under the media dir as the server
// names it.
func (p Played) Folder(folder string) string {
	if p.Windows {
		return p.Root + `\` + folder
	}
	return p.Root + "/" + folder
}

// ID is n hex characters drawn from the server's name, as a server's
// identifier reads.
func (p Played) ID(n int) string {
	sum := sha256.Sum256([]byte(p.Kind + " " + p.Name))
	return hex.EncodeToString(sum[:])[:n]
}

// Key is the API key a Jellyfin or Emby the world plays holds, which the
// world types when the server signs in with its key.
func (p Played) Key() string {
	sum := sha256.Sum256([]byte("key " + p.Kind + " " + p.Name))
	return hex.EncodeToString(sum[:16])
}

// The administrator of a Jellyfin or Emby the world plays, whom the
// world signs in as when the server signs in as admin.
const (
	Admin         = "admin"
	AdminPassword = "fake-world-password"
)

// Entered reads the addresses an Admin typed for the servers of kind from
// listMediaServers: each address as its reaching Node dials it, and each
// fallback address as the Controller does.
func Entered(kind string) func(get func(path string) ([]byte, error)) ([]fake.Entered, error) {
	return func(get func(path string) ([]byte, error)) ([]fake.Entered, error) {
		body, err := get("/api/v1/media-servers")
		if err != nil {
			return nil, err
		}
		var list struct {
			Servers []struct {
				Kind            string `json:"kind"`
				Name            string `json:"name"`
				Address         string `json:"address"`
				FallbackAddress string `json:"fallbackAddress"`
				ReachingNode    struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"reachingNode"`
			} `json:"servers"`
		}
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("listMediaServers: %w", err)
		}
		controller, err := controllerID(get)
		if err != nil {
			return nil, err
		}
		var out []fake.Entered
		for _, srv := range list.Servers {
			if srv.Kind != kind {
				continue
			}
			out = append(out, fake.Entered{What: srv.Name + " from " + srv.ReachingNode.Name, URL: srv.Address, NodeID: srv.ReachingNode.ID})
			if srv.FallbackAddress != "" {
				out = append(out, fake.Entered{What: srv.Name + " fallback", URL: srv.FallbackAddress, NodeID: controller})
			}
		}
		return out, nil
	}
}

func controllerID(get func(path string) ([]byte, error)) (string, error) {
	body, err := get("/api/v1/nodes")
	if err != nil {
		return "", err
	}
	var list struct {
		Nodes []struct {
			ID   string `json:"id"`
			Self bool   `json:"self"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return "", fmt.Errorf("listNodes: %w", err)
	}
	for _, n := range list.Nodes {
		if n.Self {
			return n.ID, nil
		}
	}
	return "", nil
}
