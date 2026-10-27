package scenario

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

// Library is what the world creates on the Controller once it is set up,
// through the API as an Admin would.
type Library struct {
	// StorageLevels are the Health levels the world sets before it makes
	// the roots. A process world's Nodes report the host's own disk, so a
	// Scenario whose Storage verdict must not depend on the host sets them.
	StorageLevels *LibraryLevels   `yaml:"storage_levels,omitempty"`
	Roots         []LibraryRoot    `yaml:"roots,omitempty"`
	Targets       []LibraryTarget  `yaml:"targets,omitempty"`
	Profiles      []LibraryProfile `yaml:"profiles,omitempty"`
	Indexers      []LibraryIndexer `yaml:"indexers,omitempty"`
}

// LibraryRoot is a root of a Content Type, the folder named Folder under
// the run's media dir, on the Node named Node, the controller Node when
// empty. A type's first root is its default.
type LibraryRoot struct {
	ContentType string `jsonschema:"required,pattern=^[a-z][a-z0-9]{1\\,15}$"  yaml:"content_type"`
	Folder      string `jsonschema:"required,pattern=^[a-z0-9][a-z0-9._-]*$" yaml:"folder"`
	Node        string `jsonschema:"pattern=^[a-z][a-z0-9-]{0\\,15}$"          yaml:"node,omitempty"`
	// Volume makes the folder a volume of its own with these figures. Only
	// the simulation tier plays it.
	Volume *LibraryVolume `yaml:"volume,omitempty"`
}

type LibraryVolume struct {
	TotalGiB float64 `jsonschema:"required,exclusiveMinimum=0" yaml:"total_gib"`
	FreeGiB  float64 `jsonschema:"required,minimum=0"          yaml:"free_gib"`
}

// LibraryLevels are the Storage levels in bytes: 0B never fires, and a
// level above any disk, such as 1024TiB, always does.
type LibraryLevels struct {
	WarnBelow  Bytes `jsonschema:"required" yaml:"warn_below"`
	ErrorBelow Bytes `jsonschema:"required" yaml:"error_below"`
}

// LibraryTarget is a Target as an Admin creates it: tiers of quality keys,
// best first, the quality whose tier is the cutoff, filters by attribute
// and sizes in MiB per minute for some accepted qualities. One named after
// the starter replaces the starter's rules.
type LibraryTarget struct {
	Name    string          `jsonschema:"required"            yaml:"name"`
	Tiers   [][]QualityKey  `jsonschema:"required,minItems=1" yaml:"tiers"`
	Cutoff  QualityKey      `jsonschema:"required"            yaml:"cutoff"`
	Filters []LibraryFilter `yaml:"filters,omitempty"`
	Sizes   []LibrarySize   `yaml:"sizes,omitempty"`
}

// QualityKey is a release.Quality's key, such as webdl-1080p.
type QualityKey string

type LibraryFilter struct {
	Attribute string   `jsonschema:"required,enum=hdr,enum=video_codec,enum=audio_codec,enum=channels,enum=language" yaml:"attribute"`
	Mode      string   `jsonschema:"required,enum=only,enum=not"                                                       yaml:"mode"`
	Values    []string `jsonschema:"required,minItems=1,uniqueItems=true"                                            yaml:"values"`
}

type LibrarySize struct {
	Quality QualityKey `jsonschema:"required"  yaml:"quality"`
	Min     float64    `jsonschema:"minimum=0" yaml:"min,omitempty"`
	Max     *float64   `jsonschema:"minimum=0" yaml:"max,omitempty"`
}

// Spec is t as targets.NewTarget reads it, so a Scenario can't declare a
// Target the API refuses.
func (t LibraryTarget) Spec() (targets.Spec, error) {
	var errs []error
	quality := func(k QualityKey) release.Quality {
		q, err := release.ParseQuality(string(k))
		errs = append(errs, err)
		return q
	}
	s := targets.Spec{Name: t.Name, Cutoff: quality(t.Cutoff), Sizes: map[release.Quality]targets.Size{}}
	for _, tier := range t.Tiers {
		var qs []release.Quality
		for _, k := range tier {
			qs = append(qs, quality(k))
		}
		s.Tiers = append(s.Tiers, qs)
	}
	for _, f := range t.Filters {
		m, err := targets.ParseMode(f.Mode)
		if err == nil && m == targets.Any {
			err = fmt.Errorf("filter %s is any; leave it out", f.Attribute)
		}
		if err == nil {
			err = s.Filters.Set(targets.Attribute(f.Attribute), m, f.Values)
		}
		errs = append(errs, err)
	}
	for _, z := range t.Sizes {
		q := quality(z.Quality)
		if _, ok := s.Sizes[q]; ok {
			errs = append(errs, fmt.Errorf("a size for %s twice", z.Quality))
		}
		s.Sizes[q] = targets.Size{Min: z.Min, Max: z.Max}
	}
	if err := errors.Join(errs...); err != nil {
		return targets.Spec{}, err
	}
	return s, nil
}

// LibraryProfile is a Profile of Targets in Profile order, each a
// library target or the starter, made its Content Type's default when
// Default is set.
type LibraryProfile struct {
	Name        string   `jsonschema:"required"                                 yaml:"name"`
	ContentType string   `jsonschema:"required,pattern=^[a-z][a-z0-9]{1\\,15}$" yaml:"content_type"`
	Targets     []string `jsonschema:"required,uniqueItems=true"                yaml:"targets"`
	Default     bool     `yaml:"default,omitempty"`
}

// LibraryIndexer is one Indexer an Admin adds, on a site of fixture-site.
// Its address, keys and credentials derive from the seed.
type LibraryIndexer struct {
	Name string `jsonschema:"required"                              yaml:"name"`
	Kind string `jsonschema:"required,enum=catalog,enum=torznab,enum=rss" yaml:"kind"`
	// Site is the fixture-site site's label: torznab for a Torznab feed,
	// brook for an RSS feed, or a catalog site such as larkspur.
	Site string `jsonschema:"required" yaml:"site"`
	// Definition is a catalog Indexer's definition id, the site's label
	// when empty.
	Definition string `yaml:"definition,omitempty"`
	// Home is the Node it searches from. Every Indexer names one when
	// several Nodes hold Searching.
	Home string `yaml:"home,omitempty"`
	// Settings are a catalog Indexer's plain settings by name, beside the
	// credentials the seed gives.
	Settings map[string]string `yaml:"settings,omitempty"`
	// Tracker and Media are an RSS feed's.
	Tracker       string   `yaml:"tracker,omitempty"`
	Media         []string `yaml:"media,omitempty"`
	AllowFailover bool     `yaml:"allow_failover,omitempty"`
}

// FixtureSites are the labels of fixture-site's sites by the kind of
// Indexer each serves.
var FixtureSites = map[string][]string{
	"torznab": {"torznab"},
	"rss":     {"brook"},
	"catalog": {"larkspur", "quarry", "cinder", "marsh", "ferry"},
}

var folderName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

func (l Library) check(add func(format string, args ...any)) {
	l.checkRoots(add)
	l.checkTargets(add)
	l.checkProfiles(add)
	l.checkIndexers(add)
}

func (l Library) checkRoots(add func(format string, args ...any)) {
	var folders []string
	for _, r := range l.Roots {
		switch {
		case !folderName.MatchString(r.Folder):
			add("library root folder %q does not match %s", r.Folder, folderName)
		case slices.Contains(folders, r.Folder):
			add("library has two roots in folder %s", r.Folder)
		}
		folders = append(folders, r.Folder)
		if v := r.Volume; v != nil && (v.TotalGiB <= 0 || v.FreeGiB < 0 || v.FreeGiB > v.TotalGiB) {
			add("library root %s's volume has %g GiB free of %g; want free from 0 to a positive total", r.Folder, v.FreeGiB, v.TotalGiB)
		}
	}
}

func (l Library) checkTargets(add func(format string, args ...any)) {
	var names []string
	for _, t := range l.Targets {
		if slices.Contains(names, t.Name) {
			add("library has two targets named %s", t.Name)
		}
		names = append(names, t.Name)
		s, err := t.Spec()
		if err == nil {
			_, err = targets.NewTarget(s)
		}
		if err != nil {
			add("library target %s: %v", t.Name, strings.ReplaceAll(err.Error(), "\n", "; "))
		}
	}
}

func (l Library) checkProfiles(add func(format string, args ...any)) {
	var names, defaults []string
	for _, p := range l.Profiles {
		if slices.Contains(names, p.Name) {
			add("library has two profiles named %s", p.Name)
		}
		names = append(names, p.Name)
		if p.Default && slices.Contains(defaults, p.ContentType) {
			add("library has two default profiles for content type %s", p.ContentType)
		}
		if p.Default {
			defaults = append(defaults, p.ContentType)
		}
		for _, t := range p.Targets {
			if t != targets.StarterName && !slices.ContainsFunc(l.Targets, func(lt LibraryTarget) bool { return lt.Name == t }) {
				add("library profile %s names target %s, which library.targets does not list and isn't the starter", p.Name, t)
			}
		}
	}
}

func (l Library) checkIndexers(add func(format string, args ...any)) {
	var names []string
	for _, ix := range l.Indexers {
		if slices.Contains(names, ix.Name) {
			add("library has two indexers named %s", ix.Name)
		}
		names = append(names, ix.Name)
		if !slices.Contains(FixtureSites[ix.Kind], ix.Site) {
			add("library indexer %s is a %s Indexer on site %q, which fixture-site doesn't serve as one", ix.Name, ix.Kind, ix.Site)
		}
		if ix.Kind == "rss" && (ix.Tracker == "" || len(ix.Media) == 0) {
			add("library indexer %s is an RSS feed with no tracker or media", ix.Name)
		}
		if ix.Kind != "catalog" && (ix.Definition != "" || ix.Settings != nil) {
			add("library indexer %s takes a definition or settings, which only a catalog Indexer has", ix.Name)
		}
	}
}

func (l Library) contentTypes() []string {
	var out []string
	for _, r := range l.Roots {
		out = append(out, r.ContentType)
	}
	for _, p := range l.Profiles {
		out = append(out, p.ContentType)
	}
	return out
}
