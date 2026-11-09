package release

import (
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release/internal/upstream"
)

// The revision patterns are Sonarr's: its VersionRegex reads every
// version Radarr's does, and a version after the resolution besides.
var (
	versionRegex = compile(upstream.SonarrQualityParserVersionRegex)
	properRegex  = compile(upstream.SonarrQualityParserProperRegex)
	repackRegex  = compile(upstream.SonarrQualityParserRepackRegex)
	realRegex    = compile(upstream.SonarrQualityParserRealRegex)
)

// readRevision is upstream's ParseQualityModifiers: vN sets the version,
// PROPER, REPACK and RERIP raise it by one, and REAL is counted, in
// capitals, in the name as given.
func readRevision(p Prepared) Revision {
	norm := regex.TextOf(strings.TrimSpace(strings.ReplaceAll(p.Original, "_", " ")))
	r := Revision{Version: 1}
	raised := 2
	if m, ok := versionRegex.find(norm); ok {
		v, _ := m.Group("version")
		r.Version, _ = strconv.Atoi(v)
		raised = r.Version + 1
	}
	if properRegex.match(norm) {
		r.Version = raised
	}
	if repackRegex.match(norm) {
		r.Version = raised
		r.Repack = true
	}
	r.Real = len(realRegex.findAll(regex.TextOf(p.Original)))
	return r
}
