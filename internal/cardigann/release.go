package cardigann

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/newznab"
)

// Release is one result of a search.
type Release struct {
	Title, Details, Link, Magnet string
	// InfoHash is nil when the site gives none.
	InfoHash []byte
	// Size is 0 when the site gives none, and Published is zero.
	Size                     int64
	Published                time.Time
	Seeders, Leechers, Grabs int
	Categories               []newznab.ID
	// IDs are keyed imdbid (tt form), tmdbid, tvdbid, rid, tvmazeid,
	// traktid and doubanid.
	IDs                          map[string]string
	DownloadFactor, UploadFactor float64
	MinRatio                     float64
	MinSeed                      time.Duration
}

// parsed is a release as the parser fills it, with what Prowlarr's
// TorrentInfo holds besides the Release.
type parsed struct {
	Release
	guid, description, infoHash string
	peers                       int
	genres                      []string
}

// Result is the search's Releases once the exchange has ended. Releases
// without a title are dropped and the rest deduplicated by GUID, as
// CleanupReleases does, before andmatch.
func (x *Search) Result() ([]Release, error) {
	if err := x.ended(); err != nil {
		return nil, err
	}
	var kept []*parsed
	var guids []string
	for _, r := range x.releases {
		x.finish(r)
		if strings.TrimSpace(r.Title) == "" || slices.Contains(guids, r.guid) {
			continue
		}
		guids = append(guids, r.guid)
		kept = append(kept, r)
	}
	if slices.ContainsFunc(x.d.def.Search.Rows.Filters, func(f filterBlock) bool { return f.Name == "andmatch" }) {
		kept = andMatch(kept, x.query)
	}
	out := make([]Release, len(kept))
	for i, r := range kept {
		out[i] = r.Release
	}
	return out, nil
}

// finish is the parser's post-processing and CleanupReleases' GUID: a
// magnet from the info hash on a site that isn't private, trackerless
// (the design's D14), and the info hash from the magnet.
func (x *Search) finish(r *parsed) {
	r.Leechers = r.peers - r.Seeders
	if r.Magnet == "" && strings.TrimSpace(r.infoHash) != "" && x.d.def.Type != string(Private) {
		if l, err := magnet.Parse("magnet:?xt=urn:btih:" + r.infoHash); err == nil {
			l.Name = r.Title
			r.Magnet = l.String()
		}
	}
	if l, err := magnet.Parse("magnet:?xt=urn:btih:" + r.infoHash); err == nil {
		r.InfoHash = l.InfoHash[:]
	} else if l, err := magnet.Parse(r.Magnet); err == nil {
		r.InfoHash = l.InfoHash[:]
	}
	if r.guid == "" {
		switch {
		case r.Link != "":
			r.guid = r.Link
		case r.Magnet != "":
			r.guid = r.Magnet
		default:
			r.guid = r.Details
		}
	}
}

var (
	termSplit   = regexp.MustCompile(`[^\p{L}\p{Mn}\p{Nd}\p{Pc}]+`)
	commonWords = []string{"and", "the", "an", "of"}
)

// andMatch is FilterReleasesByQuery (IndexerBase.cs:165-186): with one
// term a release's title or description must hold it, and with more it
// must hold two of them. A search with no text or with ids is left alone.
func andMatch(rs []*parsed, q Query) []*parsed {
	if strings.TrimSpace(q.Text) == "" || isIDSearch(q) {
		return rs
	}
	var terms []string
	for _, t := range termSplit.Split(q.Text, -1) {
		if len(utf16.Encode([]rune(t))) > 1 && !slices.ContainsFunc(commonWords, func(w string) bool { return strings.EqualFold(w, t) }) {
			terms = append(terms, strings.ToLower(t))
		}
	}
	need := min(len(terms), 2)
	var out []*parsed
	for _, r := range rs {
		title, desc := strings.ToLower(r.Title), strings.ToLower(r.description)
		n := 0
		for _, t := range terms {
			if strings.Contains(title, t) || strings.Contains(desc, t) {
				n++
			}
		}
		if n >= need && n > 0 {
			out = append(out, r)
		}
	}
	return out
}

// isIDSearch is IsIdSearch for the modes SlipStream searches: an id, or
// for a movie a year.
func isIDSearch(q Query) bool {
	switch q.Mode {
	case "movie-search":
		return q.Year != 0 || slices.ContainsFunc([]string{"imdbid", "tmdbid", "traktid", "doubanid"}, func(k string) bool { return strings.TrimSpace(q.IDs[k]) != "" })
	case "tv-search":
		return slices.ContainsFunc([]string{"imdbid", "tvdbid", "rid", "traktid", "tvmazeid", "tmdbid", "doubanid"}, func(k string) bool { return strings.TrimSpace(q.IDs[k]) != "" })
	}
	return false
}

// fieldValue is a field's handling in ParseFields (CardigannParser.cs:
// 442-697): it sets the release and gives the value .Result holds.
type fieldValue func(x *Search, r *parsed, v value, f fieldUse) (value, error)

// fieldUse is what a field's handling reads besides its value.
type fieldUse struct {
	append bool
	reqURL *url.URL
}

// categoryListText is .Result.category after a category field: the
// C# List's ToString().
const categoryListText = "System.Collections.Generic.List`1[NzbDrone.Core.Indexers.IndexerCategory]"

var errNullValue = errors.New("the value is null")

var fieldValues = map[string]fieldValue{
	"download":             setDownload,
	"magnet":               setText(func(r *parsed, s string) { r.Magnet = s }),
	"infohash":             func(_ *Search, r *parsed, v value, _ fieldUse) (value, error) { r.infoHash = v.text; return v, nil },
	"details":              setURL(func(r *parsed, s string) { r.Details = s }),
	"comments":             setURL(func(*parsed, string) {}),
	"title":                setAppendable(func(r *parsed) *string { return &r.Title }),
	"description":          setAppendable(func(r *parsed) *string { return &r.description }),
	"category":             setCategories(func(c *newznab.Categories, s string) []newznab.Category { return c.FromTracker(s) }),
	"categorydesc":         setCategories(func(c *newznab.Categories, s string) []newznab.Category { return c.FromDesc(s) }),
	"size":                 setSize,
	"leechers":             setLeechers,
	"seeders":              setSeeders,
	"date":                 setDate,
	"files":                setInt(func(*parsed, int) {}),
	"grabs":                setInt(func(r *parsed, n int) { r.Grabs = n }),
	"downloadvolumefactor": setDouble(func(r *parsed, f float64) { r.DownloadFactor = f }),
	"uploadvolumefactor":   setDouble(func(r *parsed, f float64) { r.UploadFactor = f }),
	"minimumratio":         setDouble(func(r *parsed, f float64) { r.MinRatio = f }),
	"minimumseedtime":      setSeedTime,
	"imdb":                 setID("imdbid"),
	"imdbid":               setID("imdbid"),
	"tmdbid":               setID("tmdbid"),
	"rageid":               setID("rid"),
	"tvdbid":               setID("tvdbid"),
	"tvmazeid":             setID("tvmazeid"),
	"traktid":              setID("traktid"),
	"doubanid":             setID("doubanid"),
	"poster":               setPoster,
	"genre":                setGenre,
	"year":                 setInt(func(*parsed, int) {}),
}

// setField runs the field's handling. A name without one is a scratch
// value that only .Result reads.
func (x *Search) setField(r *parsed, name string, modifiers []string, v value, reqURL *url.URL) (value, error) {
	set, ok := fieldValues[name]
	if !ok {
		return v, nil
	}
	return set(x, r, v, fieldUse{append: slices.Contains(modifiers, "append"), reqURL: reqURL})
}

func setDownload(_ *Search, r *parsed, v value, f fieldUse) (value, error) {
	switch {
	case v.text == "":
		r.Link = ""
		return nullValue, nil
	case strings.HasPrefix(v.text, "magnet:"):
		r.Magnet = v.text
	default:
		u, err := resolveURL(f.reqURL, v.text)
		if err != nil {
			return nullValue, err
		}
		r.Link = u.String()
		v = textValue(r.Link)
	}
	r.guid = v.text
	return v, nil
}

func setText(set func(*parsed, string)) fieldValue {
	return func(_ *Search, r *parsed, v value, _ fieldUse) (value, error) {
		if v.IsNull() {
			return nullValue, errNullValue
		}
		set(r, v.text)
		return v, nil
	}
}

func setURL(set func(*parsed, string)) fieldValue {
	return func(_ *Search, r *parsed, v value, f fieldUse) (value, error) {
		if v.IsNull() {
			return nullValue, errNullValue
		}
		u, err := resolveURL(f.reqURL, v.text)
		if err != nil {
			return nullValue, err
		}
		set(r, u.String())
		return textValue(u.String()), nil
	}
}

func setAppendable(field func(*parsed) *string) fieldValue {
	return func(_ *Search, r *parsed, v value, f fieldUse) (value, error) {
		p := field(r)
		if !f.append {
			*p = v.text
			if v.IsNull() {
				return nullValue, nil
			}
			return v, nil
		}
		*p += v.text
		return textValue(*p), nil
	}
}

func setCategories(mapped func(*newznab.Categories, string) []newznab.Category) fieldValue {
	return func(x *Search, r *parsed, v value, _ fieldUse) (value, error) {
		for _, c := range mapped(&x.d.categories.mapping, v.text) {
			if !slices.Contains(r.Categories, c.ID) {
				r.Categories = append(r.Categories, c.ID)
			}
		}
		if !v.blank() && len(r.Categories) == 0 {
			return v, nil
		}
		return textValue(categoryListText), nil
	}
}

func setSize(_ *Search, r *parsed, v value, _ fieldUse) (value, error) {
	if v.IsNull() {
		return nullValue, errNullValue
	}
	n, err := parseSize(v.text)
	if err != nil {
		return nullValue, err
	}
	r.Size = n
	return textValue(strconv.FormatInt(n, 10)), nil
}

// peerLimit is the count at or above which Prowlarr reads seeders and
// leechers as 0.
const peerLimit = 5_000_000

func setLeechers(_ *Search, r *parsed, v value, _ fieldUse) (value, error) {
	if v.IsNull() {
		return nullValue, errNullValue
	}
	n, err := coerceLong(v.text)
	if err != nil {
		return nullValue, err
	}
	if n >= peerLimit {
		n = 0
	}
	r.peers += int(n)
	return textValue(strconv.FormatInt(n, 10)), nil
}

func setSeeders(_ *Search, r *parsed, v value, _ fieldUse) (value, error) {
	if v.IsNull() {
		return nullValue, errNullValue
	}
	n, err := coerceInt(v.text)
	if err != nil {
		return nullValue, err
	}
	if n >= peerLimit {
		n = 0
	}
	r.Seeders = n
	r.peers += n
	return textValue(strconv.Itoa(n)), nil
}

func setDate(x *Search, r *parsed, v value, _ fieldUse) (value, error) {
	if v.IsNull() {
		return nullValue, errNullValue
	}
	t, err := parseDate(v.text, x.env.Clock)
	if err != nil {
		return nullValue, err
	}
	r.Published = t
	return textValue(formatRFC1123Z(t)), nil
}

func setInt(set func(*parsed, int)) fieldValue {
	return func(_ *Search, r *parsed, v value, _ fieldUse) (value, error) {
		if v.IsNull() {
			return nullValue, errNullValue
		}
		n, err := coerceInt(v.text)
		if err != nil {
			return nullValue, err
		}
		set(r, n)
		return textValue(strconv.Itoa(n)), nil
	}
}

func setDouble(set func(*parsed, float64)) fieldValue {
	return func(_ *Search, r *parsed, v value, _ fieldUse) (value, error) {
		if v.IsNull() {
			return nullValue, errNullValue
		}
		f, err := coerceDouble(v.text)
		if err != nil {
			return nullValue, err
		}
		set(r, f)
		return textValue(formatDotNetDouble(f)), nil
	}
}

func setSeedTime(_ *Search, r *parsed, v value, _ fieldUse) (value, error) {
	if v.IsNull() {
		return nullValue, errNullValue
	}
	n, err := coerceLong(v.text)
	if err != nil {
		return nullValue, err
	}
	r.MinSeed = time.Duration(n) * time.Second
	return textValue(strconv.FormatInt(n, 10)), nil
}

// setID reads the first run of digits as the id, kept only when above 0.
// An IMDb id is written in tt form.
func setID(key string) fieldValue {
	return func(_ *Search, r *parsed, v value, _ fieldUse) (value, error) {
		n64, _, err := longFromString(v.text)
		if err != nil {
			return nullValue, err
		}
		n := int(int32(n64)) //nolint:gosec // Prowlarr casts the long to an int
		delete(r.IDs, key)
		if n > 0 {
			if r.IDs == nil {
				r.IDs = map[string]string{}
			}
			r.IDs[key] = strconv.Itoa(n)
			if key == "imdbid" {
				r.IDs[key] = fmt.Sprintf("tt%07d", n)
			}
		}
		return textValue(strconv.Itoa(n)), nil
	}
}

func setPoster(_ *Search, _ *parsed, v value, f fieldUse) (value, error) {
	if v.blank() {
		return nullValue, nil
	}
	u, err := resolveURL(f.reqURL, v.text)
	if err != nil {
		return nullValue, err
	}
	return textValue(u.String()), nil
}

const genreSeparators = ", /)(.;[]\"|:"

// setGenre adds the value's genres, split on Prowlarr's delimiters with
// underscores read as spaces, for .Result.genre. A Release carries no
// genres.
func setGenre(_ *Search, r *parsed, v value, _ fieldUse) (value, error) {
	if v.IsNull() {
		return nullValue, errNullValue
	}
	for g := range strings.FieldsFuncSeq(v.text, func(c rune) bool { return strings.ContainsRune(genreSeparators, c) }) {
		if g = strings.TrimSpace(g); g != "" && !slices.Contains(r.genres, g) {
			r.genres = append(r.genres, g)
		}
	}
	shown := make([]string, len(r.genres))
	for i, g := range r.genres {
		shown[i] = strings.ReplaceAll(g, "_", " ")
	}
	return textValue(strings.Join(shown, ", ")), nil
}
