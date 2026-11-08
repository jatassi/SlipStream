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

// Fictional reports whether title names one of Titles, by its stem or
// by either side of a lone dash in it.
func Fictional(title string) (bool, error) {
	titles, err := Titles()
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(Stems(title), func(stem string) bool { return slices.Contains(titles, stem) }), nil
}

var (
	separators = regexp.MustCompile(`[.\s_]+`)
	// marker is the first word after a release name's title: a year, a
	// season or episode, an episode alone, a resolution, a [tag] or
	// (note), the "by" before a book's author, or a lone slash before
	// another name.
	marker = regexp.MustCompile(`(?i)^\(?((19|20)\d\d)\)?$|^s\d{1,3}(e\d{1,4})*$|^e\d{1,4}$|^\d{3,4}[pi]$|^[\[(]|^by$|^/$`)
	tagged = regexp.MustCompile(`^\[[^\]]*\]`)
	// localName is a Russian or Ukrainian site's own-language name before
	// the English one: "Тихий сад / The Quiet Orchard".
	localName = regexp.MustCompile(`^[^A-Za-z]+ / `)
	// animeEpisode follows a lone dash in an anime release's name, as in
	// "[SubsPlease] Harbor Lights - 03 (1080p)".
	animeEpisode = regexp.MustCompile(`(?i)^(\d{1,4}(v\d)?|movie)$`)
)

// Stem is title's words before its year, season, resolution or anime
// episode, joined by spaces: "The.Quiet.Orchard.2021.1080p", "The Quiet
// Orchard (2021)", "[SubsPlease] The Quiet Orchard - 01 (1080p)" and
// "Тихий сад / The Quiet Orchard [2021, WEB-DL]" are all "The Quiet
// Orchard". A leading [tag] is dropped.
func Stem(title string) string {
	title = strings.TrimSpace(tagged.ReplaceAllString(strings.TrimSpace(title), ""))
	title = localName.ReplaceAllString(title, "")
	words := separators.Split(title, -1)
	i := slices.IndexFunc(words, marker.MatchString)
	if i < 0 {
		i = len(words)
	}
	for j := range i {
		if words[j] == "-" && j+1 < len(words) && animeEpisode.MatchString(words[j+1]) {
			i = j
			break
		}
	}
	return strings.Join(words[:i], " ")
}

// Stems is title's stem, then each side of its first lone dash, so
// "Juniper & Vale - The Quiet Orchard (2021) [FLAC]" names its album and
// "Salt Meadow - Quiet Orchard [2021] FLAC" its artist.
func Stems(title string) []string {
	stems := []string{Stem(title)}
	if left, right, ok := strings.Cut(title, " - "); ok {
		stems = append(stems, Stem(left), Stem(right))
	}
	return stems
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
