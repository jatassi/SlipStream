// Package speech turns a Scenario's speech into the three things that
// must agree on it: timingtest's schedule, which subtitle cues follow;
// speechtest's runs, which mediafix renders as PCM bursts; and the Speech
// Map speechtest reads from those runs without decoding.
package speech

import (
	"github.com/jatassi/slipstream/fakes/scenario"
	mspeech "github.com/jatassi/slipstream/internal/media/speech"
	"github.com/jatassi/slipstream/internal/media/speech/speechtest"
	"github.com/jatassi/slipstream/internal/subtitles/timing/timingtest"
)

// Schedule is where s puts speech.
func Schedule(s scenario.MediaSpeech) timingtest.Schedule {
	return timingtest.NewSchedule(s.Seed, s.Length, s.Density)
}

// Runs are s's speech as speechtest writes it.
func Runs(s scenario.MediaSpeech) []speechtest.Run {
	sched := Schedule(s)
	out := make([]speechtest.Run, len(sched.Runs))
	for i, r := range sched.Runs {
		out[i] = speechtest.Run{Start: r.Start, End: r.End}
	}
	return out
}

// PCM is s as the mono 16-bit samples at speechtest.Rate a speech track
// holds.
func PCM(s scenario.MediaSpeech) []byte { return speechtest.PCM(Runs(s), s.Length) }

// Map is the Speech Map of stream, a speech track of s.
func Map(s scenario.MediaSpeech, stream int) (mspeech.Map, error) {
	return speechtest.Map(Runs(s), s.Length, stream)
}
