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

// LibsDigest names a libs layer by everything that goes into building it:
// the target row, every component, and the path and content of every recipe
// file. Two inputs that hash alike build the same layer, so an existing
// layer with this digest is never rebuilt.
func LibsDigest(in Inputs, t Target, recipe fs.FS) (string, error) {
	var files []recipeFile
	err := fs.WalkDir(recipe, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(recipe, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		files = append(files, recipeFile{path, hex.EncodeToString(sum[:])})
		return nil
	})
	if err != nil {
		return "", err
	}
	doc, err := json.Marshal(struct {
		Layer      string       `json:"layer"`
		Target     Target       `json:"target"`
		Components []Component  `json:"components"`
		Recipe     []recipeFile `json:"recipe"`
	}{"libs", t, in.Components, files})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(doc)
	return hex.EncodeToString(sum[:]), nil
}
