package release

import (
	pathpkg "path"
	"strings"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release/internal/upstream"
)

// Title reads a release title: the type's Name, then every core fact.
// Quality, revision, the stream facts and id tags read the whole name;
// the group reads Name.Masked; languages, the cut and the edition text
// read Name.Tail.
func Title(name string, names Names, cuts []Cut) Read {
	return read(Prepare(name), names, cuts)
}

func read(p Prepared, names Names, cuts []Cut) Read {
	var n Name
	if !p.Obfuscated {
		n = names.ReadName(p)
	}
	s := &scan{slow: p.slow}
	tail := n.Tail
	if tail == "" {
		tail = p.Title.String()
	}
	masked := n.Masked
	if masked == "" {
		masked = p.Release.String()
	}
	f := Facts{
		Revision:  readRevision(p, s),
		Group:     readGroup(masked, s),
		Cut:       readCut(tail, cuts),
		Edition:   readEdition(tail, s),
		Languages: readLanguages(tail),
		IDs:       readIDs(p),
	}
	readQuality(p, s, &f)
	readStream(p, &f)
	return Read{Given: p.Original, Name: n, Facts: f, Slow: s.slow}
}

// File reads a file path's name, and its folder's when the file name
// gives no title and no numbering, in Sonarr's ParsePath order: the file
// name, which the type may read with its folder's, then the folder's name
// and the file's together, then the folder's name with the file's
// extension. A path no try names reads as its file name. It reads title
// facts only: a file's probe joins with Facts.WithProbe.
func File(path string, names Names, cuts []Cut) Read {
	folder, file := splitPath(path)
	var s scan
	t := regex.TextOf(file)
	for _, sub := range pathPreSubstitutions {
		t = s.replace(sub, t)
	}
	p := Prepare(t.String())
	p.Folder = folder
	p.slow = append(s.slow, p.slow...)
	r := read(p, names, cuts)
	if folder == "" || readsAny(r.Name) {
		return r
	}
	for _, tried := range []string{folder + " " + p.Original, folder + pathpkg.Ext(p.Original)} {
		if f := Title(tried, names, cuts); readsAny(f.Name) {
			return f
		}
	}
	return r
}

var pathPreSubstitutions = compileReplaces(upstream.SonarrParserCommonPathPreSubstitutionRegex)

// splitPath is path's file name and the name of the folder holding it,
// with either separator, as a path may come from another OS.
func splitPath(path string) (folder, file string) {
	dir, file := splitLast(path)
	_, folder = splitLast(dir)
	return folder, file
}

func splitLast(path string) (head, tail string) {
	i := strings.LastIndexAny(path, `/\`)
	return path[:max(i, 0)], path[i+1:]
}

func readsAny(n Name) bool { return len(n.Titles) > 0 || len(n.Marks) > 0 }

// QualityIn is the quality a name reads as, for display callers that
// hold no Content Type.
func QualityIn(name string) Fact[Quality] {
	return Title(name, TitleOnly{}, nil).Facts.Quality
}
