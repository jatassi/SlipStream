package cues

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"slices"
	"testing"
)

var (
	vobSub = append([]byte{0, 0, 1, 0xBA, 0x44, 0, 4, 0, 4, 1, 1, 0x89, 0xC3, 0xF8}, make([]byte, 64)...)
	vobIdx = []byte("# VobSub index file, v7 (do not modify this line!)\nsize: 720x480\nid: en, index: 0\ntimestamp: 00:00:01:000, filepos: 000000000\n")
	pgsSup = append([]byte{'P', 'G', 0, 0, 0x2E, 0xE0, 0, 0, 0, 0, 0x16, 0, 0x13}, make([]byte, 19)...)
)

func TestImage(t *testing.T) {
	for name, b := range map[string][]byte{"VobSub sub": vobSub, "VobSub idx": vobIdx, "PGS sup": pgsSup} {
		if _, err := Parse(b, lang(t, "en"), 25, limit); !errors.Is(err, ErrImage) {
			t.Errorf("%s: %v; want ErrImage", name, err)
		}
	}
	es, err := Unpack(zipOf(t, member{"Heat.1995.sup", pgsSup}, member{"Heat.1995.en.srt", srt(1)}), caps)
	if err != nil || len(es) != 2 {
		t.Fatalf("%v, %d members; want the image member kept beside the text one", err, len(es))
	}
	e, ok := Pick(es, 0, nil, lang(t, "en"))
	if !ok || e.Name != "Heat.1995.en.srt" {
		t.Errorf("picked %q; want the text member over the larger image one", e.Name)
	}
	e, ok = Pick(es[:1], 0, nil, lang(t, "en"))
	if _, err := Parse(e.Bytes, lang(t, "en"), 25, limit); !ok || !errors.Is(err, ErrImage) {
		t.Errorf("an archive holding only an image picks %q, parsed: %v; want ErrImage", e.Name, err)
	}
}

var caps = Caps{Members: 20, Member: 1 << 20, Total: 4 << 20}

type member struct {
	name string
	b    []byte
}

func zipOf(t *testing.T, ms ...member) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, m := range ms {
		f, err := w.Create(m.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(m.b); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// srt is an SRT of n cues, with CRLF line ends as Gestdown serves them.
func srt(n int) []byte {
	var b bytes.Buffer
	for i := range n {
		fmt.Fprintf(&b, "%d\r\n00:%02d:%02d,000 --> 00:%02d:%02d,500\r\nLine %d of the episode.\r\n\r\n", i+1, i/60, i%60, i/60, i%60, i)
	}
	return b.Bytes()
}

func names(es []Entry) []string {
	var ns []string
	for _, e := range es {
		ns = append(ns, e.Name)
	}
	return ns
}

func TestUnpack(t *testing.T) {
	junk := []byte("1\r\n00:00:01,000 --> 00:00:05,000\r\nSchitt's Creek SEASON 1 by subdl.com\r\n\r\n")
	pack := zipOf(t,
		member{"00 Schitt's Creek SEASON 1 subdl.com.srt", junk},
		member{"01 Pilot.en.srt", srt(400)},
		member{"02 Our Cup Runneth Over.en.srt", srt(410)},
		member{"__MACOSX/._01 Pilot.en.srt", []byte{0, 5, 22, 7, 0, 2, 0, 0, 'M', 'a', 'c', ' ', 'O', 'S', ' ', 'X'}},
		member{"Schitt's Creek/", nil},
		member{"read me.txt", []byte("Downloaded from SubDL. Enjoy the show.\n")},
		member{"release.nfo", []byte("RELEASE.....: Schitt's Creek S01\n")},
		member{"cover.jpg", []byte("\xff\xd8\xff\xe0\x00\x10JFIF")},
		member{"nested.zip", zipOf(t, member{"inner.srt", srt(2)})},
	)
	es, err := Unpack(pack, caps)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"00 Schitt's Creek SEASON 1 subdl.com.srt", "01 Pilot.en.srt", "02 Our Cup Runneth Over.en.srt"}
	if !slices.Equal(names(es), want) {
		t.Fatalf("members %q; want %q", names(es), want)
	}
	if !bytes.Equal(es[1].Bytes, srt(400)) {
		t.Error("a member's bytes changed")
	}

	es, err = Unpack(rarOf(t, member{"Heat (1995).eng.srt", srt(3)}, member{"Heat (1995).nfo", []byte("notes\n")}), caps)
	if err != nil || !slices.Equal(names(es), []string{"Heat (1995).eng.srt"}) || !bytes.Equal(es[0].Bytes, srt(3)) {
		t.Fatalf("a RAR under a zip name: %q, %v", names(es), err)
	}

	if es, err := Unpack(zipOf(t), caps); err != nil || len(es) != 0 {
		t.Errorf("an empty zip: %q, %v; want no members", names(es), err)
	}
	if _, err := Unpack(srt(3), caps); !errors.Is(err, ErrNotArchive) {
		t.Errorf("a raw SRT: %v; want ErrNotArchive", err)
	}

	for name, c := range map[string]Caps{
		"members": {Members: 2, Member: caps.Member, Total: caps.Total},
		"member":  {Members: caps.Members, Member: len(srt(400)) - 1, Total: caps.Total},
		"total":   {Members: caps.Members, Member: caps.Member, Total: len(srt(400)) + len(srt(410))},
	} {
		if _, err := Unpack(pack, c); !errors.Is(err, ErrTooLarge) {
			t.Errorf("over the %s cap: %v; want ErrTooLarge", name, err)
		}
	}
}

// rarOf writes a RAR 5 archive that stores ms uncompressed.
func rarOf(t *testing.T, ms ...member) []byte {
	t.Helper()
	out := []byte("Rar!\x1a\x07\x01\x00")
	header := func(fields []byte) []byte {
		body := append(binary.AppendUvarint(nil, uint64(len(fields))), fields...)
		return append(binary.LittleEndian.AppendUint32(nil, crc32.ChecksumIEEE(body)), body...)
	}
	out = append(out, header([]byte{1, 0, 0})...)
	for _, m := range ms {
		var f []byte
		f = binary.AppendUvarint(f, 2)
		f = binary.AppendUvarint(f, 0x02)
		f = binary.AppendUvarint(f, uint64(len(m.b)))
		f = binary.AppendUvarint(f, 0x04)
		f = binary.AppendUvarint(f, uint64(len(m.b)))
		f = binary.AppendUvarint(f, 0x20)
		f = binary.LittleEndian.AppendUint32(f, crc32.ChecksumIEEE(m.b))
		f = binary.AppendUvarint(f, 0)
		f = binary.AppendUvarint(f, 0)
		f = binary.AppendUvarint(f, uint64(len(m.name)))
		f = append(f, m.name...)
		out = append(append(out, header(f)...), m.b...)
	}
	return append(out, header([]byte{5, 0, 0})...)
}
