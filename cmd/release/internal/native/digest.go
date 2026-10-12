package native

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
)

type recipeFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ToolchainDigest names a toolchain layer by everything that goes into
// building it: the target and its triple, the toolchain's sources, and the
// path and content of every file under the recipe's toolchain directory.
// Two inputs that hash alike build the same layer, so an existing layer
// with this digest is never rebuilt.
func ToolchainDigest(in Inputs, t Target, recipe fs.FS) (string, error) {
	tc, err := in.Toolchain(t)
	if err != nil {
		return "", err
	}
	files, err := recipeFiles(recipe, KindToolchain)
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
// file under the recipe's libs directory and the digest of the toolchain
// layer that compiles it, so a toolchain bump rebuilds the libs.
func LibsDigest(in Inputs, t Target, recipe fs.FS) (string, error) {
	toolchain, err := ToolchainDigest(in, t, recipe)
	if err != nil {
		return "", err
	}
	files, err := recipeFiles(recipe, KindLibs)
	if err != nil {
		return "", err
	}
	return hashDoc(struct {
		Layer      Kind         `json:"layer"`
		Target     Target       `json:"target"`
		Components []Component  `json:"components"`
		Recipe     []recipeFile `json:"recipe"`
		Toolchain  string       `json:"toolchain"`
	}{KindLibs, t, in.Components, files, toolchain})
}

func recipeFiles(recipe fs.FS, k Kind) ([]recipeFile, error) {
	dir, err := fs.Sub(recipe, string(k))
	if err != nil {
		return nil, err
	}
	var files []recipeFile
	err = fs.WalkDir(dir, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(dir, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		files = append(files, recipeFile{path, hex.EncodeToString(sum[:])})
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
