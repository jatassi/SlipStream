package sim_test

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
)

var updateHashes = flag.Bool("update", false, "rewrite testdata/info_hashes.txt from the catalog")

const infoHashesFile = "testdata/info_hashes.txt"

// TestEveryCatalogReleaseKeepsItsInfoHash pins the walk's info hash of
// every catalog release, so a change to how releases become torrents
// changes no torrent a test or a recorded walk already names.
func TestEveryCatalogReleaseKeepsItsInfoHash(t *testing.T) {
	var got bytes.Buffer
	for _, name := range scenario.Catalog() {
		s, err := scenario.Load(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range s.Releases {
			_, _, hash, err := sim.Torrent(s, r)
			if err != nil {
				t.Fatalf("scenario %s release %s: %v", name, r.Name, err)
			}
			fmt.Fprintf(&got, "%s %s %s\n", name, r.Name, hash)
		}
	}
	if *updateHashes {
		if err := os.WriteFile(infoHashesFile, got.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(infoHashesFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("the catalog's info hashes changed:\n%s\nwant:\n%s", got.Bytes(), want)
	}
}
