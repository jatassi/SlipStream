package scenario

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Release is one release the fakes offer. The release catalog makes its
// payload video and metainfo, and fixture-seeders seed it when Seeded.
type Release struct {
	Name    string `jsonschema:"required,pattern=^[A-Za-z0-9][A-Za-z0-9._-]*$" yaml:"name"`
	TMDBID  int64  `jsonschema:"required,minimum=1"                           yaml:"tmdb_id"`
	Video   Video  `jsonschema:"required"                                     yaml:"video"`
	Seeders int    `jsonschema:"minimum=0"                                    yaml:"seeders,omitempty"`
	Seeded  bool   `yaml:"seeded,omitempty"`
	// AdvertisedSize is the size fixture-site lists, separate from the
	// payload's. Zero lists the payload's.
	AdvertisedSize Bytes `yaml:"advertised_size,omitempty"`
	// Unlisted keeps it out of fixture-site's feeds until the
	// fixture-site-list cue lists it, as a release posted later.
	Unlisted bool `yaml:"unlisted,omitempty"`
	// Unregistered makes fake-tracker answer its announces with UNIT3D's
	// "Unregistered torrent", as for a torrent the tracker deleted.
	Unregistered bool `yaml:"unregistered,omitempty"`
	// BadPeer adds a peer that misbehaves this way, which fake-tracker
	// lists beside the seeders.
	BadPeer BadPeer `jsonschema:"enum=corrupt,enum=stall,enum=choke,enum=drop" yaml:"bad_peer,omitempty"`
}

// BadPeer is how a release's bad peer misbehaves once a leecher asks it
// for pieces. It always claims to hold every piece.
type BadPeer string

const (
	// Corrupt sends blocks that are not the torrent's, so their pieces fail
	// their hash check.
	Corrupt BadPeer = "corrupt"
	// Stall unchokes the leecher and never sends a block it asks for.
	Stall BadPeer = "stall"
	// Choke never unchokes the leecher.
	Choke BadPeer = "choke"
	// Drop hangs up once it has said which pieces it holds.
	Drop BadPeer = "drop"
)

// Bytes is a size in bytes, written as a number and a unit, like 8.2GiB.
type Bytes int64

var units = []struct {
	name string
	size int64
}{{"TiB", 1 << 40}, {"GiB", 1 << 30}, {"MiB", 1 << 20}, {"KiB", 1 << 10}, {"B", 1}}

const bytesPattern = `^\d+(\.\d+)?(TiB|GiB|MiB|KiB|B)$`

var bytesForm = regexp.MustCompile(bytesPattern)

func (b *Bytes) UnmarshalYAML(n *yaml.Node) error {
	m := bytesForm.FindStringSubmatch(n.Value)
	if n.Kind != yaml.ScalarNode || m == nil {
		return &yaml.TypeError{Errors: []string{fmt.Sprintf("line %d: size %q is not a number and a unit such as 8.2GiB", n.Line, n.Value)}}
	}
	v, err := strconv.ParseFloat(strings.TrimSuffix(n.Value, m[2]), 64)
	if err != nil {
		return fmt.Errorf("line %d: size %q: %w", n.Line, n.Value, err)
	}
	for _, u := range units {
		if u.name == m[2] {
			*b = Bytes(math.Round(v * float64(u.size)))
		}
	}
	return nil
}

// MarshalYAML writes b in the largest unit it fills, with the fewest
// decimals that read back as b.
func (b Bytes) MarshalYAML() (any, error) {
	u := units[len(units)-1]
	for _, larger := range units {
		if int64(b) >= larger.size {
			u = larger
			break
		}
	}
	v := float64(b) / float64(u.size)
	for prec := 0; ; prec++ {
		s := strconv.FormatFloat(v, 'f', prec, 64)
		back, _ := strconv.ParseFloat(s, 64)
		if Bytes(math.Round(back*float64(u.size))) == b {
			return s + u.name, nil
		}
	}
}

// Video is a release's payload: ffmpeg's testsrc2 pattern at Width by
// Height for Seconds, encoded as H.264 in Matroska.
type Video struct {
	Width   int `jsonschema:"required,minimum=2,maximum=7680,multipleOf=2" yaml:"width"`
	Height  int `jsonschema:"required,minimum=2,maximum=4320,multipleOf=2" yaml:"height"`
	Seconds int `jsonschema:"required,minimum=1,maximum=60"                yaml:"seconds"`
}

// Tracker is fake-tracker's behavior.
type Tracker struct {
	// Interval is the announce interval it answers. Zero is the production
	// value, 30s.
	Interval time.Duration `yaml:"interval,omitempty"`
	// MinInterval is the min interval it answers, and in unit3d mode the
	// window's base. Zero is Interval.
	MinInterval time.Duration `yaml:"min_interval,omitempty"`
	Mode        TrackerMode   `jsonschema:"enum=plain,enum=unit3d,enum=gazelle" yaml:"mode,omitempty"`
	// AllowPeerIDs are the peer ID prefixes gazelle mode accepts. Empty is
	// libtorrent's, -LT.
	AllowPeerIDs []string `yaml:"allow_peer_ids,omitempty"`
	// FailWithoutInterval answers failures with no interval or min interval.
	FailWithoutInterval bool `yaml:"fail_without_interval,omitempty"`
}

// TrackerMode is the private tracker software fake-tracker plays.
type TrackerMode string

const (
	// Plain accepts every announce for a known passkey and torrent.
	Plain TrackerMode = "plain"
	// UNIT3D rejects announces inside UNIT3D's min interval windows.
	UNIT3D TrackerMode = "unit3d"
	// Gazelle rejects peer IDs whose prefix is not on AllowPeerIDs.
	Gazelle TrackerMode = "gazelle"
)

var releaseName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func checkReleases(add func(format string, args ...any), releases []Release, movies *TMDB) {
	for i, r := range releases {
		if !releaseName.MatchString(r.Name) {
			add("release name %q does not match %s", r.Name, releaseName)
		}
		if slices.ContainsFunc(releases[:i], func(o Release) bool { return o.Name == r.Name }) {
			add("release %s is listed twice", r.Name)
		}
		if r.TMDBID < 1 {
			add("release %s has tmdb_id %d; it is at least 1", r.Name, r.TMDBID)
		} else if movies == nil || !slices.ContainsFunc(movies.Movies, func(m TMDBMovie) bool { return m.ID == r.TMDBID }) {
			add("release %s names tmdb movie %d, which tmdb does not list", r.Name, r.TMDBID)
		}
		v := r.Video
		if v.Width < 2 || v.Width > 7680 || v.Height < 2 || v.Height > 4320 || v.Width%2 != 0 || v.Height%2 != 0 {
			add("release %s video is %dx%d; width and height are even, up to 7680x4320", r.Name, v.Width, v.Height)
		}
		if v.Seconds < 1 || v.Seconds > 60 {
			add("release %s video lasts %d seconds; it lasts 1 to 60", r.Name, v.Seconds)
		}
		if r.Seeders < 0 {
			add("release %s has %d seeders", r.Name, r.Seeders)
		}
		if r.AdvertisedSize < 0 {
			add("release %s advertises %d bytes", r.Name, r.AdvertisedSize)
		}
		switch r.BadPeer {
		case "", Corrupt, Stall, Choke, Drop:
		default:
			add("release %s bad_peer %q is not corrupt, stall, choke or drop", r.Name, r.BadPeer)
		}
	}
}

func (t Tracker) check(add func(format string, args ...any)) {
	if t.Interval < 0 {
		add("tracker interval %s is negative", t.Interval)
	}
	if t.MinInterval < 0 {
		add("tracker min_interval %s is negative", t.MinInterval)
	}
	if t.Interval > 0 && t.MinInterval > t.Interval {
		add("tracker min_interval %s is above its interval %s", t.MinInterval, t.Interval)
	}
	switch t.Mode {
	case "", Plain, UNIT3D, Gazelle:
	default:
		add("tracker mode %q is not plain, unit3d or gazelle", t.Mode)
	}
	if len(t.AllowPeerIDs) > 0 && t.Mode != Gazelle {
		add("tracker allow_peer_ids is for gazelle mode, not %q", t.Mode)
	}
	if slices.Contains(t.AllowPeerIDs, "") {
		add("tracker allow_peer_ids lists an empty prefix, which allows every peer ID")
	}
}
