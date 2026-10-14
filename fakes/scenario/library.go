package scenario

import (
	"regexp"
	"slices"

	"github.com/jatassi/slipstream/internal/targets"
)

// Library is what the world creates on the Controller once it is set up,
// through the API as an Admin would.
type Library struct {
	Roots    []LibraryRoot    `yaml:"roots,omitempty"`
	Targets  []LibraryTarget  `yaml:"targets,omitempty"`
	Profiles []LibraryProfile `yaml:"profiles,omitempty"`
	Indexers []LibraryIndexer `yaml:"indexers,omitempty"`
}

// LibraryRoot is a Content Type's root, the folder named Folder under the
// run's media dir.
type LibraryRoot struct {
	ContentType string `jsonschema:"required,pattern=^[a-z][a-z0-9]{1\\,15}$"  yaml:"content_type"`
	Folder      string `jsonschema:"required,pattern=^[a-z0-9][a-z0-9._-]*$" yaml:"folder"`
}

type LibraryTarget struct {
	Name        string   `jsonschema:"required"                                                                          yaml:"name"`
	Resolutions []string `jsonschema:"required,minItems=1,uniqueItems=true,enum=2160p,enum=1080p,enum=720p,enum=576p,enum=480p" yaml:"resolutions"`
}

type LibraryProfile struct {
	Name        string   `jsonschema:"required"                                 yaml:"name"`
	ContentType string   `jsonschema:"required,pattern=^[a-z][a-z0-9]{1\\,15}$" yaml:"content_type"`
	Targets     []string `jsonschema:"required,minItems=1,uniqueItems=true"     yaml:"targets"`
}

// LibraryIndexer is a Torznab Indexer on the named site fake, with the
// site's URL and API key.
type LibraryIndexer struct {
	Name string `jsonschema:"required"                   yaml:"name"`
	Site string `jsonschema:"required,enum=fixture-site" yaml:"site"`
}

var folderName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

func (l Library) check(add func(format string, args ...any)) {
	l.checkRoots(add)
	l.checkTargets(add)
	l.checkProfiles(add)
	l.checkIndexers(add)
}

func (l Library) checkRoots(add func(format string, args ...any)) {
	var types, folders []string
	for _, r := range l.Roots {
		if slices.Contains(types, r.ContentType) {
			add("library has two roots for content type %s", r.ContentType)
		}
		switch {
		case !folderName.MatchString(r.Folder):
			add("library root folder %q does not match %s", r.Folder, folderName)
		case slices.Contains(folders, r.Folder):
			add("library has two roots in folder %s", r.Folder)
		}
		types, folders = append(types, r.ContentType), append(folders, r.Folder)
	}
}

func (l Library) checkTargets(add func(format string, args ...any)) {
	var names []string
	for _, t := range l.Targets {
		if slices.Contains(names, t.Name) {
			add("library has two targets named %s", t.Name)
		}
		names = append(names, t.Name)
		if len(t.Resolutions) == 0 {
			add("library target %s has no resolutions", t.Name)
		}
		for _, r := range t.Resolutions {
			if _, err := targets.ParseResolution(r); err != nil {
				add("library target %s: %v", t.Name, err)
			}
		}
	}
}

func (l Library) checkProfiles(add func(format string, args ...any)) {
	var names []string
	for _, p := range l.Profiles {
		if slices.Contains(names, p.Name) {
			add("library has two profiles named %s", p.Name)
		}
		names = append(names, p.Name)
		if len(p.Targets) == 0 {
			add("library profile %s has no targets", p.Name)
		}
		for _, t := range p.Targets {
			if !slices.ContainsFunc(l.Targets, func(lt LibraryTarget) bool { return lt.Name == t }) {
				add("library profile %s names target %s, which library.targets does not list", p.Name, t)
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
		if ix.Site != "fixture-site" {
			add("library indexer %s is on site %q, which no fake serves", ix.Name, ix.Site)
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
