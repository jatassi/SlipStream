package world

import (
	"slices"

	"github.com/jatassi/slipstream/fakes/capturescript"
	"github.com/jatassi/slipstream/fakes/catalog"
	"github.com/jatassi/slipstream/fakes/challengesolver"
	"github.com/jatassi/slipstream/fakes/contenttypes/music/coverart"
	"github.com/jatassi/slipstream/fakes/contenttypes/music/musicmeta"
	"github.com/jatassi/slipstream/fakes/deluge"
	"github.com/jatassi/slipstream/fakes/dht"
	fakedns "github.com/jatassi/slipstream/fakes/dns"
	"github.com/jatassi/slipstream/fakes/emby"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/gateway"
	"github.com/jatassi/slipstream/fakes/gestdown"
	"github.com/jatassi/slipstream/fakes/imagecdn"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/fakes/notifysink"
	"github.com/jatassi/slipstream/fakes/omdb"
	"github.com/jatassi/slipstream/fakes/outside"
	"github.com/jatassi/slipstream/fakes/plex"
	"github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/fakes/proxy"
	"github.com/jatassi/slipstream/fakes/qbittorrent"
	"github.com/jatassi/slipstream/fakes/rtorrent"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/smtp"
	"github.com/jatassi/slipstream/fakes/stun"
	"github.com/jatassi/slipstream/fakes/subdl"
	"github.com/jatassi/slipstream/fakes/subsource"
	"github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/transmission"
	"github.com/jatassi/slipstream/fakes/tvdb"
)

var fakes = []fake.Spec{musicmeta.Fake, coverart.Fake, tmdb.Fake, tvdb.Fake, omdb.Fake, subdl.Fake, subdl.DL, subsource.Fake, gestdown.Fake, imagecdn.Fake, tracker.Fake, tracker.UDP, dht.Fake, site.Fake, catalog.Fake, challengesolver.Fake, smtp.Fake, qbittorrent.Fake, transmission.Fake, deluge.Fake, rtorrent.Fake, plex.Fake, jellyfin.Fake, emby.Fake, plextv.Fake, plextv.Auth, plextv.Web, capturescript.Fake, notifysink.Fake, stun.Fake, stun.Second, gateway.Fake, gateway.IGD, fakedns.Fake, outside.Fake, proxy.Fake}

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

// FakeNamed is the world's fake named name.
func FakeNamed(name string) (fake.Spec, bool) {
	i := slices.IndexFunc(fakes, func(f fake.Spec) bool { return f.Name == name })
	if i < 0 {
		return fake.Spec{}, false
	}
	return fakes[i], true
}
