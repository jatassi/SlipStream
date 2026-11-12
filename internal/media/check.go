package media

import (
	"path"
	"slices"
	"strings"
)

// Videos is a set of video file extensions, each lower case without
// the dot. The Controller sends it with every walk and move, so the Node
// reads the Admin's list, never one of its own.
type Videos struct{ exts []string }

// NewVideos is the set of exts, read in any case, with or without a dot.
func NewVideos(exts ...string) Videos {
	v := Videos{exts: make([]string, 0, len(exts))}
	for _, e := range exts {
		v.exts = append(v.exts, strings.ToLower(strings.TrimPrefix(e, ".")))
	}
	slices.Sort(v.exts)
	v.exts = slices.Compact(v.exts)
	return v
}

// DefaultExtensions is the list a library holds until the Admin sets
// one: S22 (Library scan and file probe)'s, in its order.
var DefaultExtensions = []string{"mkv", "mp4", "m4v", "avi", "ts", "m2ts", "mov", "wmv", "webm", "mpg", "mpeg"}

var DefaultVideos = NewVideos(DefaultExtensions...)

// Extensions are v's extensions in byte order.
func (v Videos) Extensions() []string { return slices.Clone(v.exts) }

// Empty is a set of no extensions, which no request may carry.
func (v Videos) Empty() bool { return len(v.exts) == 0 }

// Video reports whether name's extension is one of v's, in any case.
func (v Videos) Video(name string) bool {
	_, found := slices.BinarySearch(v.exts, strings.ToLower(strings.TrimPrefix(path.Ext(name), ".")))
	return found
}

// skippedFolders are NAS and OS clutter, disc structures, and Plex's and
// Jellyfin's sample and bonus folders, in lower case.
var skippedFolders = []string{
	"@eadir", "#recycle", "#snapshot", "$recycle.bin", "system volume information", "lost+found",
	"bdmv", "video_ts",
	"sample", "samples", "extras", "featurettes", "behind the scenes", "deleted scenes", "interviews", "scenes", "shorts", "trailers", "other",
}

// extraSuffixes end the name of a sample or bonus file, before its
// extension.
var extraSuffixes = []string{"-sample", "-trailer", "-featurette", "-behindthescenes", "-deleted", "-interview", "-scene", "-short", "-other"}

// SkipsFolder reports whether a walk passes over the folder named name
// without entering it: a dot name, or one of the clutter, disc and bonus
// folders, in any case.
func SkipsFolder(name string) bool {
	return strings.HasPrefix(name, ".") || slices.Contains(skippedFolders, strings.ToLower(name))
}

// Counts reports whether rel, a '/' path under a root or in a download,
// is a video file a library holds: no folder on its path is skipped, its
// name doesn't start with ".", its extension is one of v's, and it isn't a
// sample or a bonus feature by its name's suffix. Words inside a title
// never count, so "Trailer Park Boys" and "Proof (2005)" are read.
func Counts(rel string, v Videos) bool {
	folders := strings.Split(rel, "/")
	name := strings.ToLower(folders[len(folders)-1])
	if slices.ContainsFunc(folders[:len(folders)-1], SkipsFolder) || strings.HasPrefix(name, ".") {
		return false
	}
	if !v.Video(name) {
		return false
	}
	stem := strings.TrimSuffix(name, path.Ext(name))
	return stem != "sample" && !slices.ContainsFunc(extraSuffixes, func(s string) bool { return strings.HasSuffix(stem, s) })
}
