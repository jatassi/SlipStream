package naming

import (
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/release"
)

// listForm is how a KindList entry reads modifier words and prints.
type listForm uint8

const (
	// nameList keeps or drops names, {Custom Formats:A,B}, and prints them
	// joined by spaces.
	nameList listForm = iota
	// pickList prints the one name its modifier gives when the list holds
	// it: {Custom Format:IMAX}.
	pickList
	// langList keeps or drops language codes and prints "[EN+DE]".
	langList
	// audioLangList is a langList that prints nothing when the Work's
	// original language is the only one left, as Sonarr does with English.
	audioLangList
)

// fieldScopes are the scopes each field's tokens may have.
var fieldScopes = map[field][]Scope{
	folderField: {WorkScope},
	levelField:  {WorkScope, LevelScope},
	fileField:   {WorkScope, LevelScope, ItemScope, FileScope},
}

var scopeGroups = map[Scope]Group{WorkScope: GroupWork, LevelScope: GroupLevel, ItemScope: GroupItem}

// newCatalog is d's tokens, the title variants each "<X> Title" token
// derives, and the core's tokens, by key. It fails on two names or aliases
// with one key, a Seq on a non-number, a declared token with the file
// scope, two year tokens in one scope, and a derived name the type also
// declares.
func newCatalog(d Declaration) (*catalog, error) {
	c := &catalog{byKey: map[Key]*entry{}}
	years, err := c.declare(d.Tokens)
	if err != nil {
		return nil, err
	}
	if err := c.derive(years); err != nil {
		return nil, err
	}
	for i := range core {
		if err := c.add(&core[i]); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// declare adds the type's tokens and answers each scope's year token.
func (c *catalog) declare(tokens []Token) (map[Scope]*entry, error) {
	years := map[Scope]*entry{}
	for _, t := range tokens {
		switch {
		case t.Seq != 0 && t.Kind != KindNumber:
			return nil, fmt.Errorf("token %q has a sequence role, and only a number token takes one", t.Name)
		case t.Scope == FileScope:
			return nil, fmt.Errorf("token %q has the file scope, which only the core's tokens have", t.Name)
		}
		t.Group = scopeGroups[t.Scope]
		e := &entry{Token: t, key: KeyOf(t.Name), read: lookup(t.Name, t.Scope)}
		if y := years[t.Scope]; t.Kind == KindYear && y != nil {
			return nil, fmt.Errorf("tokens %q and %q are both year tokens of one scope, so a title's year forms can't pick one", y.Name, t.Name)
		}
		if t.Kind == KindYear {
			years[t.Scope] = e
		}
		if err := c.add(e); err != nil {
			return nil, err
		}
	}
	return years, nil
}

// derive adds the variants of each declared "<X> Title" token, the Year
// forms only where its scope has a year token.
func (c *catalog) derive(years map[Scope]*entry) error {
	for _, e := range slices.Clone(c.ordered) {
		prefix, ok := strings.CutSuffix(e.Name, "Title")
		if e.Kind != KindTitle || !ok || prefix != "" && !strings.HasSuffix(prefix, " ") {
			continue
		}
		for i := range variants {
			v := &variants[i]
			if v.year && years[e.Scope] == nil {
				continue
			}
			d := derived(e, years[e.Scope], prefix, v)
			if other := c.byKey[d.key]; other != nil && other.base == nil {
				return fmt.Errorf("token %q is a title form %q derives, and the type also declares it", d.Name, e.Name)
			}
			if err := c.add(d); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *catalog) add(e *entry) error {
	for _, n := range append([]string{e.Name}, e.Aliases...) {
		k := KeyOf(n)
		if other := c.byKey[k]; other != nil {
			return fmt.Errorf("tokens %q and %q both match as %q", other.Name, n, k)
		}
		c.byKey[k] = e
	}
	c.ordered = append(c.ordered, e)
	return nil
}

// lookup reads a declared token's value from the tier its scope names.
func lookup(name string, scope Scope) func(Subject, int, FileFacts) Value {
	switch scope {
	case WorkScope:
		return func(s Subject, _ int, _ FileFacts) Value { return s.Work[name] }
	case LevelScope:
		return func(s Subject, unit int, _ FileFacts) Value { return s.Units[unit].Level[name] }
	case ItemScope:
		return func(s Subject, unit int, _ FileFacts) Value { return s.Units[unit].Item.Set[name] }
	case FileScope:
	}
	panic(fmt.Sprintf("naming: a declared token %q has scope %d", name, scope))
}

// derived is title's variant v, its year read from year, nil when the
// scope has none.
func derived(title, year *entry, prefix string, v *variant) *entry {
	t := title.Token
	t.Name, t.Aliases, t.Kind = prefix+v.suffix, nil, v.kind
	t.About = fmt.Sprintf(v.about, "{"+title.Name+"}")
	e := &entry{Token: t, key: KeyOf(t.Name), base: title, form: v, year: year}
	if t.Example != "" {
		exampleYear := int64(0)
		if year != nil {
			exampleYear, _ = strconv.ParseInt(year.Example, 10, 64)
		}
		e.Example = e.derive(t.Example, exampleYear)
	}
	e.read = func(s Subject, unit int, f FileFacts) Value {
		tv := title.read(s, unit, f)
		if tv.kind != textValue {
			return Value{}
		}
		return Text(e.derive(tv.text, e.yearAt(s, unit, f)))
	}
	return e
}

// derive is e's form of title. Its edge spaces go, as Sonarr trims every
// value.
func (e *entry) derive(title string, year int64) string {
	return strings.TrimSpace(e.form.derive(title, year))
}

func (e *entry) yearAt(s Subject, unit int, f FileFacts) int64 {
	if e.year == nil {
		return 0
	}
	return e.year.read(s, unit, f).n
}

// core is the core's tokens, rendered for every Content Type.
var core = []entry{
	fileToken("Target", KindID, GroupTarget, "4K HDR", `The label of the first Target in Profile order the file fills, with " (extra N)" for a Kept file.`,
		func(f FileFacts) Value { return Text(f.Target) }),
	fileToken("Edition", KindText, GroupTarget, "Director's Cut", "The name of the file's cut, and nothing for Standard.",
		func(f FileFacts) Value { return Text(f.Cut) }),
	tagged(fileToken("Edition Tag", KindText, GroupTarget, "{edition-Director's Cut}", "The cut as the edition tag Plex reads, and nothing for Standard.",
		func(f FileFacts) Value { return Text(f.Cut) }), editionTag),
	fileToken("Quality Title", KindText, GroupRelease, "Bluray-1080p", "The file's quality.",
		func(f FileFacts) Value { return Text(qualityTitle(f.Facts)) }),
	fileToken("Quality Full", KindText, GroupRelease, "Bluray-1080p Proper", `The file's quality, then "Proper" and "REAL" when they apply.`,
		func(f FileFacts) Value {
			return Text(joinSome(" ", qualityTitle(f.Facts), qualityProper(f.Facts.Revision), qualityReal(f.Facts.Revision)))
		}),
	fileToken("Quality Proper", KindText, GroupRelease, "Proper", `"Proper" when the release is a later version, such as a PROPER or REPACK.`,
		func(f FileFacts) Value { return Text(qualityProper(f.Facts.Revision)) }),
	fileToken("Quality Real", KindText, GroupRelease, "REAL", `"REAL" when the release name says REAL.`,
		func(f FileFacts) Value { return Text(qualityReal(f.Facts.Revision)) }),
	fileToken("Revision", KindText, GroupRelease, "v2 REPACK", "The release's version, and nothing for a first release.",
		func(f FileFacts) Value {
			if f.Facts.Revision.Original() {
				return Value{}
			}
			return Text(f.Facts.Revision.String())
		}),
	fileToken("Release Group", KindText, GroupRelease, "FLUX", "The release group, and nothing when it's unknown.",
		func(f FileFacts) Value { return Text(known(f.Facts.Group)) }),
	listToken(fileToken("Custom Formats", KindList, GroupRelease, "IMAX HDR10+", "Each matched Custom Format marked to include when renaming.",
		func(f FileFacts) Value { return List(f.Formats...) }), nameList),
	listToken(fileToken("Custom Format", KindList, GroupRelease, "IMAX", "The Custom Format named after the colon, as in :IMAX, when it matches and is marked to include when renaming.",
		func(f FileFacts) Value { return List(f.Formats...) }), pickList),
	fileToken("Original Title", KindText, GroupRelease, "The.Quiet.Orchard.2021.1080p.BluRay.x264-FLUX", "The name of the release the file came from.",
		func(f FileFacts) Value { return Text(f.Release) }),
	fileToken("Original Filename", KindText, GroupRelease, "the.quiet.orchard.2021.1080p.bluray.x264-flux", "The downloaded file's name without its extension, and nothing when it's unknown.",
		func(f FileFacts) Value { return Text(strings.TrimSuffix(f.Source, path.Ext(f.Source))) }),
	fileToken("MediaInfo Simple", KindText, GroupMedia, "x265 DDP", "The video codec and the audio codec.",
		func(f FileFacts) Value { return Text(joinSome(" ", known(f.Facts.Video), audioCodec(f.Facts))) }),
	fileToken("MediaInfo Full", KindText, GroupMedia, "x265 DDP[EN+DE]", "The video codec, the audio codec and the audio languages.",
		func(f FileFacts) Value {
			return Text(joinSome(" ", known(f.Facts.Video), audioCodec(f.Facts)) + audioLangList.print(audioLanguages(f.Facts), nil, f))
		}),
	fileToken("MediaInfo VideoCodec", KindText, GroupMedia, "x265", "The video codec.",
		func(f FileFacts) Value { return Text(known(f.Facts.Video)) }),
	fileToken("MediaInfo VideoBitDepth", KindText, GroupMedia, "10", "The video's bit depth.",
		func(f FileFacts) Value {
			if f.Media.BitDepth == 0 {
				return Value{}
			}
			return Text(strconv.Itoa(f.Media.BitDepth))
		}),
	fileToken("MediaInfo VideoDynamicRange", KindText, GroupMedia, "HDR", `"HDR" when the video has a high dynamic range.`,
		func(f FileFacts) Value {
			if len(hdrFormats(f.Facts)) == 0 {
				return Value{}
			}
			return Text("HDR")
		}),
	fileToken("MediaInfo VideoDynamicRangeType", KindText, GroupMedia, "DV HDR10", "Each high dynamic range format of the video.",
		func(f FileFacts) Value { return Text(strings.Join(hdrFormats(f.Facts), " ")) }),
	fileToken("MediaInfo 3D", KindText, GroupMedia, "3D", "Nothing yet, because SlipStream doesn't record whether a video is 3D.",
		func(FileFacts) Value { return Value{} }),
	fileToken("MediaInfo AudioCodec", KindText, GroupMedia, "DDP", "The first audio stream's codec.",
		func(f FileFacts) Value { return Text(audioCodec(f.Facts)) }),
	fileToken("MediaInfo AudioChannels", KindText, GroupMedia, "5.1", "The first audio stream's channel layout.",
		func(f FileFacts) Value {
			if c, ok := f.Facts.Channels.Get(); ok && len(c) > 0 {
				return Text(string(c[0]))
			}
			return Value{}
		}),
	listToken(fileToken("MediaInfo AudioLanguages", KindList, GroupMedia, "[EN+DE]", "Each audio language, and nothing when the Work's original language is the only one.",
		func(f FileFacts) Value { return List(audioLanguages(f.Facts)...) }), audioLangList),
	listToken(fileToken("MediaInfo AudioLanguagesAll", KindList, GroupMedia, "[EN]", "Each audio language.",
		func(f FileFacts) Value { return List(audioLanguages(f.Facts)...) }), langList),
	listToken(fileToken("MediaInfo SubtitleLanguages", KindList, GroupMedia, "[EN+DE]", "Each subtitle language.",
		func(f FileFacts) Value { return List(subtitleLanguages(f)...) }), langList),
	listToken(fileToken("MediaInfo SubtitleLanguagesAll", KindList, GroupMedia, "[EN+DE]", "Each subtitle language.",
		func(f FileFacts) Value { return List(subtitleLanguages(f)...) }), langList),
}

func fileToken(name string, kind Kind, group Group, example, about string, read func(FileFacts) Value) entry {
	return entry{
		Token: Token{Name: name, Kind: kind, Scope: FileScope, Group: group, Example: example, About: about},
		key:   KeyOf(name),
		read:  func(_ Subject, _ int, f FileFacts) Value { return read(f) },
	}
}

func tagged(e entry, t tag) entry {
	e.tag = t
	return e
}

func listToken(e entry, l listForm) entry {
	e.list = l
	return e
}

func known[T ~string](f release.Fact[T]) string {
	v, _ := f.Get()
	return string(v)
}

func joinSome(sep string, parts ...string) string {
	return strings.Join(slices.DeleteFunc(parts, func(s string) bool { return s == "" }), sep)
}

func qualityTitle(f release.Facts) string {
	if q, ok := f.Quality.Get(); ok {
		return q.Info().Name
	}
	return ""
}

func qualityProper(r release.Revision) string {
	if r.Version > 1 {
		return "Proper"
	}
	return ""
}

func qualityReal(r release.Revision) string {
	if r.Real > 0 {
		return "REAL"
	}
	return ""
}

// audioWords are the audio codecs whose vocabulary word reads differently
// in a scene name.
var audioWords = map[release.AudioCodec]string{"DD+": "DDP"}

func audioCodec(f release.Facts) string {
	a, ok := f.Audio.Get()
	if !ok || len(a) == 0 {
		return ""
	}
	if w, ok := audioWords[a[0]]; ok {
		return w
	}
	return string(a[0])
}

func hdrFormats(f release.Facts) []string {
	hdr, _ := f.HDR.Get()
	var out []string
	for _, h := range hdr {
		if h != "SDR" {
			out = append(out, string(h))
		}
	}
	return out
}

func audioLanguages(f release.Facts) []string {
	ls, _ := f.Languages.Get()
	original, _ := f.Coverage.Original.Get()
	codes := make([]string, 0, len(ls))
	for _, l := range ls {
		if l == release.Original {
			l = original
		}
		codes = append(codes, string(l))
	}
	return languageCodes(codes)
}

func subtitleLanguages(f FileFacts) []string {
	codes := make([]string, 0, len(f.Media.Subtitles))
	for _, s := range f.Media.Subtitles {
		codes = append(codes, string(s.Language))
	}
	return languageCodes(codes)
}

// languageCodes are codes in capitals, each once, with the empty ones
// left out.
func languageCodes(codes []string) []string {
	var out []string
	for _, c := range codes {
		if c = strings.ToUpper(c); c != "" && !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	return out
}
