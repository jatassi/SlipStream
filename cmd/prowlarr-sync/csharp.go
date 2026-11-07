package main

import (
	"regexp"
	"slices"
	"strings"
)

const (
	definitionsDir = "src/NzbDrone.Core/Indexers/Definitions/"
	hiddenListFile = "src/NzbDrone.Core/IndexerVersions/IndexerDefinitionUpdateService.cs"
	torrentBase    = "TorrentIndexerBase"
)

// class is one C# class declaration that derives from another.
type class struct {
	Name, Parent, File string
	Abstract           bool
	// Obsolete is the [Obsolete] message, and IsObsolete whether the
	// attribute is there at all.
	Obsolete   string
	IsObsolete bool
}

var (
	classLine    = regexp.MustCompile(`^((?:(?:public|internal|abstract|sealed|partial)\s+)*)class\s+(\w+)[^:{]*:\s*(\w+)`)
	obsoleteLine = regexp.MustCompile(`^\[Obsolete(?:\(\s*"((?:[^"\\]|\\.)*)")?`)
	categoryUse  = regexp.MustCompile(`NewznabStandardCategory\.(\w+)`)
	quoted       = regexp.MustCompile(`"([^"]+)"`)
	movedWords   = regexp.MustCompile(`(?i)\b(ya?ml|cardigann)\b`)
)

// classes are the derived classes file declares, with the attributes on
// the lines just above each.
func classes(file string, src []byte) []class {
	var out []class
	var attributes []string
	for line := range strings.Lines(string(src)) {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			attributes = append(attributes, line)
			continue
		}
		if m := classLine.FindStringSubmatch(line); m != nil {
			c := class{Name: m[2], Parent: m[3], File: file, Abstract: strings.Contains(m[1], "abstract")}
			for _, a := range attributes {
				if o := obsoleteLine.FindStringSubmatch(a); o != nil {
					c.IsObsolete, c.Obsolete = true, o[1]
				}
			}
			out = append(out, c)
		}
		if line != "" && !strings.HasPrefix(line, "//") {
			attributes = nil
		}
	}
	return out
}

// sites are the concrete classes that derive, through any chain of
// abstract bases, from TorrentIndexerBase, sorted by name.
func sites(all []class) []class {
	bases := map[string]bool{torrentBase: true}
	for grew := true; grew; {
		grew = false
		for _, c := range all {
			if c.Abstract && bases[c.Parent] && !bases[c.Name] {
				bases[c.Name], grew = true, true
			}
		}
	}
	var out []class
	for _, c := range all {
		if !c.Abstract && bases[c.Parent] {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b class) int { return strings.Compare(a.Name, b.Name) })
	return out
}

// media maps a NewznabStandardCategory member, by the longest prefix
// that names it, to the media it carries, in Newznab's words.
var media = []struct{ prefix, media string }{
	{"AudioAudiobook", "books"},
	{"PCGames", "games"},
	{"TVAnime", "anime"},
	{"Console", "games"},
	{"Movies", "movies"},
	{"Audio", "audio"},
	{"Books", "books"},
	{"Other", "other"},
	{"PC", "apps"},
	{"TV", "tv"},
	{"XXX", "adult"},
}

var mediaOrder = []string{"movies", "tv", "anime", "audio", "books", "games", "apps", "adult", "other"}

// mediaOf is the media srcs' AddCategoryMapping lines carry, in
// mediaOrder.
func mediaOf(srcs ...[]byte) []string {
	found := map[string]bool{}
	for _, src := range srcs {
		for line := range strings.Lines(string(src)) {
			if strings.Contains(line, "AddCategoryMapping") {
				for _, m := range categoryUse.FindAllStringSubmatch(line, -1) {
					found[mediaOfCategory(m[1])] = true
				}
			}
		}
	}
	var out []string
	for _, m := range mediaOrder {
		if found[m] {
			out = append(out, m)
		}
	}
	return out
}

func mediaOfCategory(member string) string {
	for _, x := range media {
		if strings.HasPrefix(member, x.prefix) {
			return x.media
		}
	}
	return ""
}

// hiddenList is the ids in _definitionBlocklist, sorted.
func hiddenList(src []byte) []string {
	_, rest, ok := strings.Cut(string(src), "_definitionBlocklist")
	if !ok {
		return nil
	}
	body, _, _ := strings.Cut(rest, "};")
	var ids []string
	for _, m := range quoted.FindAllStringSubmatch(body, -1) {
		ids = append(ids, m[1])
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}
