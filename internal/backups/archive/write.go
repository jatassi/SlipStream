package archive

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"filippo.io/age"
	"github.com/klauspost/compress/zstd"
	"github.com/zeebo/blake3"
)

// Write streams tar, then zstd, then age (scrypt, no armor) to w. It
// writes m with this binary's Format.
func Write(w io.Writer, m Manifest, s Sources, pass Passphrase, p Params) (Sums, error) {
	m.Format = Format
	manifest, err := json.Marshal(m)
	if err != nil {
		return Sums{}, fmt.Errorf("encode the manifest: %w", err)
	}
	lib, err := os.Open(s.LibraryDB)
	if err != nil {
		return Sums{}, fmt.Errorf("open the library snapshot: %w", err)
	}
	defer func() { _ = lib.Close() }()
	info, err := lib.Stat()
	if err != nil {
		return Sums{}, fmt.Errorf("open the library snapshot: %w", err)
	}

	recipient, err := age.NewScryptRecipient(pass.words)
	if err != nil {
		return Sums{}, fmt.Errorf("encrypt the Backup: %w", err)
	}
	recipient.SetWorkFactor(p.ScryptLog)
	out := &hashingWriter{w: w, h: blake3.New()}
	aw, err := age.Encrypt(out, recipient)
	if err != nil {
		return Sums{}, fmt.Errorf("encrypt the Backup: %w", err)
	}
	zw, err := zstd.NewWriter(aw, zstd.WithEncoderLevel(zstd.SpeedDefault), zstd.WithEncoderConcurrency(1), zstd.WithWindowSize(p.ZstdWindow))
	if err != nil {
		return Sums{}, fmt.Errorf("compress the Backup: %w", err)
	}
	tw := tar.NewWriter(zw)

	var sums Sums
	sources := [entries]struct {
		r    io.Reader
		size int64
	}{
		{bytes.NewReader(manifest), int64(len(manifest))},
		{lib, info.Size()},
		{bytes.NewReader(s.LibraryKey), int64(len(s.LibraryKey))},
		{bytes.NewReader(s.BackupKey), int64(len(s.BackupKey))},
		{bytes.NewReader(s.Index), int64(len(s.Index))},
	}
	for e, src := range sources {
		hdr := &tar.Header{Name: Entry(e).Name(), Typeflag: tar.TypeReg, Mode: 0o600, Size: src.size, ModTime: m.TakenAt}
		if err := tw.WriteHeader(hdr); err != nil {
			return Sums{}, fmt.Errorf("write %s: %w", hdr.Name, err)
		}
		h := blake3.New()
		if _, err := io.CopyN(io.MultiWriter(tw, h), src.r, src.size); err != nil {
			return Sums{}, fmt.Errorf("write %s: %w", hdr.Name, err)
		}
		h.Sum(sums.Entries[e][:0])
	}
	if err := errors.Join(tw.Close(), zw.Close(), aw.Close()); err != nil {
		return Sums{}, fmt.Errorf("finish the Backup: %w", err)
	}
	out.h.Sum(sums.Ciphertext[:0])
	sums.Size = out.n
	return sums, nil
}

type hashingWriter struct {
	w io.Writer
	h *blake3.Hasher
	n int64
}

func (hw *hashingWriter) Write(p []byte) (int, error) {
	n, err := hw.w.Write(p)
	_, _ = hw.h.Write(p[:n])
	hw.n += int64(n)
	return n, err //nolint:wrapcheck // Write wraps it as the entry's
}
