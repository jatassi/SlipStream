package native

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

const (
	InputsPath = "native/inputs.json"
	RecipeDir  = "native/recipe"
	LockPath   = "native/lock.json"
)

// ReleasesURL is where the bundle workflow publishes each layer as the
// asset of its own release.
const ReleasesURL = "https://github.com/jatassi/slipstream-native/releases/download"

const PublishHint = "dispatch the bundle workflow at this branch " +
	"(gh workflow run layers.yml -R jatassi/slipstream-native -f ref=<branch-or-sha>, " +
	"or https://github.com/jatassi/slipstream-native/actions/workflows/layers.yml), " +
	"then run scripts/native-deps pin"

// Kind names what a layer holds.
type Kind string

const (
	KindToolchain Kind = "toolchain"
	KindLibs      Kind = "libs"
)

// LayerID names a layer by what builds it.
type LayerID struct {
	Kind   Kind   `json:"kind"`
	Target string `json:"target"`
	Digest string `json:"digest"`
}

func (id LayerID) Tag() string { return "layer-" + id.Digest }

func (id LayerID) AssetName() string {
	return string(id.Kind) + "-" + id.Target + "-" + id.Digest + ".tar.gz"
}

func (id LayerID) URL(base string) string { return base + "/" + id.Tag() + "/" + id.AssetName() }

// Layer is a published layer as the lock pins it.
type Layer struct {
	LayerID
	Asset  string `json:"asset"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// ToolArchive is one tool's published archive for one target.
type ToolArchive struct {
	Tool   string `json:"tool"`
	Target string `json:"target"`
	URL    string `json:"url"`
}

// PinnedTool is a tool archive as the lock pins it.
type PinnedTool struct {
	ToolArchive
	SHA256 string `json:"sha256"`
}

// Lock pins every layer the inputs build to its published asset, and every
// tool archive the inputs list to its content. Only `release deps pin`
// writes it.
type Lock struct {
	Layers []Layer      `json:"layers"`
	Tools  []PinnedTool `json:"tools"`
}

// ToolArchives lists every tool's archive for every target, in input order.
func (in Inputs) ToolArchives() []ToolArchive {
	var archives []ToolArchive
	for _, tool := range in.Tools {
		for _, t := range in.Targets {
			archives = append(archives, ToolArchive{tool.Name, t.Name, tool.Archives[t.Name]})
		}
	}
	return archives
}

// Layers names the layers t ships by the digest of the current inputs and
// recipe, in build order: its toolchain layer when its toolchain has one,
// then its libs layer.
func (in Inputs) Layers(t Target, recipe fs.FS) ([]LayerID, error) {
	tc, err := in.Toolchain(t)
	if err != nil {
		return nil, err
	}
	var ids []LayerID
	if tc.Layered() {
		digest, err := ToolchainDigest(in, t, recipe)
		if err != nil {
			return nil, err
		}
		ids = append(ids, LayerID{KindToolchain, t.Name, digest})
	}
	digest, err := LibsDigest(in, t, recipe)
	if err != nil {
		return nil, err
	}
	return append(ids, LayerID{KindLibs, t.Name, digest}), nil
}

// Want lists the layers of every target in the table.
func Want(in Inputs, recipe fs.FS) ([]LayerID, error) {
	var ids []LayerID
	for _, t := range in.Targets {
		layers, err := in.Layers(t, recipe)
		if err != nil {
			return nil, err
		}
		ids = append(ids, layers...)
	}
	return ids, nil
}

func WantRepo(root string) (Inputs, []LayerID, error) {
	in, err := LoadInputs(filepath.Join(root, InputsPath))
	if err != nil {
		return Inputs{}, nil, err
	}
	ids, err := Want(in, os.DirFS(filepath.Join(root, RecipeDir)))
	return in, ids, err
}

func LoadLock(path string) (Lock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Lock{}, fmt.Errorf("read the native lock: %w", err)
	}
	var l Lock
	if err := json.Unmarshal(data, &l); err != nil {
		return Lock{}, fmt.Errorf("%s: %w", path, err)
	}
	return l, nil
}

func (l Lock) Write(path string) error {
	data, err := json.MarshalIndent(l, "", "\t")
	if err != nil {
		return fmt.Errorf("encode the native lock: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write the native lock: %w", err)
	}
	return nil
}

func (l Lock) Pinned(id LayerID) (Layer, error) {
	for _, layer := range l.Layers {
		if layer.LayerID == id {
			return layer, nil
		}
	}
	return Layer{}, fmt.Errorf("%s pins no %s layer %s for %s; %s", LockPath, id.Kind, id.Digest, id.Target, PublishHint)
}

func (l Lock) PinnedTool(a ToolArchive) (PinnedTool, error) {
	for _, p := range l.Tools {
		if p.ToolArchive == a {
			return p, nil
		}
	}
	return PinnedTool{}, fmt.Errorf("%s pins no %s archive %s for %s; run scripts/native-deps pin", LockPath, a.Tool, a.URL, a.Target)
}

// StaleTools describes each way the lock's tool archives differ from want.
func (l Lock) StaleTools(want []ToolArchive) []string {
	type slot struct{ tool, target string }
	pinned := map[slot]string{}
	for _, p := range l.Tools {
		pinned[slot{p.Tool, p.Target}] = p.URL
	}
	var stale []string
	for _, a := range want {
		s := slot{a.Tool, a.Target}
		switch url, ok := pinned[s]; {
		case !ok:
			stale = append(stale, fmt.Sprintf("pins no %s archive for %s, inputs list %s", a.Tool, a.Target, a.URL))
		case url != a.URL:
			stale = append(stale, fmt.Sprintf("pins %s archive %s for %s, inputs list %s", a.Tool, url, a.Target, a.URL))
		}
		delete(pinned, s)
	}
	for s, url := range pinned {
		stale = append(stale, fmt.Sprintf("pins %s archive %s for %s, which %s does not list", s.tool, url, s.target, InputsPath))
	}
	slices.Sort(stale)
	return stale
}

func (l Lock) Stale(want []LayerID) []string {
	type slot struct {
		kind   Kind
		target string
	}
	pinned := map[slot]string{}
	for _, layer := range l.Layers {
		pinned[slot{layer.Kind, layer.Target}] = layer.Digest
	}
	var stale []string
	for _, id := range want {
		s := slot{id.Kind, id.Target}
		switch digest, ok := pinned[s]; {
		case !ok:
			stale = append(stale, fmt.Sprintf("pins no %s layer for %s, inputs and recipe build %s", id.Kind, id.Target, id.Digest))
		case digest != id.Digest:
			stale = append(stale, fmt.Sprintf("pins %s layer %s for %s, inputs and recipe build %s", id.Kind, digest, id.Target, id.Digest))
		}
		delete(pinned, s)
	}
	for s, digest := range pinned {
		stale = append(stale, fmt.Sprintf("pins %s layer %s for %s, which %s does not list", s.kind, digest, s.target, InputsPath))
	}
	slices.Sort(stale)
	return stale
}

// StaleLock describes each way the lock in the checkout at root differs from
// the layers its inputs and recipe build and the tool archives its inputs
// list, each with how to bring the lock up to date.
func StaleLock(root string) ([]string, error) {
	in, want, err := WantRepo(root)
	if err != nil {
		return nil, err
	}
	l, err := LoadLock(filepath.Join(root, LockPath))
	if err != nil {
		return nil, err
	}
	var stale []string
	for _, msg := range l.Stale(want) {
		stale = append(stale, msg+"; "+PublishHint)
	}
	for _, msg := range l.StaleTools(in.ToolArchives()) {
		stale = append(stale, msg+"; run scripts/native-deps pin")
	}
	return stale, nil
}
