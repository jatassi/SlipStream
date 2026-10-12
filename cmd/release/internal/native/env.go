package native

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// MetadataFile sits at every layer's root. The recipe writes it, and every
// path in it is relative to the layer root, so the layer can live anywhere.
const MetadataFile = "slipstream-native.json"

// Compilers is a toolchain layer's metadata.
type Compilers struct {
	CC  string `json:"cc"`
	CXX string `json:"cxx"`
}

// ReadCompilers reads the compilers of the toolchain layer at layerDir as
// absolute paths.
func ReadCompilers(layerDir string) (Compilers, error) {
	var cc Compilers
	if err := readMetadata(layerDir, &cc); err != nil {
		return Compilers{}, err
	}
	return Compilers{abs(layerDir, cc.CC), abs(layerDir, cc.CXX)}, nil
}

// Metadata is a libs layer's metadata: what cgo needs to build against it.
type Metadata struct {
	IncludeDirs []string `json:"include_dirs"`
	Defines     []string `json:"defines"`
	// StaticLibs are in link order.
	StaticLibs []string `json:"static_libs"`
	SystemLibs []string `json:"system_libs"`
}

func ReadMetadata(layerDir string) (Metadata, error) {
	var md Metadata
	return md, readMetadata(layerDir, &md)
}

func readMetadata(layerDir string, v any) error {
	data, err := os.ReadFile(filepath.Join(layerDir, MetadataFile))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func abs(layerDir, path string) string { return filepath.Join(layerDir, filepath.FromSlash(path)) }

type EnvVar struct {
	Name, Value string
}

// CgoEnv is the environment that builds cgo packages against the libs layer
// at layerDir with cc and the target's build tags. The flags start with
// cgo's own defaults, -O2 -g, which setting the variables would otherwise
// drop.
func CgoEnv(layerDir string, md Metadata, cc Compilers, tags []string) []EnvVar {
	cflags := []string{"-O2", "-g"}
	for _, dir := range md.IncludeDirs {
		cflags = append(cflags, "-I"+abs(layerDir, dir))
	}
	for _, def := range md.Defines {
		cflags = append(cflags, "-D"+def)
	}
	var ldflags []string
	for _, lib := range md.StaticLibs {
		ldflags = append(ldflags, abs(layerDir, lib))
	}
	ldflags = append(ldflags, md.SystemLibs...)
	return []EnvVar{
		{"CGO_ENABLED", "1"},
		{"GOFLAGS", "-tags=" + strings.Join(tags, ",")},
		{"CC", cc.CC},
		{"CXX", cc.CXX},
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
