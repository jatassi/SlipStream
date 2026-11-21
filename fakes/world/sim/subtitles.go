package sim

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/gestdown"
	"github.com/jatassi/slipstream/fakes/internal/speech"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/subdl"
	"github.com/jatassi/slipstream/fakes/subsource"
	mspeech "github.com/jatassi/slipstream/internal/media/speech"
	"github.com/jatassi/slipstream/internal/node"
)

// subtitleFakes are the subtitle provider fakes the walk serves for a
// Scenario whose subtitle_providers catalog lists any subtitle.
var subtitleFakes = []fake.Spec{subdl.Fake, subsource.Fake, gestdown.Fake}

func subtitleFronts(s scenario.Scenario) []fault.Front {
	if len(s.SubtitleProviders.Catalog) == 0 {
		return nil
	}
	out := make([]fault.Front, len(subtitleFakes))
	for i, spec := range subtitleFakes {
		out[i] = fault.Front{Target: fault.Target(spec.Name), Layers: fault.HTTP}
	}
	return out
}

// serveSubtitles serves the subtitle provider fakes on local, reading the
// walk's fault Plan, with their cues played through World.Cue by name.
func (w *walk) serveSubtitles(t *testing.T, s scenario.Scenario, local *fault.Host, fakes *servers, endpoints *node.EndpointsBlock, log *slog.Logger) {
	t.Helper()
	if len(s.SubtitleProviders.Catalog) == 0 {
		return
	}
	for _, spec := range subtitleFakes {
		env := fake.NewEnv(s, spec, log, w.faults, nil)
		server, err := spec.New(s, env)
		if err != nil {
			t.Fatal(err)
		}
		spec.Endpoints(endpoints, fakes.serve(t, local, server))
		w.envs[spec.Name] = env
	}
}

// voices is how a Node hears each file's audio in the walk: a file holding
// a Scenario library file's bytes speaks as that file's speech track, so
// a copy or a hard link of it, such as an Import by hand, speaks the same.
// Any other file speaks as talk.
type voices map[string]scenario.MediaSpeech

func voicesOf(s scenario.Scenario) voices {
	out := voices{}
	if s.Library == nil {
		return out
	}
	for _, f := range s.Library.Folders {
		for _, m := range f.Files {
			if sp := m.SpeechTrack(); sp != nil {
				out[f.Folder+"/"+m.Path] = *sp
			}
		}
	}
	return out
}

func (v voices) Map(ctx context.Context, path string, stream int) (mspeech.Map, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return mspeech.Map{}, fmt.Errorf("read the file to hear: %w", err)
	}
	if sp, ok := v[string(data)]; ok {
		return speech.Map(sp, stream)
	}
	return talk.Map(ctx, path, stream)
}
