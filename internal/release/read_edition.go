package release

import (
	"strings"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release/internal/upstream"
)

var reportEdition = compile(upstream.RadarrParserReportEditionRegex)

// readEdition ports Radarr's ParseEdition: the first edition text tail
// names after its first character, with dots read as spaces.
func readEdition(tail string, s *scan) Fact[string] {
	m, ok := s.find(reportEdition, regex.TextOf(tail))
	if !ok {
		return Fact[string]{}
	}
	e, _ := m.Group("edition")
	if strings.TrimSpace(e) == "" {
		return Fact[string]{}
	}
	return Known(strings.ReplaceAll(e, ".", " "))
}
