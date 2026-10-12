package world

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

const (
	manifestFile        = "manifest.json"
	stoppedManifestFile = "manifest.stopped.json"
	manifestSchema      = 1
)

// Manifest describes a ready world. It exists only while the world is
// ready: the launch writes it as its last step and stop renames it.
type Manifest struct {
	Schema      int         `json:"schema"`
	RunID       string      `json:"runId"`
	RunDir      string      `json:"runDir"`
	ScratchDir  string      `json:"scratchDir"`
	Scenario    ScenarioRef `json:"scenario"`
	Binary      Binary      `json:"binary"`
	Supervisor  Supervisor  `json:"supervisor"`
	CA          CAFile      `json:"ca"`
	Divergences []string    `json:"divergences"`
	Nodes       []Node      `json:"nodes"`
	Fakes       []Fake      `json:"fakes"`
	Admin       *Admin      `json:"admin,omitempty"`
	APIKey      string      `json:"apiKey,omitempty"`
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

// Supervisor is the process that launched the run and stops it through
// its stop channel, Control.
type Supervisor struct {
	PID     int    `json:"pid"`
	Log     string `json:"log"`
	Control string `json:"control"`
}

type CAFile struct {
	File string `json:"file"`
}

type Node struct {
	Name       string            `json:"name"`
	Setup      scenario.Setup    `json:"setup"`
	Claim      Claim             `json:"claim"`
	Address    string            `json:"address"`
	Host       string            `json:"host"`
	URL        string            `json:"url"`
	BrowserURL string            `json:"browserUrl"`
	Listeners  map[string]string `json:"listeners"`
	DataDir    string            `json:"dataDir"`
	PID        int               `json:"pid"`
	Seed       string            `json:"seed"`
	Log        string            `json:"log"`
	Overrides  []node.Key        `json:"overrides"`
}

// Claim is what a Node is once the world has set it up.
type Claim string

const (
	ClaimUnclaimed  Claim = "unclaimed"
	ClaimController Claim = "controller"
)

var claimAfter = [...]Claim{scenario.SetupNone: ClaimUnclaimed, scenario.SetupController: ClaimController}

type Fake struct {
	Name string `json:"name"`
	URL  string `json:"url"`
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
