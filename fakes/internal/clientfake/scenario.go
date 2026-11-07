package clientfake

import (
	"cmp"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// Rate is how many bytes a second each torrent of a Scenario's client
// moves.
const Rate = 1 << 20

// Folder is the folder the client of the Node named node downloads into,
// as that Node sees it, under the fake's Dir.
func Folder(dir, node string) string { return filepath.Join(dir, node) }

// Played is one Node's client of a kind, as a Scenario names it.
type Played struct {
	Node string
	scenario.DownloadClient
	// Dir is the folder the client downloads into as its Node sees it, and
	// Folder the client's own name for it, Dir when the Scenario names none.
	Dir, Folder string
	// Windows is a Folder that starts with a drive letter.
	Windows bool
}

// Playing is the client of kind on each Node the Scenario gives one, each
// downloading into that Node's folder under dir.
func Playing(s scenario.Scenario, kind, dir string) []Played {
	var out []Played
	for _, n := range s.Nodes {
		for _, c := range n.DownloadClients {
			if c.Kind != kind {
				continue
			}
			at := Folder(dir, n.Name)
			folder := cmp.Or(c.Folder, at)
			out = append(out, Played{Node: n.Name, DownloadClient: c, Dir: at, Folder: folder, Windows: len(folder) >= 2 && folder[1] == ':'})
		}
	}
	return out
}

// Entered reads the addresses an Admin typed for the external clients of
// kind, each with the Node that dials it, from listDownloadClients.
func Entered(kind string) func(get func(path string) ([]byte, error)) ([]fake.Entered, error) {
	return func(get func(path string) ([]byte, error)) ([]fake.Entered, error) {
		body, err := get("/api/v1/download-clients")
		if err != nil {
			return nil, err
		}
		var list struct {
			Cards []struct {
				NodeName  string `json:"nodeName"`
				Externals []struct {
					NodeID  string `json:"nodeId"`
					Kind    string `json:"kind"`
					Name    string `json:"name"`
					Address string `json:"address"`
				} `json:"externals"`
			} `json:"cards"`
		}
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("listDownloadClients: %w", err)
		}
		var out []fake.Entered
		for _, card := range list.Cards {
			for _, e := range card.Externals {
				if e.Kind == kind {
					out = append(out, fake.Entered{What: e.Name + " on " + card.NodeName, URL: e.Address, NodeID: e.NodeID})
				}
			}
		}
		return out, nil
	}
}
