package scanning

import (
	"path"
	"strings"

	"github.com/jatassi/slipstream/internal/release"
)

// QualityOf reads a found file by its root-relative path: release.File's
// reading of it, and its quality at its probed resolution. Its source is
// the one its file name's words name, else its nearest folder's, else its
// extension's, as Sonarr augments a file's quality from its folder's.
func QualityOf(rel string, names release.Names, cuts []release.Cut, probed release.Resolution) (release.Read, release.Quality) {
	read := release.File(rel, names, cuts)
	facts := read.Facts
	folders := strings.Split(rel, "/")
	file := folders[len(folders)-1]
	if !namesSource(release.Title(strings.TrimSuffix(file, path.Ext(file)), names, nil).Facts) {
		for i := len(folders) - 2; i >= 0; i-- {
			if f := release.Title(folders[i], names, nil).Facts; namesSource(f) {
				facts.Source, facts.Modifier = f.Source, f.Modifier
				break
			}
		}
	}
	q, _ := facts.WithProbe(release.Probed{Resolution: release.Known(probed)}).Quality.Get()
	return read, q
}

func namesSource(f release.Facts) bool {
	_, ok := f.Source.Get()
	return ok
}

// Requalified is a recorded file's quality after a reprobe: q while its
// resolution class holds, else the same source at the probed class when
// the quality table has that pair, else QualityOf over its path.
func Requalified(q release.Quality, rel string, names release.Names, cuts []release.Cut, probed release.Resolution) release.Quality {
	source := q.Info().Source
	if at, _ := release.QualityOf(release.Known(source), release.Known(probed)); at.Info().Source == source {
		return at
	}
	_, at := QualityOf(rel, names, cuts, probed)
	return at
}
