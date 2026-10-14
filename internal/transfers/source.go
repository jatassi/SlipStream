package transfers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/link"
	transferv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
)

type SourceConfig struct {
	Payloads Payloads
	// Controller is the one peer the Source serves.
	Controller link.Key
	Hooks      Hooks
	Log        *slog.Logger
}

// Source answers a receiver's streams from the files this Node holds.
type Source struct {
	cfg SourceConfig

	mu        sync.Mutex
	manifests map[fileKey]Manifest
}

// fileKey is a file as it stood when hashed, so a resumed stream reuses the
// manifest and a changed file is hashed again.
type fileKey struct {
	path     string
	size     int64
	modified time.Time
}

func NewSource(cfg SourceConfig) *Source {
	return &Source{cfg: cfg, manifests: map[fileKey]Manifest{}}
}

// Serve answers one stream: the kind-2 handler.
func (s *Source) Serve(peer link.Key, st link.Stream) {
	if peer != s.cfg.Controller {
		_ = st.Reset()
		return
	}
	if err := s.serve(newFrames(st)); err != nil {
		s.cfg.Log.Warn("transfer source stopped", "err", err)
	}
	_ = st.Close()
}

func (s *Source) serve(f *frames) error {
	fr, err := f.read()
	if err != nil {
		return err
	}
	open := fr.GetOpen()
	if open == nil || len(open.GetInfoHash()) != len(bencode.InfoHash{}) {
		return fmt.Errorf("the receiver opened with %v", fr)
	}
	hash, index := bencode.InfoHash(open.GetInfoHash()), int(open.GetFileIndex())
	in, m, err := s.file(hash, index)
	if err != nil {
		s.cfg.Log.Info("transfer source has no file", "info_hash", hash, "file_index", index, "err", err)
		return f.send(absentFrame)
	}
	defer func() { _ = in.Close() }()
	if err := f.send(manifestFrame(m)); err != nil {
		return err
	}
	var r io.ReaderAt = in
	if s.cfg.Hooks.Read != nil {
		r = s.cfg.Hooks.Read(in)
	}
	return s.answer(f, r, m, hash, index)
}

func (s *Source) file(hash bencode.InfoHash, index int) (*os.File, Manifest, error) {
	path, err := s.cfg.Payloads.Path(hash, index)
	if err != nil {
		return nil, Manifest{}, err
	}
	in, err := os.Open(path) //nolint:gosec // the downloading client's own path for a torrent's file
	if err != nil {
		return nil, Manifest{}, fmt.Errorf("open the payload: %w", err)
	}
	m, err := s.manifest(path, in)
	if err != nil {
		_ = in.Close()
		return nil, Manifest{}, err
	}
	return in, m, nil
}

func (s *Source) manifest(path string, in *os.File) (Manifest, error) {
	info, err := in.Stat()
	if err != nil {
		return Manifest{}, fmt.Errorf("stat the payload: %w", err)
	}
	key := fileKey{path: path, size: info.Size(), modified: info.ModTime()}
	s.mu.Lock()
	m, ok := s.manifests[key]
	s.mu.Unlock()
	if ok {
		return m, nil
	}
	m = Manifest{Size: key.size, Hashes: make([]Hash, chunkCount(key.size))}
	buf := make([]byte, ChunkSize)
	for i := range m.Hashes {
		b := buf[:m.chunkLen(i)]
		if _, err := in.ReadAt(b, m.offset(i)); err != nil {
			return Manifest{}, fmt.Errorf("hash chunk %d of the payload: %w", i, err)
		}
		m.Hashes[i] = blake3.Sum256(b)
	}
	s.mu.Lock()
	s.manifests[key] = m
	s.mu.Unlock()
	return m, nil
}

// answer sends each chunk the receiver wants until it closes the stream.
func (s *Source) answer(f *frames, r io.ReaderAt, m Manifest, hash bencode.InfoHash, index int) error {
	buf := make([]byte, ChunkSize)
	for {
		fr, err := f.read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		chunks, err := wanted(fr, m)
		if err != nil {
			return err
		}
		if s.cfg.Hooks.Wanted != nil {
			s.cfg.Hooks.Wanted(hash, index, chunks)
		}
		for _, i := range chunks {
			b := buf[:m.chunkLen(i)]
			if _, err := r.ReadAt(b, m.offset(i)); err != nil {
				return fmt.Errorf("read chunk %d of the payload: %w", i, err)
			}
			if err := f.send(chunkFrame(i, b)); err != nil {
				return err
			}
		}
	}
}

func wanted(fr *transferv1.Frame, m Manifest) ([]int, error) {
	if fr.GetWant() == nil {
		return nil, fmt.Errorf("the receiver sent %v, not a Want", fr)
	}
	chunks := make([]int, len(fr.GetWant().GetChunks()))
	for i, c := range fr.GetWant().GetChunks() {
		if chunks[i] = int(c); chunks[i] >= len(m.Hashes) {
			return nil, fmt.Errorf("the receiver wants chunk %d of %d", c, len(m.Hashes))
		}
	}
	return chunks, nil
}
