package transfers

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/link"
	transferv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transferlegacy/v1"
)

// maxFrame is the largest frame either side reads: a whole chunk and its
// envelope.
const maxFrame = ChunkSize + 64<<10

// frames reads and writes one stream's frames, each a uvarint length and
// then a transferv1.Frame.
type frames struct {
	s link.Stream
	r *bufio.Reader
}

func newFrames(s link.Stream) *frames { return &frames{s: s, r: bufio.NewReader(s)} }

func (f *frames) send(fr *transferv1.Frame) error {
	b, err := proto.Marshal(fr)
	if err != nil {
		return fmt.Errorf("encode a frame: %w", err)
	}
	out := binary.AppendUvarint(make([]byte, 0, binary.MaxVarintLen64+len(b)), uint64(len(b)))
	if _, err := f.s.Write(append(out, b...)); err != nil {
		return fmt.Errorf("send a frame: %w", err)
	}
	return nil
}

// read returns the next frame. A frame that doesn't decode is a
// *FailedError: the peer runs this version, so it won't decode next time.
func (f *frames) read() (*transferv1.Frame, error) {
	n, err := binary.ReadUvarint(f.r)
	if err != nil {
		return nil, fmt.Errorf("read a frame: %w", err)
	}
	if n > maxFrame {
		return nil, &FailedError{Reason: fmt.Sprintf("the peer sent a frame of %d bytes, over %d", n, maxFrame)}
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(f.r, b); err != nil {
		return nil, fmt.Errorf("read a frame: %w", err)
	}
	var fr transferv1.Frame
	if err := proto.Unmarshal(b, &fr); err != nil {
		return nil, &FailedError{Reason: "the peer sent a frame that doesn't decode: " + err.Error()}
	}
	return &fr, nil
}

func openFrame(j Job) *transferv1.Frame {
	return &transferv1.Frame{Frame: &transferv1.Frame_Open{Open: &transferv1.Open{
		InfoHash:  j.InfoHash[:],
		FileIndex: uint32(j.FileIndex), //nolint:gosec // a torrent's file index, from its own file list
	}}}
}

func manifestFrame(m Manifest) *transferv1.Frame {
	hashes := make([][]byte, len(m.Hashes))
	for i := range m.Hashes {
		hashes[i] = m.Hashes[i][:]
	}
	return &transferv1.Frame{Frame: &transferv1.Frame_Manifest{Manifest: &transferv1.Manifest{
		Size: m.Size, ChunkSize: ChunkSize, ChunkHashes: hashes,
	}}}
}

func parseManifest(p *transferv1.Manifest) (Manifest, error) {
	switch {
	case p.GetChunkSize() != ChunkSize:
		return Manifest{}, &FailedError{Reason: fmt.Sprintf("the source's chunks are %d bytes, not %d", p.GetChunkSize(), ChunkSize)}
	case p.GetSize() < 0 || len(p.GetChunkHashes()) != chunkCount(p.GetSize()):
		return Manifest{}, &FailedError{Reason: fmt.Sprintf("the source's manifest has %d hashes for %d bytes", len(p.GetChunkHashes()), p.GetSize())}
	}
	m := Manifest{Size: p.GetSize(), Hashes: make([]Hash, len(p.GetChunkHashes()))}
	for i, h := range p.GetChunkHashes() {
		if len(h) != len(Hash{}) {
			return Manifest{}, &FailedError{Reason: fmt.Sprintf("the source's hash of chunk %d is %d bytes", i, len(h))}
		}
		m.Hashes[i] = Hash(h)
	}
	return m, nil
}

func wantFrame(chunks []int) *transferv1.Frame {
	w := make([]uint32, len(chunks))
	for i, c := range chunks {
		w[i] = uint32(c) //nolint:gosec // a chunk index, under a Manifest's chunk count
	}
	return &transferv1.Frame{Frame: &transferv1.Frame_Want{Want: &transferv1.Want{Chunks: w}}}
}

func chunkFrame(i int, data []byte) *transferv1.Frame {
	return &transferv1.Frame{Frame: &transferv1.Frame_Chunk{Chunk: &transferv1.Chunk{
		Index: uint32(i), //nolint:gosec // a chunk index, under a Manifest's chunk count
		Data:  data,
	}}}
}

var absentFrame = &transferv1.Frame{Frame: &transferv1.Frame_Absent{Absent: &transferv1.Absent{}}}
