package naming

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

// arrDeclarations are SlipStream's tokens under the names the upstream
// cases use, per upstream.
var arrDeclarations = map[string]Declaration{
	"sonarr": {Tokens: []Token{
		{Name: "Series Title", Kind: KindTitle, Scope: WorkScope},
		{Name: "Series Year", Kind: KindYear, Scope: WorkScope},
		{Name: "Season", Kind: KindNumber, Scope: LevelScope, Seq: SeqOuter},
		{Name: "Season Title", Kind: KindTitle, Scope: LevelScope},
		{Name: "Episode", Kind: KindNumber, Scope: ItemScope, Seq: SeqInner},
		{Name: "Episode Title", Kind: KindTitle, Scope: ItemScope},
		{Name: "ImdbId", Kind: KindID, Scope: WorkScope},
		{Name: "TvdbId", Kind: KindID, Scope: WorkScope},
		{Name: "TmdbId", Kind: KindID, Scope: WorkScope},
	}},
	"radarr": {Tokens: []Token{
		{Name: "Movie Title", Kind: KindTitle, Scope: WorkScope},
		{Name: "Release Year", Aliases: []string{"Year"}, Kind: KindYear, Scope: WorkScope},
		{Name: "ImdbId", Kind: KindID, Scope: WorkScope},
		{Name: "TmdbId", Kind: KindID, Scope: WorkScope},
	}},
}

// arrSkipped are the upstream cases SlipStream renders differently on
// purpose, by file, method or case, with the spec's reason.
var arrSkipped = map[string]string{
	"EpisodeTitleCollapseFixture.cs":                                                  "joining several episode titles is multi-episode rendering",
	"MultiEpisodeTitleFixture.cs":                                                     "joining several episode titles is multi-episode rendering",
	"CollectionTheFixture.cs":                                                         "SlipStream declares no collection token",
	"CleanCollectionTheFixture.cs":                                                    "SlipStream declares no collection token",
	"EditionTagsFixture.cs should_always_lowercase_ordinals":                          "{Edition} prints the cut's own name, where Radarr title-cases free-text editions",
	"EditionTagsFixture.cs should_always_uppercase_special_strings":                   "{Edition} prints the cut's own name, where Radarr title-cases free-text editions",
	"EditionTagsFixture.cs should_handle_edition_tag_curly_brackets":                  "there is no brace escape, so Parse refuses {{edition-{X}}}",
	"EditionTagsFixture.cs should_conditional_hide_edition_tags #4":                   "there is no brace escape, so Parse refuses {{edition-{X}}}",
	"IdFixture.cs should_handle_imdb_tag_curly_brackets":                              "there is no brace escape, so Parse refuses {{imdb-{X}}}",
	"IdFixture.cs should_handle_tmdb_tag_curly_brackets":                              "there is no brace escape, so Parse refuses {{tmdb-{X}}}",
	"IdFixture.cs should_skip_imdb_tag_if_null #3":                                    "there is no brace escape, so Parse refuses {{imdb-{X}}}",
	"IdFixture.cs should_skip_imdb_tag_if_null #4":                                    "there is no brace escape, so Parse refuses {{imdbid-{X}}}",
	"ReservedDeviceNameFixture.cs":                                                    "a device name gets \"_\" before it, as _CON, where Sonarr replaces the dot after it",
	"GetSeriesFolderFixture.cs should_use_seriesFolderFormat_to_build_folder_name #5": "a value keeps its leading dot, and only leading spaces go",
	"GetSeriesFolderFixture.cs should_use_seriesFolderFormat_to_build_folder_name #6": "a literal keeps its leading dot, and only leading spaces go",
	"SeriesTitleFirstCharacterFixture.cs should_get_expected_folder_name_back #7":     "a value keeps its leading dot, and only leading spaces go",
	"GetSeasonFolderFixture.cs should_use_seriesFolderFormat_to_build_folder_name #1": "only separators meeting across an empty value collapse, so a title's own dot stays beside a literal one",
	"CleanTitleFixture.cs should_get_expected_title_back #12":                         "a value's own last dot stays before the extension, where Sonarr trims the separators ending a file name",
	"FileNameBuilderFixture.cs should_format_subtitle_languages_all #7":               "a trailing \"+\" prints \"--\" and nothing inside a value collapses, where Sonarr's cleanup makes it \"-\"",
}

type arrCase struct {
	Template string
	Values   map[string]any
	Settings map[string]string
	Items    []map[string]any
	Want     string
	Source   string
}

var arrColons = map[string]Colon{
	"Delete": ColonDelete, "Dash": ColonDash, "SpaceDash": ColonSpaceDash,
	"SpaceDashSpace": ColonSpaceDashSpace, "Smart": ColonSmart, "Custom": ColonCustom,
}

var editionTags = regexp.MustCompile(`(?i)(edition)[ ._-]tags`)

func TestArrCases(t *testing.T) {
	files, err := filepath.Glob("testdata/arr/*/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no arr case files: %v", err)
	}
	ran, skipped := 0, 0
	for _, file := range files {
		upstream := filepath.Base(filepath.Dir(file))
		cat, err := newCatalog(arrDeclarations[upstream])
		if err != nil {
			t.Fatal(err)
		}
		var f struct{ Cases []arrCase }
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &f); err != nil {
			t.Fatal(file, err)
		}
		for _, c := range f.Cases {
			if reason := arrSkip(c); reason != "" {
				skipped++
				continue
			}
			ran++
			got, err := renderArr(cat, c)
			if err != nil {
				t.Errorf("%s/%s: %q: %v", upstream, c.Source, c.Template, err)
			} else if got != c.Want {
				t.Errorf("%s/%s: %q with %v = %q; want %q", upstream, c.Source, c.Template, c.Values, got, c.Want)
			}
		}
	}
	if ran < 300 {
		t.Errorf("ran %d arr cases and skipped %d; the vendored files hold more", ran, skipped)
	}
}

func arrSkip(c arrCase) string {
	for source, reason := range arrSkipped {
		if c.Source == source || strings.HasPrefix(c.Source, source+" ") {
			return reason
		}
	}
	return ""
}

// renderArr renders c's template with c's values. A "\" between folders
// renders each side as its own segment.
func renderArr(cat *catalog, c arrCase) (string, error) {
	st := style{colon: arrColons[c.Settings["colon"]], custom: c.Settings["colonCustom"]}
	if st.colon == 0 {
		st.colon = ColonSmart
	}
	s, ff, err := arrSubject(cat, c.Values)
	if err != nil {
		return "", err
	}
	var out []string
	for _, seg := range strings.Split(editionTags.ReplaceAllString(c.Template, "${1}"), `\`) {
		tmpl, err := Parse(seg)
		if err != nil {
			return "", err
		}
		b, err := bind(tmpl, cat, fileField)
		if err != nil {
			return "", err
		}
		out = append(out, arrSegment(st.render(b, s, ff), c.Source))
	}
	return strings.Join(out, `\`), nil
}

// arrSegment applies the segment rule as a Scheme does: to a folder name,
// or to a file name with its extension, which the upstream cases leave out.
func arrSegment(name, source string) string {
	if strings.Contains(source, "Folder") {
		return segment(name)
	}
	return strings.TrimSuffix(segment(name+".mkv"), ".mkv")
}

func arrSubject(cat *catalog, values map[string]any) (Subject, FileFacts, error) {
	s := Subject{Work: Set{}, Units: []Unit{{Level: Set{}, Item: ItemValues{Set: Set{}}}}}
	ff := FileFacts{Facts: release.Facts{Coverage: release.Coverage{Original: release.Known(release.Language("en"))}}}
	for name, v := range values {
		if ok, err := arrFact(&ff, name, v); ok || err != nil {
			if err != nil {
				return s, ff, err
			}
			continue
		}
		e := cat.byKey[KeyOf(name)]
		if e == nil {
			return s, ff, fmt.Errorf("no token for the upstream value %q", name)
		}
		val := Text(fmt.Sprint(v))
		if n, isNumber := v.(float64); isNumber && e.Kind != KindID {
			val = Number(int64(n))
		}
		map[Scope]Set{WorkScope: s.Work, LevelScope: s.Units[0].Level, ItemScope: s.Units[0].Item.Set}[e.Scope][e.Name] = val
	}
	return s, ff, nil
}

// arrFact sets the file fact an upstream value names, and reports
// whether name is one.
func arrFact(ff *FileFacts, name string, v any) (bool, error) {
	text := fmt.Sprint(v)
	switch name {
	case "Quality Title", "Quality Full":
		q, err := release.ParseQuality(strings.ToLower(text))
		ff.Facts.Quality = release.Known(q)
		return true, err
	case "Release Group":
		ff.Facts.Group = release.Known(text)
	case "Edition Tags":
		ff.Cut = text
	case "Custom Formats":
		list, _ := v.([]any)
		for _, n := range list {
			ff.Formats = append(ff.Formats, fmt.Sprint(n))
		}
	case "MediaInfo AudioLanguages":
		ls, err := arrLanguages(text)
		ff.Facts.Languages = release.Known(ls)
		return true, err
	case "MediaInfo SubtitleLanguages":
		ls, err := arrLanguages(text)
		for _, l := range ls {
			ff.Media.Subtitles = append(ff.Media.Subtitles, media.Subtitle{Language: l})
		}
		return true, err
	default:
		return false, nil
	}
	return true, nil
}

func arrLanguages(s string) ([]release.Language, error) {
	var out []release.Language
	for _, code := range strings.Split(s, "/") {
		l, err := release.ParseLanguage(code)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}
