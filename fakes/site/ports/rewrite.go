package ports

import (
	"fmt"
	"html"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// Rewriter turns a recording, Prowlarr's or the record tool's, into what
// a fixture may hold. Its mappings hold across every body it rewrites, so
// one release keeps one title and one info hash.
type Rewriter struct {
	// Host is the fixture host each of Hosts becomes.
	Host  string
	Hosts []string
	// Replace maps a literal, such as a recorded passkey, to what takes
	// its place, such as "{{passkey}}". It runs first.
	Replace map[string]string
	// Titles are titles the body holds that aren't dotted release names,
	// such as "24 S03E12 720p WEBRip h264-DRAWER".
	Titles []string

	hashes map[string]string
	stems  map[string]string
}

// Rewrite is body with Replace applied, every info hash swapped for the
// next one Mint makes, every title's words before its year for the next
// fictional title, every username for FixtureUser and every host for
// Host.
func (r *Rewriter) Rewrite(body []byte) ([]byte, error) {
	titles, err := Titles()
	if err != nil {
		return nil, err
	}
	if r.hashes == nil {
		r.hashes, r.stems = map[string]string{}, map[string]string{}
	}
	s := string(body)
	for _, k := range slices.Sorted(maps.Keys(r.Replace)) {
		s = strings.ReplaceAll(s, k, r.Replace[k])
	}
	s = hexHash.ReplaceAllStringFunc(s, r.hash)
	s = userKey.ReplaceAllStringFunc(s, func(m string) string {
		sub := userKey.FindStringSubmatch(m)
		return strings.Replace(m, `"`+sub[2]+`"`, `"`+FixtureUser+`"`, 1)
	})
	for _, t := range r.Titles {
		to := r.retitle(titles, t)
		for _, esc := range escapes {
			s = strings.ReplaceAll(s, esc(t), esc(to))
		}
	}
	s = releaseName.ReplaceAllStringFunc(s, func(name string) string { return r.retitle(titles, name) })
	for _, from := range slices.Sorted(maps.Keys(r.stems)) {
		if !standalone(from) {
			continue
		}
		to := r.stems[from]
		for _, form := range stemForms {
			for _, esc := range escapes {
				old, nu := esc(form(from)), esc(form(to))
				s = wordBounded(old).ReplaceAllString(s, "${1}"+strings.ReplaceAll(nu, "$", "$$")+"${2}")
			}
		}
	}
	for _, h := range r.Hosts {
		s = strings.ReplaceAll(s, h, r.Host)
	}
	return []byte(s), nil
}

func (r *Rewriter) hash(h string) string {
	if Minted(h) {
		return h
	}
	lower := strings.ToLower(h)
	to, ok := r.hashes[lower]
	if !ok {
		to = Mint(len(r.hashes))
		r.hashes[lower] = to
	}
	if h != lower {
		return strings.ToUpper(to)
	}
	return to
}

// retitle is title with its stem swapped for the fictional title the
// stem maps to, in title's own separator.
func (r *Rewriter) retitle(fictional []string, title string) string {
	stem := Stem(title)
	if stem == "" || slices.Contains(fictional, stem) {
		return title
	}
	to, ok := r.stems[stem]
	if !ok {
		to = fictional[len(r.stems)%len(fictional)]
		r.stems[stem] = to
	}
	sep := " "
	if i := strings.IndexAny(title, ". _"); i >= 0 {
		sep = title[i : i+1]
	}
	from := strings.ReplaceAll(stem, " ", sep)
	if !strings.Contains(title, from) {
		return title
	}
	return strings.Replace(title, from, strings.ReplaceAll(to, " ", sep), 1)
}

// standalone reports whether stem is long enough to swap wherever it
// stands alone, as a group's name does beside its releases. A short one,
// such as "24", is swapped only inside a title.
func standalone(stem string) bool {
	return strings.Contains(stem, " ") || len(stem) >= 5
}

// stemForms are the ways a body may write a stem: its words joined by a
// space, a dot or an underscore, or as a lowercase-hyphen slug such as
// "the-quiet-orchard" in a link.
var stemForms = []func(string) string{
	func(s string) string { return s },
	func(s string) string { return strings.ReplaceAll(s, " ", ".") },
	func(s string) string { return strings.ReplaceAll(s, " ", "_") },
	func(s string) string { return strings.Trim(slugGaps.ReplaceAllString(strings.ToLower(s), "-"), "-") },
}

var slugGaps = regexp.MustCompile(`[^a-z0-9]+`)

// escapes are the ways a body may hold a title: plain, HTML-escaped,
// and with an apostrophe as a padded numeric reference.
var escapes = []func(string) string{
	func(s string) string { return s },
	html.EscapeString,
	func(s string) string { return strings.ReplaceAll(html.EscapeString(s), "&#39;", "&#039;") },
}

func wordBounded(s string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`(^|[^A-Za-z0-9])%s([^A-Za-z0-9]|$)`, regexp.QuoteMeta(s)))
}
