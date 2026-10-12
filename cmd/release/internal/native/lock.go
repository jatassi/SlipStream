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

// Kind names what a layer holds. A target row lists the kinds it ships.
type Kind string

const KindLibs Kind = "libs"

var layerDigests = map[Kind]func(Inputs, Target, fs.FS) (string, error){
	KindLibs: LibsDigest,
}

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

// Lock pins every layer the inputs build to its published asset. Only
// `release deps pin` writes it.
type Lock struct {
	Layers []Layer `json:"layers"`
}

// Want lists the layers the target table asks for, named by the digest of
// the current inputs and recipe.
func Want(in Inputs, recipe fs.FS) ([]LayerID, error) {
	var ids []LayerID
	for _, t := range in.Targets {
		for _, k := range t.Layers {
			digest, err := layerDigests[k](in, t, recipe)
			if err != nil {
				return nil, err
			}
			ids = append(ids, LayerID{k, t.Name, digest})
		}
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
		return Lock{}, err
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
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func (l Lock) Pinned(id LayerID) (Layer, error) {
	for _, layer := range l.Layers {
		if layer.LayerID == id {
			return layer, nil
		}
	}
	return Layer{}, fmt.Errorf("%s pins no %s layer %s for %s; %s", LockPath, id.Kind, id.Digest, id.Target, PublishHint)
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
// the layers its inputs and recipe build.
func StaleLock(root string) ([]string, error) {
	_, want, err := WantRepo(root)
	if err != nil {
		return nil, err
	}
	l, err := LoadLock(filepath.Join(root, LockPath))
	if err != nil {
		return nil, err
	}
	return l.Stale(want), nil
}
