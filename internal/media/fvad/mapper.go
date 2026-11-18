package fvad

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/speech"
)

const (
	// Mode is libfvad's most aggressive mode, which calls the least noise
	// speech.
	Mode = 3
	// frameSamples is one 10 ms frame at media.SampleRate.
	frameSamples = media.SampleRate / 100
	readBytes    = 64 << 10
)

// Mapper builds Speech Maps with the bundled ffmpeg and libfvad: one
// instance per map in Mode at 16 kHz, fed 10 ms frames in order. A
// partial last frame is dropped.
type Mapper struct{ Decoder media.Decoder }

// Map answers path's map. A file without the stream is speech.ErrNoAudio,
// a tool or detector that can't run is speech.ErrNodeFault, and audio
// ffmpeg can't read is a *speech.DecodeError.
func (m Mapper) Map(ctx context.Context, path string, stream int) (speech.Map, error) {
	vad, err := New(Mode, media.SampleRate)
	if err != nil {
		return speech.Map{}, fmt.Errorf("%w: %w", speech.ErrNodeFault, err)
	}
	defer vad.Close()
	audio, err := m.Decoder.Audio(ctx, path, stream)
	if err != nil {
		return speech.Map{}, mapError(err)
	}
	b, readErr := heard(audio, vad)
	closeErr := audio.Close()
	switch {
	case ctx.Err() != nil:
		return speech.Map{}, fmt.Errorf("map the speech: %w", ctx.Err())
	case readErr != nil:
		return speech.Map{}, readErr
	case closeErr != nil:
		return speech.Map{}, mapError(closeErr)
	}
	return b.Map(speech.Header{
		Format: speech.FormatVersion, VAD: Version, Mode: Mode, Rate: media.SampleRate, FrameMS: int(speech.Frame.Milliseconds()),
		Stream: audio.Stream, Recipe: media.Recipe, Start: audio.Start,
	})
}

func heard(r io.Reader, vad *VAD) (*speech.Builder, error) {
	var b speech.Builder
	buf := make([]byte, readBytes)
	frame := make([]int16, frameSamples)
	held := 0
	for {
		n, err := r.Read(buf[held:])
		held += n
		whole := held / (2 * frameSamples) * (2 * frameSamples)
		for at := 0; at < whole; at += 2 * frameSamples {
			for i := range frame {
				frame[i] = int16(binary.LittleEndian.Uint16(buf[at+2*i:])) //nolint:gosec // the sample's two's-complement bits
			}
			s, err := vad.Speech(frame)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", speech.ErrNodeFault, err)
			}
			if b.Frames() == speech.MaxFrames {
				return nil, &speech.DecodeError{First: "the audio runs longer than 12 hours"}
			}
			b.Add(s)
		}
		held = copy(buf, buf[whole:held])
		switch {
		case errors.Is(err, io.EOF):
			return &b, nil
		case err != nil:
			return nil, fmt.Errorf("read the decoded audio: %w", err)
		}
	}
}

func mapError(err error) error {
	if d, ok := errors.AsType[*media.DecodeError](err); ok {
		return &speech.DecodeError{First: d.First}
	}
	switch {
	case errors.Is(err, media.ErrNoAudio):
		return speech.ErrNoAudio
	case errors.Is(err, media.ErrStart):
		return fmt.Errorf("%w: %w", speech.ErrNodeFault, err)
	}
	return err
}
