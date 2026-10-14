package world

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// Bundle is a run dir read back to replay it: a live or stopped run, or
// one CI uploaded.
type Bundle struct {
	Dir      string
	Manifest Manifest
	// Scenario is the bundle's scenario.yaml, which hashes to the
	// manifest's.
	Scenario scenario.Scenario
	// Binary is the binary the run ran, checked against the manifest's
	// SHA-256.
	Binary string
	Origin Origin
	Steps  []JournaledStep
}

// ReadBundle reads the run dir dir and checks that its scenario.yaml and
// the binary, binary or else the bundle's own copy, are the ones its
// manifest names.
func ReadBundle(dir, binary string) (Bundle, error) {
	b := Bundle{Dir: dir}
	var err error
	if b.Manifest, err = ReadLastManifest(dir); err != nil {
		return Bundle{}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "scenario.yaml"))
	if err != nil {
		return Bundle{}, fmt.Errorf("read the bundle's Scenario: %w", err)
	}
	if b.Scenario, err = scenario.Parse(b.Manifest.Scenario.Name, data); err != nil {
		return Bundle{}, err
	}
	if sum := b.Scenario.Hash(); sum != b.Manifest.Scenario.SHA256 {
		return Bundle{}, fmt.Errorf("scenario.yaml hashes to %s, not the manifest's %s", sum, b.Manifest.Scenario.SHA256)
	}
	if b.Binary = binary; b.Binary == "" {
		b.Binary = filepath.Join(dir, "bin", filepath.Base(b.Manifest.Binary.Path))
	}
	sum, err := fileSHA256(b.Binary)
	if err != nil {
		return Bundle{}, fmt.Errorf("%w; pass the binary the run ran with --binary", err)
	}
	if sum != b.Manifest.Binary.SHA256 {
		return Bundle{}, fmt.Errorf("%s hashes to %s, not the manifest's %s", b.Binary, sum, b.Manifest.Binary.SHA256)
	}
	recs, err := events.Read(dir)
	if err != nil {
		return Bundle{}, err
	}
	var l launched
	if len(recs) == 0 || recs[0].Kind != events.RunLaunched || json.Unmarshal(recs[0].Data, &l) != nil {
		return Bundle{}, fmt.Errorf("%s does not start with a %s that names its origin", filepath.Join(dir, events.File), events.RunLaunched)
	}
	b.Origin = l.Origin
	if b.Steps, err = readSteps(recs); err != nil {
		return Bundle{}, err
	}
	return b, nil
}
