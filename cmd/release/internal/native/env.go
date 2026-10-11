package native

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// MetadataFile sits at a libs layer's root. The recipe writes it.
const MetadataFile = "slipstream-native.json"

// Metadata lists what cgo needs to build against a layer. Paths are relative
// to the layer root, so the layer can live anywhere.
type Metadata struct {
	IncludeDirs []string `json:"include_dirs"`
	Defines     []string `json:"defines"`
	// StaticLibs are in link order.
	StaticLibs []string `json:"static_libs"`
	SystemLibs []string `json:"system_libs"`
}

func ReadMetadata(layerDir string) (Metadata, error) {
	data, err := os.ReadFile(filepath.Join(layerDir, MetadataFile))
	if err != nil {
		return Metadata{}, err
	}
	var md Metadata
	return md, json.Unmarshal(data, &md)
}

type EnvVar struct {
	Name, Value string
}

// CgoEnv is the environment that builds cgo packages against the layer at
// layerDir. The flags start with cgo's own defaults, -O2 -g, which setting
// the variables would otherwise drop.
func CgoEnv(layerDir string, md Metadata, tc Toolchain) []EnvVar {
	abs := func(p string) string { return filepath.Join(layerDir, filepath.FromSlash(p)) }
	cflags := []string{"-O2", "-g"}
	for _, dir := range md.IncludeDirs {
		cflags = append(cflags, "-I"+abs(dir))
	}
	for _, def := range md.Defines {
		cflags = append(cflags, "-D"+def)
	}
	var ldflags []string
	for _, lib := range md.StaticLibs {
		ldflags = append(ldflags, abs(lib))
	}
	ldflags = append(ldflags, md.SystemLibs...)
	return []EnvVar{
		{"CC", tc.CC},
		{"CXX", tc.CXX},
		{"CGO_CFLAGS", joinFlags(cflags)},
		{"CGO_CXXFLAGS", joinFlags(cflags)},
		{"CGO_LDFLAGS", joinFlags(ldflags)},
	}
}

// joinFlags quotes each flag the way the go command splits CGO_*FLAGS:
// whitespace separates, and single or double quotes group without escapes.
func joinFlags(flags []string) string {
	quoted := make([]string, len(flags))
	for i, f := range flags {
		switch {
		case !strings.ContainsAny(f, " \t\r\n'\""):
			quoted[i] = f
		case !strings.Contains(f, "'"):
			quoted[i] = "'" + f + "'"
		default:
			quoted[i] = `"` + f + `"`
		}
	}
	return strings.Join(quoted, " ")
}

// ExportLines renders vars as POSIX shell export lines for eval.
func ExportLines(vars []EnvVar) string {
	var b strings.Builder
	for _, v := range vars {
		b.WriteString("export " + v.Name + "='" + strings.ReplaceAll(v.Value, "'", `'\''`) + "'\n")
	}
	return b.String()
}
