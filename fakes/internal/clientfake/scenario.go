package clientfake

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/jatassi/slipstream/fakes/fake"
)

// Rate is how many bytes a second each torrent of a Scenario's client
// moves.
const Rate = 1 << 20

// Folder is the folder the client of the Node named node downloads into,
// as that Node sees it, under the fake's Dir.
func Folder(dir, node string) string { return filepath.Join(dir, node) }

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
