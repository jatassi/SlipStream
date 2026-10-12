package native

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
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

// xcodeCompilers are the compilers of the selected Xcode, which xcrun finds
// through DEVELOPER_DIR or xcode-select.
var xcodeCompilers = Compilers{"clang", "clang++"}

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

// CgoEnv is the environment that builds cgo packages for t against the
// libs layer at layerDir with cc. The flags start with cgo's own defaults,
// -O2 -g, which setting the variables would otherwise drop.
func CgoEnv(layerDir string, md Metadata, cc Compilers, t Target) []EnvVar {
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
	env := []EnvVar{
		{"CGO_ENABLED", "1"},
		{"GOFLAGS", "-tags=" + strings.Join(t.GoTags, ",")},
		{"CC", cc.CC},
		{"CXX", cc.CXX},
		{"CGO_CFLAGS", joinFlags(cflags)},
		{"CGO_CXXFLAGS", joinFlags(cflags)},
		{"CGO_LDFLAGS", joinFlags(ldflags)},
	}
	for _, name := range slices.Sorted(maps.Keys(t.Env)) {
		env = append(env, EnvVar{name, t.Env[name]})
	}
	return env
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
