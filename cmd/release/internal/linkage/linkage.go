// Package linkage checks that a release binary loads only OS-provided libraries (P-352).
package linkage

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

// Import is one library the OS loader must resolve before the binary runs.
type Import struct {
	Kind string
	Name string
}

// Allowed is one import the policy permits and the reason it ships.
type Allowed struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Policy lists, per GOOS, every import a release binary may carry.
type Policy map[string][]Allowed

const policyPath = "cmd/release/internal/linkage/policy.json"

//go:embed policy.json
var policyJSON []byte

// Check fails when the binary at path carries an import its OS's policy does not list.
func Check(path string) error {
	var policy Policy
	if err := json.Unmarshal(policyJSON, &policy); err != nil {
		return fmt.Errorf("%s: %w", policyPath, err)
	}
	goos, imports, err := read(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	var unlisted []string
	for _, imp := range imports {
		listed := slices.ContainsFunc(policy[goos], func(a Allowed) bool { return a.Name == imp.Name })
		if !listed {
			unlisted = append(unlisted, imp.Kind+" "+imp.Name)
		}
	}
	if len(unlisted) > 0 {
		return fmt.Errorf("%s: %s binary imports what %s does not list: %s",
			path, goos, policyPath, strings.Join(unlisted, ", "))
	}
	return nil
}

var formats = []struct {
	magic string
	goos  string
	read  func(path string) ([]Import, error)
}{
	{"\x7fELF", "linux", readELF},
	{"MZ", "windows", readPE},
	{"\xcf\xfa\xed\xfe", "darwin", readMachO},
}

func read(path string) (string, []Import, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("open the binary: %w", err)
	}
	head := make([]byte, 4)
	_, err = io.ReadFull(f, head)
	_ = f.Close()
	if err != nil {
		return "", nil, fmt.Errorf("%s: read the magic number: %w", path, err)
	}
	for _, format := range formats {
		if strings.HasPrefix(string(head), format.magic) {
			imports, err := format.read(path)
			return format.goos, imports, err
		}
	}
	return "", nil, errors.New("not an ELF, PE or 64-bit Mach-O binary")
}

func readELF(path string) ([]Import, error) {
	f, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	var imports []Import
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			interp, err := io.ReadAll(p.Open())
			if err != nil {
				return nil, fmt.Errorf("%s: read the interpreter: %w", path, err)
			}
			imports = append(imports, Import{"interpreter", strings.TrimRight(string(interp), "\x00")})
		}
	}
	libs, err := f.ImportedLibraries()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, lib := range libs {
		imports = append(imports, Import{"library", lib})
	}
	return imports, nil
}

func readPE(path string) ([]Import, error) {
	f, err := pe.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	syms, err := f.ImportedSymbols()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var dlls []string
	for _, sym := range syms {
		dlls = append(dlls, strings.ToLower(sym[strings.LastIndex(sym, ":")+1:]))
	}
	slices.Sort(dlls)
	var imports []Import
	for _, dll := range slices.Compact(dlls) {
		imports = append(imports, Import{"library", dll})
	}
	return imports, nil
}

var dylibLoads = map[macho.LoadCmd]string{
	0xc:        "library",
	0x80000018: "weak library",
	0x8000001f: "reexported library",
	0x20:       "lazy library",
	0x80000023: "upward library",
}

func readMachO(path string) ([]Import, error) {
	f, err := macho.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	var imports []Import
	for _, load := range f.Loads {
		raw := load.Raw()
		kind, ok := dylibLoads[macho.LoadCmd(f.ByteOrder.Uint32(raw))]
		if !ok {
			continue
		}
		name, _, _ := bytes.Cut(raw[f.ByteOrder.Uint32(raw[8:]):], []byte{0})
		imports = append(imports, Import{kind, string(name)})
	}
	return imports, nil
}
