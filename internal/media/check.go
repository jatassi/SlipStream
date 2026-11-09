package media

import (
	"path"
	"slices"
	"strings"
)

// Extensions are the extensions, lower case without the dot, of a video
// file a library or a download holds. S33 (Import pipeline) turns this
// list into a setting.
var Extensions = []string{"mkv", "mp4", "m4v", "avi", "ts", "m2ts", "mov", "wmv", "webm", "mpg", "mpeg"}

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
// name doesn't start with ".", its extension is a video's, and it isn't a
// sample or a bonus feature by its name's suffix. Words inside a title
// never count, so "Trailer Park Boys" and "Proof (2005)" are read.
func Counts(rel string) bool {
	folders := strings.Split(rel, "/")
	name := strings.ToLower(folders[len(folders)-1])
	if slices.ContainsFunc(folders[:len(folders)-1], SkipsFolder) || strings.HasPrefix(name, ".") {
		return false
	}
	ext := path.Ext(name)
	if !slices.Contains(Extensions, strings.TrimPrefix(ext, ".")) {
		return false
	}
	stem := strings.TrimSuffix(name, ext)
	return stem != "sample" && !slices.ContainsFunc(extraSuffixes, func(s string) bool { return strings.HasSuffix(stem, s) })
}
