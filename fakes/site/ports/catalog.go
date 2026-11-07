package ports

import (
	"crypto/sha1" //nolint:gosec // an info hash is SHA-1 by the protocol
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/fakes/scenario"
)

// showcase carries real movies, the one exception ADR 0014 makes to
// fictional fake content, so its titles never count as fictional.
const showcase = "showcase"

// Titles are the shared release catalog's fictional titles, each as the
// words before its year: every release name of every catalog Scenario
// but showcase, such as "The Quiet Orchard".
var Titles = sync.OnceValues(func() ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, name := range scenario.Catalog() {
		if name == showcase {
			continue
		}
		s, err := scenario.Load(name)
		if err != nil {
			return nil, fmt.Errorf("read the release catalog: %w", err)
		}
		for _, r := range s.Releases {
			if stem := Stem(r.Name); stem != "" && !seen[stem] {
				seen[stem] = true
				out = append(out, stem)
			}
		}
	}
	slices.Sort(out)
	return out, nil
})

// Fictional reports whether title names one of Titles.
func Fictional(title string) (bool, error) {
	titles, err := Titles()
	if err != nil {
		return false, err
	}
	return slices.Contains(titles, Stem(title)), nil
}

var (
	separators = regexp.MustCompile(`[.\s_]+`)
	// marker is the first word after a release name's title: a year, a
	// season or episode, an episode alone, or a resolution.
	marker = regexp.MustCompile(`(?i)^\(?((19|20)\d\d)\)?$|^s\d{1,3}(e\d{1,4})*$|^e\d{1,4}$|^\d{3,4}[pi]$`)
	tagged = regexp.MustCompile(`^\[[^\]]*\]`)
)

// Stem is title's words before its year, season or resolution, joined by
// spaces: "The.Quiet.Orchard.2021.1080p" and "The Quiet Orchard (2021)"
// are both "The Quiet Orchard". A leading [tag] is dropped.
func Stem(title string) string {
	title = strings.TrimSpace(tagged.ReplaceAllString(strings.TrimSpace(title), ""))
	words := separators.Split(title, -1)
	i := slices.IndexFunc(words, marker.MatchString)
	if i < 0 {
		i = len(words)
	}
	return strings.Join(words[:i], " ")
}

const minted = 4096

// Mint is the fixture catalog's i-th info hash, lowercase hex. The
// rewrite gives each distinct info hash of a recording the next one.
func Mint(i int) string {
	sum := sha1.Sum(fmt.Appendf(nil, "fixture-site port info hash %d", i)) //nolint:gosec // see the import
	return hex.EncodeToString(sum[:])
}

var mintedSet = sync.OnceValue(func() map[string]bool {
	out := make(map[string]bool, minted)
	for i := range minted {
		out[Mint(i)] = true
	}
	return out
})

// Minted reports whether hash, in either case, is one Mint made.
func Minted(hash string) bool { return mintedSet()[strings.ToLower(hash)] }
