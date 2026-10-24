package world

import (
	"github.com/jatassi/slipstream/fakes/contenttypes/music/coverart"
	"github.com/jatassi/slipstream/fakes/contenttypes/music/musicmeta"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/imagecdn"
	"github.com/jatassi/slipstream/fakes/omdb"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/smtp"
	"github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/fakes/tracker"
)

var fakes = []fake.Spec{musicmeta.Fake, coverart.Fake, tmdb.Fake, omdb.Fake, imagecdn.Fake, tracker.Fake, site.Fake, smtp.Fake}

// Cues are the cues the world's fakes declare, each a name fakeworld cue
// takes.
func Cues() []fake.CueSpec {
	var out []fake.CueSpec
	for _, f := range fakes {
		out = append(out, f.Cues...)
	}
	return out
}

// Captures are the captures the world's fakes declare, each the name of a
// fakeworld reader verb.
func Captures() []fake.CaptureSpec {
	var out []fake.CaptureSpec
	for _, f := range fakes {
		out = append(out, f.Captures...)
	}
	return out
}
