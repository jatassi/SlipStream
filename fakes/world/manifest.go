package world

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/world/internal/setup"
)

const (
	manifestFile        = "manifest.json"
	stoppedManifestFile = "manifest.stopped.json"
	manifestSchema      = 2
)

// Manifest describes a ready world. It exists only while the world is
// ready: the launch writes it as its last step and stop renames it.
type Manifest struct {
	Schema     int    `json:"schema"`
	RunID      string `json:"runId"`
	RunDir     string `json:"runDir"`
	ScratchDir string `json:"scratchDir"`
	// MediaDir holds the library roots' folders, beside the Nodes' data
	// dirs; absent for a Scenario without a library.
	MediaDir   string      `json:"mediaDir,omitempty"`
	Scenario   ScenarioRef `json:"scenario"`
	Binary     Binary      `json:"binary"`
	Supervisor Supervisor  `json:"supervisor"`
	CA         CAFile      `json:"ca"`
	// Events is the event log's path relative to the run dir.
	Events      string   `json:"events"`
	Divergences []string `json:"divergences"`
	Nodes       []Node   `json:"nodes"`
	Fakes       []Fake   `json:"fakes"`
	// Fronts is every front: where each Node, and the world, reaches each
	// target.
	Fronts []Front `json:"fronts"`
	// Links maps each Node to its link front, the address it advertises
	// and every peer dials.
	Links    map[string]string `json:"links"`
	Releases []Release         `json:"releases"`
	Admin    *Admin            `json:"admin,omitempty"`
	APIKey   string            `json:"apiKey,omitempty"`
}

type ScenarioRef struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Seed   string `json:"seed"`
}

// Binary is the run's copy of the binary under test and where it came
// from.
type Binary struct {
	Source  string `json:"source"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Version string `json:"version"`
}

// Supervisor is the process that launched the run and serves its control
// socket, Control.
type Supervisor struct {
	PID     int    `json:"pid"`
	Log     string `json:"log"`
	Control string `json:"control"`
}

type CAFile struct {
	File string `json:"file"`
}

// Node is one Node's manifest entry, built where both worlds share it.
type Node = setup.Node

// Claim is what a Node is once the world has set it up.
type Claim = setup.Claim

const (
	ClaimUnclaimed  = setup.ClaimUnclaimed
	ClaimController = setup.ClaimController
	ClaimWorker     = setup.ClaimWorker
)

// Fake is a fake the world hosts. URL is its world front.
type Fake struct {
	Name     string             `json:"name"`
	URL      string             `json:"url"`
	Captures []fake.CaptureSpec `json:"captures"`
	Cues     []fake.CueSpec     `json:"cues"`
}

// Front is one front: target as Node reaches it, Node "world" for the
// harness and "" for a link front, which every peer dials. URL is absent
// for a target that speaks no HTTP.
type Front struct {
	Target  string `json:"target"`
	Node    string `json:"node"`
	Address string `json:"address"`
	URL     string `json:"url,omitempty"`
}

// Release is a catalog release: its payload's size and BLAKE3, and its
// torrent's info hash, all in hex.
type Release struct {
	Name     string `json:"name"`
	InfoHash string `json:"infoHash"`
	Size     int64  `json:"size"`
	BLAKE3   string `json:"blake3"`
}

type Admin struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	PIN         string `json:"pin"`
}

func ReadManifest(runDir string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(runDir, manifestFile))
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("read manifest %s: %w", filepath.Join(runDir, manifestFile), err)
	}
	return m, nil
}

func writeAtomically(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	_, werr := f.Write(data)
	serr := f.Sync()
	cerr := f.Close()
	if err := firstErr(werr, serr, cerr); err != nil {
		_ = os.Remove(f.Name())
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		_ = os.Remove(f.Name())
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func marshal(v any) []byte {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic("world: " + err.Error())
	}
	return append(data, '\n')
}
