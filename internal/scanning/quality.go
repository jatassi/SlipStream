package scanning

import (
	"strings"

	"github.com/jatassi/slipstream/internal/release"
)

// QualityOf is a found file's quality and cut: release.File over its file
// name at its probed resolution, with the source its nearest folder names
// when the file name names none. The cut comes from {edition-<cut>} or a
// cut's aliases in the file name. rel is root-relative. S23 rewrites this
// one function.
func QualityOf(rel string, cuts []release.Cut, probed release.Resolution) (release.Quality, release.CutKey) {
	folders := strings.Split(rel, "/")
	name := folders[len(folders)-1]
	facts := release.File(name, cuts, probed)
	q, _ := facts.Quality.Get()
	if _, named := release.SourceOf(name).Get(); named {
		return q, facts.Cut
	}
	for i := len(folders) - 2; i >= 0; i-- {
		if source, named := release.SourceOf(folders[i]).Get(); named {
			q, _ = release.QualityOf(release.Known(source), release.Known(probed))
			break
		}
	}
	return q, facts.Cut
}

// Requalified is a recorded file's quality after a reprobe: q while its
// resolution class holds, else the same source at the probed class when
// the quality table has that pair, else QualityOf over its path.
func Requalified(q release.Quality, rel string, cuts []release.Cut, probed release.Resolution) release.Quality {
	source := q.Info().Source
	if at, _ := release.QualityOf(release.Known(source), release.Known(probed)); at.Info().Source == source {
		return at
	}
	at, _ := QualityOf(rel, cuts, probed)
	return at
}
