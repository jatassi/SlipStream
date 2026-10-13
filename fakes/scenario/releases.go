package scenario

import (
	"regexp"
	"slices"
	"time"
)

// Release is one release the fakes offer. The release catalog makes its
// payload video and metainfo, and fixture-seeders seed it when Seeded.
type Release struct {
	Name    string `jsonschema:"required,pattern=^[A-Za-z0-9][A-Za-z0-9._-]*$" yaml:"name"`
	TMDBID  int    `jsonschema:"required,minimum=1"                           yaml:"tmdb_id"`
	Video   Video  `jsonschema:"required"                                     yaml:"video"`
	Seeders int    `jsonschema:"minimum=0"                                    yaml:"seeders,omitempty"`
	Seeded  bool   `yaml:"seeded,omitempty"`
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
}

var releaseName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func checkReleases(add func(format string, args ...any), releases []Release) {
	for i, r := range releases {
		if !releaseName.MatchString(r.Name) {
			add("release name %q does not match %s", r.Name, releaseName)
		}
		if slices.ContainsFunc(releases[:i], func(o Release) bool { return o.Name == r.Name }) {
			add("release %s is listed twice", r.Name)
		}
		if r.TMDBID < 1 {
			add("release %s has tmdb_id %d; it is at least 1", r.Name, r.TMDBID)
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
	}
}

func (t Tracker) check(add func(format string, args ...any)) {
	if t.Interval < 0 {
		add("tracker interval %s is negative", t.Interval)
	}
}
