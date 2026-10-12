package native

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"path"
)

type recipeFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ToolchainDigest names a toolchain layer by everything that goes into
// building it: the target and its triple, the toolchain's sources, and the
// path and content of every file under the toolchain's recipe directory.
// Two inputs that hash alike build the same layer, so an existing layer
// with this digest is never rebuilt.
func ToolchainDigest(in Inputs, t Target, recipe fs.FS) (string, error) {
	tc, err := in.Toolchain(t)
	if err != nil {
		return "", err
	}
	files, err := recipeFiles(recipe, path.Join(string(KindToolchain), tc.Name))
	if err != nil {
		return "", err
	}
	return hashDoc(struct {
		Layer   Kind         `json:"layer"`
		Target  string       `json:"target"`
		Triple  string       `json:"triple"`
		Sources []Source     `json:"sources"`
		Recipe  []recipeFile `json:"recipe"`
	}{KindToolchain, t.Name, t.Triple, tc.Sources, files})
}

// LibsDigest names a libs layer by the target row, every component, every
// file under the recipe's libs directory and the toolchain that compiles
// it: the digest of its toolchain layer, or the pinned Xcode version. A
// toolchain bump rebuilds the libs.
func LibsDigest(in Inputs, t Target, recipe fs.FS) (string, error) {
	tc, err := in.Toolchain(t)
	if err != nil {
		return "", err
	}
	var toolchain string
	if tc.Layered() {
		if toolchain, err = ToolchainDigest(in, t, recipe); err != nil {
			return "", err
		}
	}
	files, err := recipeFiles(recipe, string(KindLibs))
	if err != nil {
		return "", err
	}
	return hashDoc(struct {
		Layer      Kind         `json:"layer"`
		Target     Target       `json:"target"`
		Components []Component  `json:"components"`
		Recipe     []recipeFile `json:"recipe"`
		Toolchain  string       `json:"toolchain,omitempty"`
		Xcode      string       `json:"xcode,omitempty"`
	}{KindLibs, t, in.Components, files, toolchain, tc.Xcode})
}

func recipeFiles(recipe fs.FS, dir string) ([]recipeFile, error) {
	sub, err := fs.Sub(recipe, dir)
	if err != nil {
		return nil, err
	}
	var files []recipeFile
	err = fs.WalkDir(sub, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(sub, name)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		files = append(files, recipeFile{name, hex.EncodeToString(sum[:])})
		return nil
	})
	return files, err
}

func hashDoc(doc any) (string, error) {
	data, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
