package naming

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

var stubEntries = map[Key]*entry{}

func init() {
	for _, tok := range []Token{
		{Name: "Series Title", Kind: KindTitle, Scope: WorkScope},
		{Name: "Episode Title", Kind: KindTitle, Scope: ItemScope},
		{Name: "season", Kind: KindNumber, Scope: LevelScope, Seq: SeqOuter},
		{Name: "episode", Kind: KindNumber, Scope: ItemScope, Seq: SeqInner},
		{Name: "absolute", Kind: KindNumber, Scope: ItemScope, Seq: SeqAlone},
	} {
		e := &entry{Token: tok, key: KeyOf(tok.Name)}
		e.read = func(s Subject, unit int, _ FileFacts) Value {
			u := s.Units[unit]
			if v, ok := s.Work[tok.Name]; ok {
				return v
			}
			if v, ok := u.Level[tok.Name]; ok {
				return v
			}
			return u.Item.Set[tok.Name]
		}
		stubEntries[e.key] = e
	}
}

// stubValue stands in for the render pipeline: a number padded to its
// modifier, text as it is.
func stubValue(r *boundRef, u Unit) piece {
	v := r.entry.read(Subject{Work: Set{"Series Title": Text("Show")}, Units: []Unit{u}}, 0, FileFacts{})
	if v.kind == numberValue {
		width, _ := r.mod.(padding)
		return piece{text: fmt.Sprintf("%0*d", int(width), v.n)}
	}
	return piece{text: v.text, empty: v.kind == 0}
}

// renderStub renders text for units the way a file template's render
// does around the block: each block span through multi, every other ref
// for the first unit, Item-scope titles joined.
func renderStub(t *testing.T, text string, style MultiEpisode, units []Unit) string {
	t.Helper()
	tmpl, err := Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	b := bound{parts: make([]part, len(tmpl.parts))}
	for i, p := range tmpl.parts {
		if r, ok := p.(*ref); ok {
			p = &boundRef{ref: *r, entry: stubEntries[r.key]}
		}
		b.parts[i] = p
	}
	b.block, b.alone = blockSpan(b.parts)
	var out strings.Builder
	at := 0
	for _, s := range b.blockSpans() {
		out.WriteString(renderPlain(b.parts[at:s.from], units))
		for _, p := range multi(blockOf(b.parts, s), style, units, stubValue) {
			out.WriteString(p.text)
		}
		at = s.to
	}
	out.WriteString(renderPlain(b.parts[at:], units))
	return out.String()
}

func renderPlain(parts []part, units []Unit) string {
	var out strings.Builder
	for _, p := range parts {
		switch p := p.(type) {
		case literal:
			out.WriteString(string(p))
		case *boundRef:
			if p.entry.Kind == KindTitle && p.entry.Scope == ItemScope {
				titles := make([]Value, len(units))
				for i, u := range units {
					titles[i] = u.Item.Set[p.entry.Name]
				}
				out.WriteString(joinTitles(titles, nil).text)
				continue
			}
			out.WriteString(stubValue(p, units[0]).text)
		}
	}
	return out.String()
}

func ep(season, episode, absolute int64) Unit {
	return Unit{
		Level: Set{"season": Number(season)},
		Item: ItemValues{Set: Set{
			"episode":       Number(episode),
			"absolute":      Number(absolute),
			"Episode Title": Text("City Sushi"),
		}},
	}
}

func withoutAbsolute(u Unit) Unit {
	delete(u.Item.Set, "absolute")
	return u
}

// eps are episodes of season 1, each numbered the same in absolute order.
func eps(episodes ...int64) []Unit {
	units := make([]Unit, len(episodes))
	for i, e := range episodes {
		units[i] = ep(1, e, e)
	}
	return units
}

var styles = []MultiEpisode{Extend, Duplicate, Repeat, Scene, Range, PrefixedRange}

func TestMultiEpisodeStyles(t *testing.T) {
	crossing := []Unit{ep(1, 12, 12), ep(2, 1, 13)}
	cases := []struct {
		template string
		units    []Unit
		// want is by style, in the order of styles.
		want [6]string
	}{
		{"S{season:00}E{episode:00}", eps(1, 2, 3), [6]string{
			"S01E01-02-03", "S01E01-S01E02-S01E03", "S01E01E02E03", "S01E01-E02-E03", "S01E01-03", "S01E01-E03",
		}},
		{"{Series Title} - S{season:00}E{episode:00} - {Episode Title}", eps(1, 2, 3), [6]string{
			"Show - S01E01-02-03 - City Sushi",
			"Show - S01E01 - S01E02 - S01E03 - City Sushi",
			"Show - S01E01E02E03 - City Sushi",
			"Show - S01E01-E02-E03 - City Sushi",
			"Show - S01E01-03 - City Sushi",
			"Show - S01E01-E03 - City Sushi",
		}},
		{"{Series Title} - S{season:00}E{episode:00} - {Episode Title}", eps(3, 1, 2), [6]string{
			"Show - S01E01-02-03 - City Sushi",
			"Show - S01E01 - S01E02 - S01E03 - City Sushi",
			"Show - S01E01E02E03 - City Sushi",
			"Show - S01E01-E02-E03 - City Sushi",
			"Show - S01E01-03 - City Sushi",
			"Show - S01E01-E03 - City Sushi",
		}},
		{"{Series Title} - S{season:00}E{episode:00}", eps(1, 3), [6]string{
			"Show - S01E01E03", "Show - S01E01E03", "Show - S01E01E03", "Show - S01E01E03", "Show - S01E01E03", "Show - S01E01E03",
		}},
		{"{Series Title} - S{season:00}E{episode:00}", crossing, [6]string{
			"Show - S01E12 - S02E01", "Show - S01E12 - S02E01", "Show - S01E12 - S02E01",
			"Show - S01E12 - S02E01", "Show - S01E12 - S02E01", "Show - S01E12 - S02E01",
		}},
		{"{Series Title} - S{season:00}E{episode:00}", eps(5), [6]string{
			"Show - S01E05", "Show - S01E05", "Show - S01E05", "Show - S01E05", "Show - S01E05", "Show - S01E05",
		}},
		{"{Series Title} - s{season}e{episode}", eps(1, 2, 3), [6]string{
			"Show - s1e1-2-3", "Show - s1e1 - s1e2 - s1e3", "Show - s1e1e2e3", "Show - s1e1-e2-e3", "Show - s1e1-3", "Show - s1e1-e3",
		}},
		{"{Series Title} - s{season}e{episode}", eps(1, 3), [6]string{
			"Show - s1e1e3", "Show - s1e1e3", "Show - s1e1e3", "Show - s1e1e3", "Show - s1e1e3", "Show - s1e1e3",
		}},
		{"{Series Title} - s{season}e{episode}", crossing, [6]string{
			"Show - s1e12 - s2e1", "Show - s1e12 - s2e1", "Show - s1e12 - s2e1", "Show - s1e12 - s2e1", "Show - s1e12 - s2e1", "Show - s1e12 - s2e1",
		}},
		{"{Series Title} - {season}x{episode:00}", eps(1, 2, 3), [6]string{
			"Show - 1x01-02-03", "Show - 1x01 - 1x02 - 1x03", "Show - 1x01x02x03", "Show - 1x01-x02-x03", "Show - 1x01-03", "Show - 1x01-x03",
		}},
		{"{Series Title} - {season}x{episode:00}", eps(1, 3), [6]string{
			"Show - 1x01x03", "Show - 1x01x03", "Show - 1x01x03", "Show - 1x01x03", "Show - 1x01x03", "Show - 1x01x03",
		}},
		{"{Series Title} - {season}x{episode:00}", crossing, [6]string{
			"Show - 1x12 - 2x01", "Show - 1x12 - 2x01", "Show - 1x12 - 2x01", "Show - 1x12 - 2x01", "Show - 1x12 - 2x01", "Show - 1x12 - 2x01",
		}},
		{"{Series Title} - {absolute:000} - {Episode Title}", eps(1, 2, 3), [6]string{
			"Show - 001-002-003 - City Sushi",
			"Show - 001 - 002 - 003 - City Sushi",
			"Show - 001-002-003 - City Sushi",
			"Show - 001-002-003 - City Sushi",
			"Show - 001-003 - City Sushi",
			"Show - 001-003 - City Sushi",
		}},
		{"{Series Title} - {absolute:000}", eps(1, 3, 4), [6]string{
			"Show - 001-003-004", "Show - 001-003-004", "Show - 001-003-004", "Show - 001-003-004", "Show - 001-003-004", "Show - 001-003-004",
		}},
		{"{Series Title} - {absolute:000}", []Unit{ep(1, 1, 1), withoutAbsolute(ep(1, 2, 2))}, [6]string{
			"Show - ", "Show - ", "Show - ", "Show - ", "Show - ", "Show - ",
		}},
		{"{Series Title} [{absolute:000}]", eps(1, 2), [6]string{
			"Show [001-002]", "Show [001-002]", "Show [001-002]", "Show [001-002]", "Show [001-002]", "Show [001-002]",
		}},
		{"{Series Title} - S{season:00}E{episode:00} - {absolute:000}", eps(1, 2, 3), [6]string{
			"Show - S01E01-02-03 - 001-002-003",
			"Show - S01E01 - S01E02 - S01E03 - 001 - 002 - 003",
			"Show - S01E01E02E03 - 001-002-003",
			"Show - S01E01-E02-E03 - 001-002-003",
			"Show - S01E01-03 - 001-003",
			"Show - S01E01-E03 - 001-003",
		}},
	}
	for _, c := range cases {
		for i, style := range styles {
			if got := renderStub(t, c.template, style, c.units); got != c.want[i] {
				t.Errorf("%s in style %d = %q; want %q", c.template, style, got, c.want[i])
			}
		}
	}
}

// Sonarr's MultiEpisodeFixture at the pinned commit: South Park episodes
// 6, 7 and 8 of season 15, absolute 100 to 102, every one titled
// "City Sushi".
func TestMultiEpisodeSonarrCases(t *testing.T) {
	south := func(n int) []Unit {
		units := make([]Unit, n)
		for i := range units {
			units[i] = ep(15, int64(6+i), int64(100+i))
			units[i].Item.Set["Episode Title"] = Text("City Sushi")
		}
		return units
	}
	const standard = "{Series Title} - S{season:00}E{episode:00} - {Episode Title}"
	const anime = "{Series Title} - {absolute:000} - {Episode Title}"
	cases := []struct {
		name     string
		template string
		style    MultiEpisode
		units    []Unit
		want     string
	}{
		{"should_format_extend_multi_episode_properly", standard, Extend, south(2), "Show - S15E06-07 - City Sushi"},
		{"should_format_duplicate_multi_episode_properly", standard, Duplicate, south(2), "Show - S15E06 - S15E07 - City Sushi"},
		{"should_format_repeat_multi_episode_properly", standard, Repeat, south(2), "Show - S15E06E07 - City Sushi"},
		{"should_format_scene_multi_episode_properly", standard, Scene, south(2), "Show - S15E06-E07 - City Sushi"},
		{"should_use_dash_as_separator_when_multi_episode_style_is_extend_for_anime", anime, Extend, south(2), "Show - 100-101 - City Sushi"},
		{"should_duplicate_absolute_pattern_when_multi_episode_style_is_duplicate", anime, Duplicate, south(3), "Show - 100 - 101 - 102 - City Sushi"},
		{"should_format_range_multi_episode_properly", standard, Range, south(3), "Show - S15E06-08 - City Sushi"},
		{"should_format_range_multi_episode_anime_properly", anime, Range, south(3), "Show - 100-102 - City Sushi"},
		{"should_format_repeat_multi_episode_anime_properly", anime, Repeat, south(3), "Show - 100-101-102 - City Sushi"},
		{"should_format_single_episode_with_range_multi_episode_properly", standard, Range, south(1), "Show - S15E06 - City Sushi"},
		{"should_format_single_anime_episode_with_range_multi_episode_properly", anime, Range, south(1), "Show - 100 - City Sushi"},
		{
			"should_default_to_dash_when_serparator_is_not_set_for_absolute_number",
			"{Series Title} - {season}x{episode:00} - [{absolute:000}] - {Episode Title}", Duplicate, south(2),
			"Show - 15x06 - 15x07 - [100-101] - City Sushi",
		},
		{"should_format_prefixed_range_multi_episode_properly", standard, PrefixedRange, south(3), "Show - S15E06-E08 - City Sushi"},
		{"should_format_prefixed_range_multi_episode_anime_properly", anime, PrefixedRange, south(3), "Show - 100-102 - City Sushi"},
		{"should_format_single_episode_with_prefixed_range_multi_episode_properly", standard, PrefixedRange, south(1), "Show - S15E06 - City Sushi"},
		{"should_format_single_anime_episode_with_prefixed_range_multi_episode_properly", anime, PrefixedRange, south(1), "Show - 100 - City Sushi"},
		{"should_format_prefixed_range_multi_episode_using_episode_separator", "{Series Title} - {season:0}x{episode:00} - {Episode Title}", PrefixedRange, south(3), "Show - 15x06-x08 - City Sushi"},
		{"should_format_range_multi_episode_wrapped_in_brackets", "{Series Title} (S{season:00}E{episode:00}) {Episode Title}", Range, south(3), "Show (S15E06-08) City Sushi"},
	}
	for _, c := range cases {
		if got := renderStub(t, c.template, c.style, c.units); got != c.want {
			t.Errorf("%s: got %q; want %q", c.name, got, c.want)
		}
	}
}

func TestBlockSpanSkipsALevelTemplate(t *testing.T) {
	if got := renderStub(t, "Season {season:00}", PrefixedRange, eps(1, 2)); got != "Season 01" {
		t.Errorf("level template = %q; want %q", got, "Season 01")
	}
}

func TestJoinTitles(t *testing.T) {
	titles := func(ts ...string) []Value {
		vs := make([]Value, len(ts))
		for i, s := range ts {
			vs[i] = Text(s)
		}
		return vs
	}
	upper := strings.ToUpper
	cases := []struct {
		in    []Value
		clean func(string) string
		want  string
	}{
		{titles("Pilot (1)", "Pilot (2)"), nil, "Pilot"},
		{titles("Hello", "World"), nil, "Hello + World"},
		{titles("Hello", "World"), upper, "HELLO and WORLD"},
		{titles("Pilot (1)", "Pilot (2)"), upper, "PILOT"},
		{titles("Part 1", "Part 2"), nil, "Part 1 + Part 2"},
		{titles("Pilot (1)"), nil, "Pilot (1)"},
		{titles("Why?", "Why."), nil, "Why"},
		{[]Value{Text("Hello"), {}, Text("World")}, nil, "Hello + World"},
	}
	for _, c := range cases {
		if got := joinTitles(c.in, c.clean); got.text != c.want {
			t.Errorf("joinTitles(%v) = %q; want %q", c.in, got.text, c.want)
		}
	}
}

// The vendored Sonarr cases whose file covers several episodes: each
// title join, with every other token of the template its stated value.
func TestJoinTitlesSonarrCases(t *testing.T) {
	n := 0
	for _, file := range []string{"testdata/arr/sonarr/MultiEpisodeTitleFixture.json", "testdata/arr/sonarr/EpisodeTitleCollapseFixture.json"} {
		var f struct {
			Cases []struct {
				Template, Want, Source string
				Values                 map[string]any
				Items                  []map[string]any
			}
		}
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &f); err != nil {
			t.Fatal(file, err)
		}
		for _, c := range f.Cases {
			if len(c.Items) < 2 {
				continue
			}
			n++
			titles := make([]Value, len(c.Items))
			for i, it := range c.Items {
				titles[i] = Text(fmt.Sprint(it["Episode Title"]))
			}
			pairs := []string{"{Episode Title}", joinTitles(titles, nil).text}
			for k, v := range c.Values {
				pairs = append(pairs, "{"+k+"}", fmt.Sprint(v))
			}
			if got := strings.NewReplacer(pairs...).Replace(c.Template); got != c.Want {
				t.Errorf("%s: got %q; want %q", c.Source, got, c.Want)
			}
		}
	}
	if n < 11 {
		t.Errorf("read %d multi-episode title cases; the snapshot holds 11", n)
	}
}
