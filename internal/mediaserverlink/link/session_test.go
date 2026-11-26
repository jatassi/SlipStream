package link_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// TestAPlaysDecisionIsWhatItsStreamsDo checks every stream combination:
// any transcoded stream or burned subtitle is a transcode, else a new
// container is a direct stream, else it is a direct play.
func TestAPlaysDecisionIsWhatItsStreamsDo(t *testing.T) {
	for _, s := range everyStreams() {
		want := link.DecisionDirectPlay
		switch {
		case s.Video == "transcode" || s.Audio == "transcode" || s.Subtitles == "burned":
			want = link.DecisionTranscode
		case s.ContainerChanged:
			want = link.DecisionDirectStream
		}
		if got := s.Decision(); got != want {
			t.Errorf("%+v decides %s, want %s", s, got, want)
		}
	}
}

func everyStreams() []link.Streams {
	var all []link.Streams
	for _, video := range link.StreamDecisions() {
		for _, audio := range link.StreamDecisions() {
			for _, subtitles := range link.SubtitleDecisions() {
				for _, remuxed := range []bool{false, true} {
					all = append(all, link.Streams{Video: video, Audio: audio, Subtitles: subtitles, ContainerChanged: remuxed})
				}
			}
		}
	}
	return all
}

func TestStreamDecisionsParseStrictly(t *testing.T) {
	for _, c := range []struct {
		word string
		ok   bool
	}{{"direct", true}, {"transcode", true}, {"none", true}, {"", false}, {"copy", false}, {"Direct", false}} {
		if _, err := link.ParseStreamDecision(c.word); (err == nil) != c.ok {
			t.Errorf("ParseStreamDecision(%q) err = %v, want ok %v", c.word, err, c.ok)
		}
	}
	for _, c := range []struct {
		word string
		ok   bool
	}{{"none", true}, {"direct", true}, {"burned", true}, {"", false}, {"burn", false}, {"transcode", false}} {
		if _, err := link.ParseSubtitleDecision(c.word); (err == nil) != c.ok {
			t.Errorf("ParseSubtitleDecision(%q) err = %v, want ok %v", c.word, err, c.ok)
		}
	}
}
