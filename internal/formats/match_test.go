package formats_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
)

// limit is long enough that no test but the timeout tests sees one on a
// busy machine.
var limit = formats.Timings{Match: time.Minute}

func title(pattern string) formats.Condition {
	return cond(formats.ReleaseTitle, map[string]any{"value": pattern})
}

func cond(k formats.Kind, fields map[string]any) formats.Condition {
	b, err := json.Marshal(fields)
	if err != nil {
		panic(err)
	}
	return formats.Condition{Name: string(k), Kind: k, Fields: b}
}

func negated(c formats.Condition) formats.Condition {
	c.Negate = true
	return c
}

func required(c formats.Condition) formats.Condition {
	c.Required = true
	return c
}

func format(id byte, d formats.Dialect, cs ...formats.Condition) formats.Format {
	return formats.Format{ID: formats.ID{id}, Name: "format " + string('A'+rune(id)), Dialect: d, Conditions: cs}
}

func matches(t *testing.T, f formats.Format, r release.Read) bool {
	t.Helper()
	if err := formats.Check(f, limit); err != nil {
		t.Fatalf("check: %v", err)
	}
	res := formats.Compile([]formats.Format{f}, limit).Match(r)
	return slices.Contains(res.Matched, f.ID)
}

func named(name string) release.Read { return release.Read{Given: name} }

func TestCombining(t *testing.T) {
	r := named("Film.2020.REPACK.1080p.BluRay.x264-GRP")
	cases := []struct {
		name  string
		conds []formats.Condition
		want  bool
	}{
		{"one matching condition", []formats.Condition{title(`\bREPACK\b`)}, true},
		{"one condition of a kind is enough", []formats.Condition{title(`\bPROPER\b`), title(`\bREPACK\b`)}, true},
		{"none of a kind matching fails", []formats.Condition{title(`\bPROPER\b`), title(`\bRERIP\b`)}, false},
		{"a required condition that fails fails its group", []formats.Condition{title(`\bREPACK\b`), required(title(`\bPROPER\b`))}, false},
		{"every required condition matching", []formats.Condition{required(title(`\bREPACK\b`)), required(title(`1080p`))}, true},
		{"negate flips a match", []formats.Condition{negated(title(`\bREPACK\b`))}, false},
		{"negated and required means must not match", []formats.Condition{title(`\bREPACK\b`), required(negated(title(`\bREPACK2\b`)))}, true},
		{"negated and required failing", []formats.Condition{title(`\bREPACK\b`), required(negated(title(`1080p`)))}, false},
		{"a group of only failed required conditions fails", []formats.Condition{required(negated(title(`1080p`)))}, false},
		{"kinds combine with and", []formats.Condition{title(`\bREPACK\b`), cond(formats.ReleaseGroup, map[string]any{"value": "^OTHER$"})}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matches(t, format(1, formats.Radarr, tc.conds...), r); got != tc.want {
				t.Errorf("matched = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCheckRefusesAFormatWithNoConditions(t *testing.T) {
	var empty *formats.EmptyError
	if err := formats.Check(format(1, formats.Radarr), limit); !errors.As(err, &empty) || empty.Name != "format B" {
		t.Errorf("Check = %v, want an EmptyError naming format B", err)
	}
}

type kindCase struct {
	name    string
	dialect formats.Dialect
	cond    formats.Condition
	read    release.Read
	want    bool
}

func runKindCases(t *testing.T, cases []kindCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matches(t, format(1, tc.dialect, tc.cond), tc.read); got != tc.want {
				t.Errorf("matched = %v, want %v", got, tc.want)
			}
		})
	}
}

func withFacts(f release.Facts) release.Read { return release.Read{Given: "Name", Facts: f} }

func TestPatternKinds(t *testing.T) {
	group := withFacts(release.Facts{Group: release.Known("FraMeSToR")})
	edition := withFacts(release.Facts{Edition: release.Known("Director's Cut")})
	runKindCases(t, []kindCase{
		{"a title is matched case-insensitively", formats.Radarr, title(`bluray`), named("Film.1080p.BluRay"), true},
		{"a title pattern is .NET syntax", formats.Sonarr, title(`(?<=\.)DV(?=\.)`), named("Show.S01E01.DV.HDR"), true},
		{"a title pattern reads .NET's Unicode blocks", formats.Sonarr, title(`\p{IsGreek}`), named("Ταινία.2020"), true},
		{"a group is matched", formats.Radarr, cond(formats.ReleaseGroup, map[string]any{"value": `^(framestor)$`}), group, true},
		{"an unknown group never matches", formats.Radarr, cond(formats.ReleaseGroup, map[string]any{"value": `.*`}), named("Name"), false},
		{"so a negated group condition matches an unknown group", formats.Radarr, negated(cond(formats.ReleaseGroup, map[string]any{"value": `.*`})), named("Name"), true},
		{"an edition is matched", formats.Radarr, cond(formats.Edition, map[string]any{"value": `director'?s`}), edition, true},
		{"an unknown edition never matches", formats.Radarr, cond(formats.Edition, map[string]any{"value": `.*`}), named("Name"), false},
		{"a slipstream title pattern", formats.SlipStream, title(`x264`), named("Film.x264"), true},
	})
}

func languages(original release.Fact[release.Language], tagged ...release.Language) release.Read {
	f := release.Facts{Coverage: release.Coverage{Original: original}}
	if tagged != nil {
		f.Languages = release.Known(tagged)
	}
	return withFacts(f)
}

func lang(value any, except bool) formats.Condition {
	return cond(formats.Language, map[string]any{"value": value, "exceptLanguage": except})
}

func TestLanguage(t *testing.T) {
	const english, french, japanese = 1, 2, 8
	en, ja := release.Known[release.Language]("en"), release.Known[release.Language]("ja")
	unknown := release.Fact[release.Language]{}
	runKindCases(t, []kindCase{
		{"a tagged language", formats.Radarr, lang(french, false), languages(en, "fr"), true},
		{"a language the name isn't tagged with", formats.Radarr, lang(english, false), languages(en, "fr"), false},
		{"an untagged name reads as the original language", formats.Sonarr, lang(english, false), languages(en), true},
		{"so a negated condition for it doesn't match", formats.Sonarr, negated(lang(english, false)), languages(en), false},
		{"an untagged name with no known original reads as Unknown", formats.Radarr, lang(0, false), languages(unknown), true},
		{"and as no language", formats.Radarr, lang(english, false), languages(unknown), false},
		{"Original reads as the Work's original language", formats.Radarr, lang(-2, false), languages(ja, "ja"), true},
		{"Original with a tag in another language", formats.Sonarr, lang(-2, false), languages(ja, "en"), false},
		{"TRaSH's Not Original on an untagged name", formats.Sonarr, negated(lang(-2, false)), languages(ja), false},
		{"Original with no known original language never matches", formats.Radarr, lang(-2, false), languages(unknown, "en"), false},
		{"a name tagged as the original reads as the original language", formats.Radarr, lang(japanese, false), languages(ja, release.Original), true},
		{"tagged as the original and in it adds Unknown", formats.Radarr, lang(0, false), languages(ja, "ja", release.Original), true},
		{"except matches a name with another language", formats.Radarr, lang(english, true), languages(en, "en", "fr"), true},
		{"except doesn't match a name in that language alone", formats.Radarr, lang(english, true), languages(en, "en"), false},
		{"Radarr's Any is never tagged", formats.Radarr, lang(-1, false), languages(en, "en"), false},
		{"slipstream names a language by its code", formats.SlipStream, lang("fr", false), languages(en, "fr"), true},
		{"slipstream's original", formats.SlipStream, lang("original", false), languages(ja), true},
	})
}

func TestIndexerFlag(t *testing.T) {
	flagged := func(fs ...release.Flag) release.Read { return withFacts(release.Facts{Flags: fs}) }
	flag := func(v any) formats.Condition { return cond(formats.IndexerFlag, map[string]any{"value": v}) }
	runKindCases(t, []kindCase{
		{"radarr G_Freeleech", formats.Radarr, flag(1), flagged(release.Freeleech), true},
		{"radarr G_Internal", formats.Radarr, flag(32), flagged(release.Scene, release.Internal), true},
		{"radarr PTP_Golden", formats.Radarr, flag(8), flagged(release.Golden), true},
		{"radarr AHD_Internal is no flag a Release carries", formats.Radarr, flag(64), flagged(release.Internal), false},
		{"sonarr Internal", formats.Sonarr, flag(8), flagged(release.Internal), true},
		{"sonarr Freeleech without it", formats.Sonarr, flag(1), flagged(release.Halfleech), false},
		{"slipstream by name", formats.SlipStream, flag("nuked"), flagged(release.Nuked), true},
	})
}

func TestSource(t *testing.T) {
	fine := func(o release.Origin, m ...release.Modifier) release.Read {
		f := release.Facts{Source: release.Known(o)}
		if len(m) > 0 {
			f.Modifier = release.Known(m[0])
		}
		return withFacts(f)
	}
	source := func(v any) formats.Condition { return cond(formats.Source, map[string]any{"value": v}) }
	runKindCases(t, []kindCase{
		{"radarr BLURAY", formats.Radarr, source(9), fine(release.OriginBluray), true},
		{"radarr BLURAY matches a Remux", formats.Radarr, source(9), fine(release.OriginBluray, release.ModRemux), true},
		{"radarr WEBDL", formats.Radarr, source(7), fine(release.OriginWEBDL), true},
		{"radarr WEBRIP isn't WEBDL", formats.Radarr, source(8), fine(release.OriginWEBDL), false},
		{"radarr TELESYNC", formats.Radarr, source(2), fine(release.OriginTelesync), true},
		{"radarr UNKNOWN matches an unknown source", formats.Radarr, source(0), named("Name"), true},
		{"sonarr Bluray", formats.Sonarr, source(6), fine(release.OriginBluray), true},
		{"sonarr Bluray matches a BR-DISK", formats.Sonarr, source(6), fine(release.OriginBluray, release.ModBRDisk), true},
		{"sonarr Bluray doesn't match a Remux", formats.Sonarr, source(6), fine(release.OriginBluray, release.ModRemux), false},
		{"sonarr BlurayRaw is a Remux", formats.Sonarr, source(7), fine(release.OriginBluray, release.ModRemux), true},
		{"sonarr BlurayRaw isn't a plain Bluray", formats.Sonarr, source(7), fine(release.OriginBluray), false},
		{"sonarr Television", formats.Sonarr, source(1), fine(release.OriginTV), true},
		{"sonarr Television isn't Raw-HD", formats.Sonarr, source(1), fine(release.OriginTV, release.ModRawHD), false},
		{"sonarr TelevisionRaw is Raw-HD", formats.Sonarr, source(2), fine(release.OriginTV, release.ModRawHD), true},
		{"sonarr Web is WEB-DL", formats.Sonarr, source(3), fine(release.OriginWEBDL), true},
		{"sonarr WebRip", formats.Sonarr, source(4), fine(release.OriginWEBRip), true},
		{"sonarr Unknown", formats.Sonarr, source(0), fine(release.OriginDVD), false},
		{"slipstream webrip", formats.SlipStream, source("webrip"), fine(release.OriginWEBRip), true},
		{"slipstream bluray matches a Remux", formats.SlipStream, source("bluray"), fine(release.OriginBluray, release.ModRemux), true},
	})
}

func TestResolutionModifierAndReleaseType(t *testing.T) {
	at := func(r release.Resolution) release.Read { return withFacts(release.Facts{Resolution: release.Known(r)}) }
	mod := func(m release.Modifier) release.Read { return withFacts(release.Facts{Modifier: release.Known(m)}) }
	span := func(s release.Span) release.Read { return release.Read{Given: "Name", Name: release.Name{Span: s}} }
	value := func(k formats.Kind, v any) formats.Condition { return cond(k, map[string]any{"value": v}) }
	runKindCases(t, []kindCase{
		{"radarr 2160p", formats.Radarr, value(formats.Resolution, 2160), at(release.R2160p), true},
		{"sonarr 1080p isn't 720p", formats.Sonarr, value(formats.Resolution, 1080), at(release.R720p), false},
		{"Unknown matches an unknown resolution", formats.Sonarr, value(formats.Resolution, 0), named("Name"), true},
		{"slipstream 540p", formats.SlipStream, value(formats.Resolution, "540p"), at(release.R540p), true},
		{"radarr REMUX", formats.Radarr, value(formats.QualityModifier, 5), mod(release.ModRemux), true},
		{"radarr BRDISK isn't REMUX", formats.Radarr, value(formats.QualityModifier, 4), mod(release.ModRemux), false},
		{"radarr NONE matches no modifier", formats.Radarr, value(formats.QualityModifier, 0), named("Name"), true},
		{"slipstream screener", formats.SlipStream, value(formats.QualityModifier, "screener"), mod(release.ModScreener), true},
		{"sonarr SeasonPack is a whole level", formats.Sonarr, value(formats.ReleaseType, 3), span(release.Level), true},
		{"sonarr SingleEpisode", formats.Sonarr, value(formats.ReleaseType, 1), span(release.Several), false},
		{"sonarr MultiSeasonPack", formats.Sonarr, value(formats.ReleaseType, 4), span(release.Levels), true},
		{"sonarr Unknown matches an unknown span", formats.Sonarr, value(formats.ReleaseType, 0), named("Name"), true},
		{"slipstream several", formats.SlipStream, value(formats.ReleaseType, "several"), span(release.Several), true},
	})
}

func TestSizeAndYear(t *testing.T) {
	const gib = 1 << 30
	sized := func(b int64) release.Read { return withFacts(release.Facts{Size: release.Known(b)}) }
	dated := func(y int) release.Read {
		return release.Read{Given: "Name", Name: release.Name{Year: release.Known(y)}}
	}
	size := cond(formats.Size, map[string]any{"min": 1, "max": 2.5})
	year := cond(formats.Year, map[string]any{"min": 2000, "max": 2010})
	runKindCases(t, []kindCase{
		{"inside the size range", formats.Radarr, size, sized(2 * gib), true},
		{"at the minimum is outside", formats.Radarr, size, sized(gib), false},
		{"at the maximum is inside", formats.Sonarr, size, sized(5 * gib / 2), true},
		{"above the maximum", formats.Sonarr, size, sized(5*gib/2 + 1), false},
		{"an unknown size reads as 0", formats.Radarr, cond(formats.Size, map[string]any{"min": 0, "max": 1}), named("Name"), false},
		{"inside the years", formats.Radarr, year, dated(2005), true},
		{"the years are inclusive", formats.Radarr, year, dated(2010), true},
		{"before the years", formats.Radarr, year, dated(1999), false},
		{"an unknown year never matches", formats.Radarr, year, named("Name"), false},
	})
}

func TestCheck(t *testing.T) {
	cases := []struct {
		name string
		f    formats.Format
		want any
	}{
		{
			"a group left open", format(1, formats.Radarr, title(`ok`), title(`(open`)),
			formats.PatternError{Condition: 1, Name: "ReleaseTitleSpecification", Offset: -1, Reason: "unterminated group"},
		},
		{
			"a stray parenthesis at its position", format(1, formats.Sonarr, title(`ab)c`)),
			formats.PatternError{Name: "ReleaseTitleSpecification", Offset: 3, Reason: "a ) has no matching ("},
		},
		{
			"an empty pattern", format(1, formats.Radarr, cond(formats.ReleaseGroup, map[string]any{"value": ""})),
			formats.PatternError{Name: "ReleaseGroupSpecification", Offset: -1, Reason: "it's empty"},
		},
		{
			"a kind no format has", format(1, formats.Radarr, cond("CodecSpecification", map[string]any{"value": 1})),
			formats.KindError{Name: "CodecSpecification", Kind: "CodecSpecification"},
		},
		{
			"a field the kind lacks", format(1, formats.Radarr, cond(formats.Source, map[string]any{"value": 1, "min": 2})),
			formats.FieldError{Name: "SourceSpecification", Kind: formats.Source, Field: "min"},
		},
		{
			"a value outside the dialect", format(1, formats.Radarr, cond(formats.Source, map[string]any{"value": 42})),
			formats.ValueError{Name: "SourceSpecification", Kind: formats.Source, Field: "value", Value: "42", Want: "a radarr SourceSpecification value"},
		},
		{
			"a kind the dialect has no values for", format(1, formats.Sonarr, cond(formats.QualityModifier, map[string]any{"value": 5})),
			formats.ValueError{Name: "QualityModifierSpecification", Kind: formats.QualityModifier, Field: "value", Value: "5", Want: "a sonarr QualityModifierSpecification value"},
		},
		{
			"a number where a name goes", format(1, formats.SlipStream, cond(formats.Source, map[string]any{"value": 9})),
			formats.ValueError{Name: "SourceSpecification", Kind: formats.Source, Field: "value", Value: "9", Want: "text"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := refusal(formats.Check(tc.f, limit)); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Check = %#v, want %#v", got, tc.want)
			}
			if c := formats.Compile([]formats.Format{tc.f}, limit); !slices.Equal(c.Broken(), []formats.ID{tc.f.ID}) {
				t.Errorf("Broken = %v", c.Broken())
			}
		})
	}
}

func refusal(err error) any {
	var pe *formats.PatternError
	var ke *formats.KindError
	var fe *formats.FieldError
	var ve *formats.ValueError
	switch {
	case errors.As(err, &pe):
		return *pe
	case errors.As(err, &ke):
		return *ke
	case errors.As(err, &fe):
		return *fe
	case errors.As(err, &ve):
		return *ve
	}
	return err
}

// slow backtracks for far longer than the limit on a run of a's.
const slow = `^(a+)+$`

var aaa = named(strings.Repeat("a", 40) + "!")

func TestTimeout(t *testing.T) {
	short := formats.Timings{Match: 10 * time.Millisecond}
	f := format(1, formats.Radarr, title(`x`), negated(title(slow)))
	c := formats.Compile([]formats.Format{f}, short)

	res := c.Match(aaa)
	if len(res.Matched) != 0 {
		t.Errorf("a timed-out negated condition matched")
	}
	if !slices.Equal(res.Timeouts, []formats.Timeout{{Format: f.ID, Condition: 1}}) {
		t.Errorf("Timeouts = %v", res.Timeouts)
	}

	e := c.Explain(aaa)[0]
	if e.Matched || !e.TimedOut || !e.Conditions[1].TimedOut || e.Conditions[1].Matched {
		t.Errorf("Explain = %+v", e)
	}
}

func TestMatchStopsAtTheFirstFailedGroup(t *testing.T) {
	short := formats.Timings{Match: 10 * time.Millisecond}
	f := format(1, formats.Radarr, title(slow), cond(formats.Source, map[string]any{"value": 9}))
	c := formats.Compile([]formats.Format{f}, short)
	if res := c.Match(aaa); len(res.Timeouts) != 0 {
		t.Errorf("the title ran after the cheaper Source group failed: %v", res.Timeouts)
	}
	if e := c.Explain(aaa)[0]; !e.Conditions[0].TimedOut {
		t.Errorf("Explain skipped the title: %+v", e)
	}
}

func TestExplain(t *testing.T) {
	f := format(1, formats.Radarr, title(`REPACK`), required(negated(title(`REPACK2`))), cond(formats.Resolution, map[string]any{"value": 1080}))
	r := named("Film.REPACK.1080p")
	got := formats.Compile([]formats.Format{f}, limit).Explain(r)
	want := []formats.Explained{{
		Format: f.ID, Name: f.Name, Matched: false,
		Conditions: []formats.Evaluated{
			{Name: string(formats.ReleaseTitle), Kind: formats.ReleaseTitle, Matched: true},
			{Name: string(formats.ReleaseTitle), Kind: formats.ReleaseTitle, Negate: true, Required: true, Matched: true},
			{Name: string(formats.Resolution), Kind: formats.Resolution, Matched: false},
		},
	}}
	if !equalExplained(got, want) {
		t.Errorf("Explain = %+v\nwant %+v", got, want)
	}
}

func equalExplained(a, b []formats.Explained) bool {
	return slices.EqualFunc(a, b, func(x, y formats.Explained) bool {
		return x.Format == y.Format && x.Name == y.Name && x.Matched == y.Matched && x.TimedOut == y.TimedOut && slices.Equal(x.Conditions, y.Conditions)
	})
}

func TestCatalog(t *testing.T) {
	b := format(2, formats.Radarr, title(`1080p`))
	b.Name = "Alpha"
	a := format(1, formats.Radarr, title(`x264`))
	a.Name = "Zulu"
	broken := format(3, formats.Radarr, title(`(`))
	c := formats.Compile([]formats.Format{b, a, broken}, limit)
	r := named("Film.1080p.x264")

	if c.Len() != 3 || !slices.Equal(c.Broken(), []formats.ID{broken.ID}) {
		t.Fatalf("Len = %d, Broken = %v", c.Len(), c.Broken())
	}
	if got := c.Match(r).Matched; !slices.Equal(got, formats.Set{a.ID, b.ID}) {
		t.Errorf("Match = %v, want both, sorted by ID", got)
	}
	if names := explainedNames(c.Explain(r)); !slices.Equal(names, []string{"Alpha", "Zulu", broken.Name}) {
		t.Errorf("Explain order = %v, want name order", names)
	}

	only := c.Only(formats.Set{b.ID})
	if only.Len() != 1 || !slices.Equal(only.Match(r).Matched, formats.Set{b.ID}) {
		t.Errorf("Only = %v", only.Match(r).Matched)
	}

	draft := a
	draft.Conditions = []formats.Condition{title(`x265`)}
	with := c.With(draft)
	if with.Len() != 3 || !slices.Equal(with.Match(r).Matched, formats.Set{b.ID}) {
		t.Errorf("With a draft = %v", with.Match(r).Matched)
	}
	if !slices.Equal(c.Match(r).Matched, formats.Set{a.ID, b.ID}) {
		t.Errorf("With changed the catalog it came from")
	}

	fixed := broken
	fixed.Conditions = []formats.Condition{title(`Film`)}
	if w := c.With(fixed); len(w.Broken()) != 0 || len(w.Match(r).Matched) != 3 {
		t.Errorf("With a fixed format: Broken = %v, Matched = %v", w.Broken(), w.Match(r).Matched)
	}
	if w := c.With(format(4, formats.Radarr, title(`Film`))); w.Len() != 4 {
		t.Errorf("With a new format: Len = %d", w.Len())
	}
}

func explainedNames(es []formats.Explained) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}

func TestKinds(t *testing.T) {
	var regex []formats.Kind
	for _, k := range formats.Kinds() {
		if k.Regex {
			regex = append(regex, k.Kind)
		}
	}
	if !slices.Equal(regex, []formats.Kind{formats.ReleaseTitle, formats.Edition, formats.ReleaseGroup}) {
		t.Errorf("regex kinds = %v", regex)
	}
	if n := len(formats.Kinds()); n != 11 {
		t.Errorf("%d kinds, want 11", n)
	}
}
