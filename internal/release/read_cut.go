package release

import "strings"

// readCut is the cut whose alias tail names, the longest alias winning,
// then the leftmost, then the first cut in cuts. No alias named is
// Standard.
func readCut(tail string, cuts []Cut) CutKey {
	w := words(tail)
	best, bestLen, bestAt := Standard, 0, 0
	for _, c := range cuts {
		for _, alias := range c.Aliases {
			p := words(alias)
			if strings.TrimSpace(p) == "" {
				continue
			}
			at := strings.Index(w, p)
			if at < 0 {
				continue
			}
			if len(p) > bestLen || len(p) == bestLen && at < bestAt {
				best, bestLen, bestAt = c.Key, len(p), at
			}
		}
	}
	return best
}
