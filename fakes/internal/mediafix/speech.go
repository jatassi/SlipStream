package mediafix

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/converting/clip"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
)

// SpeechRecipe is a Matroska file whose audio is speech at known times:
// speechtest's PCM bursts in silence over Length, kept as pcm_s16le at 16
// kHz mono so ffmpeg only remuxes, beside a small black video. The audio
// starts Delay after the video.
type SpeechRecipe struct {
	Runs   []speechtest.Run `json:"runs"`
	Length time.Duration    `json:"length"`
	Delay  time.Duration    `json:"delay"`
}

// Speech returns the cached file r describes, making it first if the
// cache lacks it. Callers copy it out and never write to it.
func (b *Builder) Speech(ctx context.Context, r SpeechRecipe) (string, error) {
	key, err := json.Marshal(struct {
		FFmpeg string       `json:"ffmpeg"`
		Speech SpeechRecipe `json:"speech"`
	}{b.ffmpeg, r})
	if err != nil {
		return "", fmt.Errorf("fixture key: %w", err)
	}
	return b.cached(ctx, key, r.recipe(), ".mkv", "the speech file")
}

func (r SpeechRecipe) recipe() recipe {
	video := fmt.Sprintf("color=c=black:s=64x64:r=%d:d=%s", clip.FPS, seconds(r.Length+r.Delay))
	args := []string{"-v", "error", "-y", "-f", "lavfi", "-i", video, "-itsoffset", seconds(r.Delay), "-i", "speech.wav",
		"-map", "0:v", "-map", "1:a", "-c:v", "libx264", "-preset", "ultrafast", "-threads", "1", "-c:a", "pcm_s16le"}
	return recipe{
		files: []file{{name: "speech.wav", text: string(speechtest.WAV(r.Runs, r.Length))}},
		steps: []clip.Step{{Tool: clip.FFmpeg, Args: append(append(args, bitexact...), "speech.mkv")}},
		out:   "speech.mkv",
	}
}

func seconds(d time.Duration) string { return strconv.FormatFloat(d.Seconds(), 'f', -1, 64) }
