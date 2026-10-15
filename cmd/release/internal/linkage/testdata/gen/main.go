// Command gen writes the minimal binaries the linkage tests check.
// Run it from the linkage package: go run ./testdata/gen
package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
)

func main() {
	for name, data := range map[string][]byte{
		"testdata/dynamic-linux.elf":   dynamicELF(),
		"testdata/foreign-dll.exe":     peFile(3, "libcrypto-3-x64.dll", "KERNEL32.dll"),
		"testdata/gui.exe":             peFile(2, "KERNEL32.dll"),
		"testdata/foreign-dylib.macho": foreignMachO(),
	} {
		if err := os.WriteFile(name, data, 0o644); err != nil {
			log.Fatal(err)
		}
	}
}

func le(fields ...any) []byte {
	var b bytes.Buffer
	for _, f := range fields {
		if err := binary.Write(&b, binary.LittleEndian, f); err != nil {
			log.Fatal(err)
		}
	}
	return b.Bytes()
}

func pad(b []byte, n int) []byte {
	return append(b, make([]byte, n-len(b))...)
}

func dynamicELF() []byte {
	const (
		ehsize   = 64
		phOff    = ehsize
		phsize   = 56
		interpAt = phOff + phsize
		shsize   = 64
	)
	interp := []byte("/lib64/ld-linux-x86-64.so.2\x00")
	dynstr := []byte("\x00libc.so.6\x00")
	shstr := []byte("\x00.dynstr\x00.dynamic\x00.shstrtab\x00")
	dynstrAt := interpAt + len(interp)
	dynamicAt := dynstrAt + len(dynstr)
	dynamic := le(int64(1), uint64(1), int64(0), uint64(0))
	shstrAt := dynamicAt + len(dynamic)
	shAt := shstrAt + len(shstr)

	header := le([16]byte{0x7f, 'E', 'L', 'F', 2, 1, 1},
		uint16(2), uint16(62), uint32(1), uint64(0), uint64(phOff), uint64(shAt),
		uint32(0), uint16(ehsize), uint16(phsize), uint16(1), uint16(shsize), uint16(4), uint16(3))
	prog := le(uint32(3), uint32(4), uint64(interpAt), uint64(interpAt), uint64(interpAt),
		uint64(len(interp)), uint64(len(interp)), uint64(1))
	section := func(name, typ, flags uint32, off, size, link, entsize int) []byte {
		return le(name, typ, uint64(flags), uint64(off), uint64(off), uint64(size),
			uint32(link), uint32(0), uint64(1), uint64(entsize))
	}
	return bytes.Join([][]byte{
		header, prog, interp, dynstr, dynamic, shstr,
		make([]byte, shsize),
		section(1, 3, 2, dynstrAt, len(dynstr), 0, 0),
		section(9, 6, 2, dynamicAt, len(dynamic), 1, 16),
		section(18, 3, 0, shstrAt, len(shstr), 0, 0),
	}, nil)
}

// peFile is a PE32+ of the given subsystem, 3 console or 2 GUI, importing
// one symbol from each DLL.
func peFile(subsystem uint16, dlls ...string) []byte {
	const (
		peAt      = 0x40
		optSize   = 112 + 16*8
		rawAt     = 0x200
		sectionVA = 0x1000
	)
	rva := func(off int) uint32 { return uint32(sectionVA + off) }
	const symbol = "\x00\x00OpenSSL_version\x00"
	descriptors := (len(dlls) + 1) * 20
	thunkAt := descriptors
	nameAt := thunkAt + 16
	dllAt := nameAt + len(symbol)
	var sec, names []byte
	for _, dll := range dlls {
		sec = append(sec, le(rva(thunkAt), uint32(0), uint32(0), rva(dllAt+len(names)), rva(thunkAt))...)
		names = append(names, dll+"\x00"...)
	}
	sec = append(sec, make([]byte, 20)...)
	sec = append(sec, le(uint64(rva(nameAt)), uint64(0))...)
	sec = append(sec, symbol...)
	sec = append(sec, names...)

	dos := pad(append([]byte("MZ"), make([]byte, 0x3a)...), 0x3c)
	dos = append(dos, le(uint32(peAt))...)
	fileHeader := le([4]byte{'P', 'E'}, uint16(0x8664), uint16(1), uint32(0), uint32(0), uint32(0),
		uint16(optSize), uint16(0x22))
	dirs := make([]byte, 16*8)
	copy(dirs[8:], le(rva(0), uint32(descriptors)))
	opt := le(uint16(0x20b), uint8(0), uint8(0), uint32(0x200), uint32(0), uint32(0),
		uint32(sectionVA), uint32(sectionVA), uint64(0x140000000), uint32(0x1000), uint32(0x200),
		uint16(6), uint16(0), uint16(0), uint16(0), uint16(6), uint16(0), uint32(0),
		uint32(0x2000), uint32(rawAt), uint32(0), subsystem, uint16(0x8160),
		uint64(0x100000), uint64(0x1000), uint64(0x100000), uint64(0x1000), uint32(0), uint32(16))
	opt = append(opt, dirs...)
	section := le([8]byte{'.', 'i', 'd', 'a', 't', 'a'}, uint32(len(sec)), uint32(sectionVA),
		uint32(0x200), uint32(rawAt), uint32(0), uint32(0), uint16(0), uint16(0), uint32(0xc0000040))
	head := pad(append(append(append(dos, fileHeader...), opt...), section...), rawAt)
	return append(head, pad(sec, 0x200)...)
}

func foreignMachO() []byte {
	dylib := func(cmd uint32, name string) []byte {
		size := (24 + len(name) + 1 + 7) &^ 7
		return pad(append(le(cmd, uint32(size), uint32(24), uint32(0), uint32(0x10000), uint32(0x10000)), name...), size)
	}
	cmds := append(
		dylib(0xc, "/usr/lib/libSystem.B.dylib"),
		dylib(0x80000018, "/opt/homebrew/opt/openssl@3/lib/libssl.3.dylib")...)
	header := le(uint32(0xfeedfacf), uint32(0x0100000c), uint32(0), uint32(2),
		uint32(2), uint32(len(cmds)), uint32(0x00200085), uint32(0))
	return append(header, cmds...)
}
