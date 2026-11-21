package cues

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/nwaples/rardecode/v2"

	"github.com/jatassi/slipstream/internal/language"
)

var (
	ErrNotArchive = errors.New("the bytes aren't a zip or RAR archive")
	ErrTooLarge   = errors.New("the archive is over its size or member limit")
)

// Caps bound one archive: how many members it may hold, the bytes of any
// one member, and the bytes of all members together.
type Caps struct{ Members, Member, Total int }

// Entry is one subtitle member: its name in the archive and its bytes.
type Entry struct {
	Name  string
	Bytes []byte
}

// Unpack opens a zip or a RAR by its magic, whatever the download was
// named, and answers its subtitle members in archive order. A member is a
// subtitle when its bytes are a text or image format. Folders, __MACOSX
// forks, .nfo and .txt notes, pictures and nested archives are not. An
// archive with no subtitle members answers none.
func Unpack(b []byte, c Caps) ([]Entry, error) {
	switch {
	case isZip(b):
		return unzip(b, c)
	case isRAR(b):
		return unrar(b, c)
	}
	return nil, ErrNotArchive
}

func isZip(b []byte) bool {
	return bytes.HasPrefix(b, []byte("PK\x03\x04")) || bytes.HasPrefix(b, []byte("PK\x05\x06"))
}

func isRAR(b []byte) bool { return bytes.HasPrefix(b, []byte("Rar!\x1a\x07")) }

func unzip(b []byte, c Caps) ([]Entry, error) {
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, fmt.Errorf("read the zip archive: %w", err)
	}
	u := unpacking{caps: c}
	for _, f := range z.File {
		if f.FileInfo().IsDir() {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open %q in the zip archive: %w", f.Name, err)
		}
		err = u.add(f.Name, r)
		_ = r.Close()
		if err != nil {
			return nil, err
		}
	}
	return u.entries, nil
}

func unrar(b []byte, c Caps) ([]Entry, error) {
	r, err := rardecode.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("read the RAR archive: %w", err)
	}
	u := unpacking{caps: c}
	for {
		h, err := r.Next()
		if errors.Is(err, io.EOF) {
			return u.entries, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read the RAR archive: %w", err)
		}
		if h.IsDir {
			continue
		}
		if err := u.add(h.Name, r); err != nil {
			return nil, err
		}
	}
}

type unpacking struct {
	caps    Caps
	members int
	total   int
	entries []Entry
}

// add reads one member under the caps, counting every member it reads,
// and keeps it when it is a subtitle.
func (u *unpacking) add(name string, r io.Reader) error {
	u.members++
	if u.members > u.caps.Members {
		return fmt.Errorf("%w: more than %d members", ErrTooLarge, u.caps.Members)
	}
	b, err := io.ReadAll(io.LimitReader(r, int64(u.caps.Member)+1))
	if err != nil {
		return fmt.Errorf("read %q in the archive: %w", name, err)
	}
	if len(b) > u.caps.Member {
		return fmt.Errorf("%w: %q is over %d bytes", ErrTooLarge, name, u.caps.Member)
	}
	if u.total += len(b); u.total > u.caps.Total {
		return fmt.Errorf("%w: its members are over %d bytes", ErrTooLarge, u.caps.Total)
	}
	if subtitle(b) {
		u.entries = append(u.entries, Entry{Name: name, Bytes: b})
	}
	return nil
}

// subtitle reports whether b is an image subtitle, or text in a format
// sniff reads once decoded with the default code pages. A nested archive
// is not, though its stored members can read as text.
func subtitle(b []byte) bool {
	if image(b) != 0 {
		return true
	}
	if isZip(b) || isRAR(b) {
		return false
	}
	text, _, err := Decode(b, language.Code{})
	return err == nil && sniff(text) != 0
}
