// Package linkage checks that a release binary loads only OS-provided
// libraries (P-352) and, on Windows, has its role's subsystem.
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
	"strconv"
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

// Role is what a release binary is for, which decides what it may import
// and, on Windows, its subsystem.
type Role string

const (
	// Node is the slipstream binary, and the e2e build of it.
	Node Role = "node"
	// Trampoline is slipstream-app, the desktop launcher.
	Trampoline Role = "trampoline"
)

// subsystems is each role's PE subsystem. The Node is a console program,
// so a terminal waits for it and gets its exit code; the trampoline is a
// GUI program, so a shortcut or a sign-in that starts it shows no console.
var subsystems = map[Role]uint16{
	Node:       pe.IMAGE_SUBSYSTEM_WINDOWS_CUI,
	Trampoline: pe.IMAGE_SUBSYSTEM_WINDOWS_GUI,
}

var subsystemNames = map[uint16]string{
	pe.IMAGE_SUBSYSTEM_WINDOWS_CUI: "console",
	pe.IMAGE_SUBSYSTEM_WINDOWS_GUI: "GUI",
}

func subsystemName(s uint16) string {
	if name, ok := subsystemNames[s]; ok {
		return name
	}
	return strconv.Itoa(int(s))
}

// Policy lists, per role and GOOS, every import a release binary may carry.
type Policy map[Role]map[string][]Allowed

const policyPath = "cmd/release/internal/linkage/policy.json"

//go:embed policy.json
var policyJSON []byte

// Check fails when the binary at path carries an import its role's policy
// for its OS does not list, or when a Windows binary has another role's
// subsystem.
func Check(path string, role Role) error {
	var policy Policy
	if err := json.Unmarshal(policyJSON, &policy); err != nil {
		return fmt.Errorf("%s: %w", policyPath, err)
	}
	b, err := read(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	allowed, ok := policy[role][b.goos]
	if !ok {
		return fmt.Errorf("%s: %s lists no %s %s binary", path, policyPath, b.goos, role)
	}
	var unlisted []string
	for _, imp := range b.imports {
		if !slices.ContainsFunc(allowed, func(a Allowed) bool { return a.Name == imp.Name }) {
			unlisted = append(unlisted, imp.Kind+" "+imp.Name)
		}
	}
	if len(unlisted) > 0 {
		return fmt.Errorf("%s: %s %s binary imports what %s does not list: %s",
			path, b.goos, role, policyPath, strings.Join(unlisted, ", "))
	}
	if want := subsystems[role]; b.goos == "windows" && b.subsystem != want {
		return fmt.Errorf("%s: windows %s binary has the %s subsystem, want %s",
			path, role, subsystemName(b.subsystem), subsystemNames[want])
	}
	return nil
}

type binary struct {
	goos      string
	imports   []Import
	subsystem uint16
}

var formats = []struct {
	magic string
	goos  string
	read  func(path string) (binary, error)
}{
	{"\x7fELF", "linux", readELF},
	{"MZ", "windows", readPE},
	{"\xcf\xfa\xed\xfe", "darwin", readMachO},
}

func read(path string) (binary, error) {
	f, err := os.Open(path)
	if err != nil {
		return binary{}, fmt.Errorf("open the binary: %w", err)
	}
	head := make([]byte, 4)
	_, err = io.ReadFull(f, head)
	_ = f.Close()
	if err != nil {
		return binary{}, fmt.Errorf("%s: read the magic number: %w", path, err)
	}
	for _, format := range formats {
		if strings.HasPrefix(string(head), format.magic) {
			b, err := format.read(path)
			b.goos = format.goos
			return b, err
		}
	}
	return binary{}, errors.New("not an ELF, PE or 64-bit Mach-O binary")
}

func readELF(path string) (binary, error) {
	f, err := elf.Open(path)
	if err != nil {
		return binary{}, fmt.Errorf("%s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	var b binary
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			interp, err := io.ReadAll(p.Open())
			if err != nil {
				return binary{}, fmt.Errorf("%s: read the interpreter: %w", path, err)
			}
			b.imports = append(b.imports, Import{"interpreter", strings.TrimRight(string(interp), "\x00")})
		}
	}
	libs, err := f.ImportedLibraries()
	if err != nil {
		return binary{}, fmt.Errorf("%s: %w", path, err)
	}
	for _, lib := range libs {
		b.imports = append(b.imports, Import{"library", lib})
	}
	return b, nil
}

func readPE(path string) (binary, error) {
	f, err := pe.Open(path)
	if err != nil {
		return binary{}, fmt.Errorf("%s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	var b binary
	switch h := f.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		b.subsystem = h.Subsystem
	case *pe.OptionalHeader32:
		b.subsystem = h.Subsystem
	default:
		return binary{}, fmt.Errorf("%s: no PE optional header", path)
	}
	syms, err := f.ImportedSymbols()
	if err != nil {
		return binary{}, fmt.Errorf("%s: %w", path, err)
	}
	var dlls []string
	for _, sym := range syms {
		dlls = append(dlls, strings.ToLower(sym[strings.LastIndex(sym, ":")+1:]))
	}
	slices.Sort(dlls)
	for _, dll := range slices.Compact(dlls) {
		b.imports = append(b.imports, Import{"library", dll})
	}
	return b, nil
}

var dylibLoads = map[macho.LoadCmd]string{
	0xc:        "library",
	0x80000018: "weak library",
	0x8000001f: "reexported library",
	0x20:       "lazy library",
	0x80000023: "upward library",
}

func readMachO(path string) (binary, error) {
	f, err := macho.Open(path)
	if err != nil {
		return binary{}, fmt.Errorf("%s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	var b binary
	for _, load := range f.Loads {
		raw := load.Raw()
		kind, ok := dylibLoads[macho.LoadCmd(f.ByteOrder.Uint32(raw))]
		if !ok {
			continue
		}
		name, _, _ := bytes.Cut(raw[f.ByteOrder.Uint32(raw[8:]):], []byte{0})
		b.imports = append(b.imports, Import{kind, string(name)})
	}
	return b, nil
}
