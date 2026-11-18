package subtitles_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/subtitles"
)

func TestParseTail(t *testing.T) {
	for _, c := range []struct {
		tail string
		want string
		ok   bool
	}{
		{"en.srt", "en.srt", true},
		{"en.hi.srt", "en.sdh.srt", true},
		{"hi.srt", "hi.srt", true},
		{"en.sdh.srt", "en.sdh.srt", true},
		{"en.cc.ass", "en.sdh.ass", true},
		{"EN.Forced.SRT", "en.forced.srt", true},
		{"pt-BR.foreign.ssa", "pt-BR.forced.ssa", true},
		{"en.forced.sdh.srt", "en.forced.sdh.srt", true},
		{"en.default.srt", "en.srt", true},
		{"Release.Group.fr.vtt", "fr.vtt", true},
		{"eng.srt", "en.srt", true},
		{"en.sub", "", false},
		{"srt", "", false},
		{"forced.srt", "", false},
		{"commentary.srt", "", false},
	} {
		got, ok := subtitles.ParseTail(c.tail)
		if ok != c.ok || ok && got.String() != c.want {
			t.Errorf("ParseTail(%q) = %v, %t, want %q, %t", c.tail, got, ok, c.want, c.ok)
		}
	}
}

func TestParseWantReadsOnlyItsOwnForm(t *testing.T) {
	for _, s := range []string{"en", "pt-BR", "en-forced", "pt-BR-forced"} {
		w, err := subtitles.ParseWant(s)
		if err != nil || w.String() != s {
			t.Errorf("ParseWant(%q) = %v, %v", s, w, err)
		}
	}
	for _, s := range []string{"eng", "EN", "en-Forced", "forced", ""} {
		if w, err := subtitles.ParseWant(s); err == nil {
			t.Errorf("ParseWant(%q) = %v, want a refusal", s, w)
		}
	}
}
