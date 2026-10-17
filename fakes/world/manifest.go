package world

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/world/internal/lifecycle"
	"github.com/jatassi/slipstream/fakes/world/internal/setup"
	"github.com/jatassi/slipstream/web"
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
	Browser  Browser           `json:"browser"`
	Admin    *Admin            `json:"admin,omitempty"`
	// Members are the Scenario's Members, each with its Account's id.
	Members []Member `json:"members"`
	APIKey  string   `json:"apiKey,omitempty"`
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

type NodeState = lifecycle.State

const (
	NodeRunning = lifecycle.Running
	NodeStopped = lifecycle.Stopped
	NodeKilled  = lifecycle.Killed
	NodeExited  = lifecycle.Exited
)

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

// Browser is what a browser driving the run needs: Chromium's host
// resolver rules for every Node's name, the playwright cli config
// fakeworld browser writes for each shell, relative to the run dir, and
// each shell's preset.
type Browser struct {
	HostResolverRules string                `json:"hostResolverRules"`
	Config            map[string]string     `json:"config"`
	Presets           map[string]web.Preset `json:"presets"`
}

type Admin struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	PIN         string `json:"pin"`
}

type Member struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	PIN         string `json:"pin"`
	Admin       bool   `json:"admin"`
}

// ReadManifest and writeAtomically go through os.Root because on Windows it
// opens files sharing delete access and renames with POSIX semantics, so a
// reader holding manifest.json open neither blocks the supervisor replacing
// it nor sees the name missing mid-replace. os.ReadFile and os.Rename do
// neither.
func ReadManifest(runDir string) (Manifest, error) {
	return readManifestFile(runDir, manifestFile)
}

func readManifestFile(runDir, name string) (Manifest, error) {
	root, err := os.OpenRoot(runDir)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	defer func() { _ = root.Close() }()
	data, err := root.ReadFile(name)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest in %s: %w", runDir, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("read manifest %s: %w", filepath.Join(runDir, name), err)
	}
	return m, nil
}

func writeAtomically(path string, data []byte) error {
	dir, name := filepath.Dir(path), filepath.Base(path)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	defer func() { _ = root.Close() }()
	f, err := os.CreateTemp(dir, "."+name+".*")
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
	if err := root.Rename(filepath.Base(f.Name()), name); err != nil {
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
