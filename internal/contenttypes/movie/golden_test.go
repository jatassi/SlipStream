package movie_test

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttypes/movie"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/release/golden"
)

var names = new(movie.Movie)

func title(name string) release.Read { return release.Title(name, names, release.BuiltIn()) }

func postTitle(r golden.Row) release.Read { return title(r.String("postTitle")) }

func titled(r golden.Row) error {
	return golden.Same("title", golden.FirstTitle(postTitle(r).Name), r.String("title"))
}

func year(read release.Read, want int) error {
	y := release.Fact[int]{}
	if want != 0 {
		y = release.Known(want)
	}
	return golden.Same("year", read.Name.Year, y)
}

func group(read release.Read, want string) error {
	return golden.Same("group", read.Facts.Group, release.Known(want))
}

func edition(read release.Read, want string) error {
	e := release.Fact[string]{}
	if want != "" {
		e = release.Known(want)
	}
	return golden.Same("edition", read.Facts.Edition, e)
}

func idTag(kind string, value func(golden.Row) string) golden.Check {
	return func(r golden.Row) error {
		return golden.Same(kind, postTitle(r).Facts.IDs[kind], value(r))
	}
}

// languages is check on what the name reads as, or, where an override
// says what SlipStream reads, whether it reads exactly that.
func languages(check func(release.Fact[[]release.Language]) error) golden.Check {
	return func(r golden.Row) error {
		got := postTitle(r).Facts.Languages
		var ours []string
		if r.Reads(&ours) {
			return golden.Languages(got, ours...)
		}
		return check(got)
	}
}

func has(langs ...string) golden.Check {
	return languages(func(got release.Fact[[]release.Language]) error { return golden.HasLanguages(got, langs...) })
}

func exactly(langs ...string) golden.Check {
	return languages(func(got release.Fact[[]release.Language]) error { return golden.Languages(got, langs...) })
}

func first(lang string) golden.Check {
	return languages(func(got release.Fact[[]release.Language]) error {
		langs, _ := got.Get()
		return golden.Languages(release.Known(langs[:min(len(langs), 1)]), lang)
	})
}

func unreadTitle(param string) golden.Check {
	return func(r golden.Row) error { return golden.Unread(title(r.String(param)).Name) }
}

func TestGoldenParser(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/radarr/ParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_movie_title":       titled,
			"should_parse_movie_folder_name": titled,
			"should_parse_anime_movie_title": func(r golden.Row) error {
				read := postTitle(r)
				return errors.Join(titled(r), group(read, r.String("releaseGroup")), year(read, r.Int("year")))
			},
			"should_parse_anime_movie_title_without_year": func(r golden.Row) error {
				return errors.Join(titled(r), group(postTitle(r), r.String("releaseGroup")))
			},
			"should_parse_german_movie": func(r golden.Row) error {
				read := postTitle(r)
				return errors.Join(titled(r), edition(read, r.String("edition")), year(read, r.Int("year")))
			},
			"should_parse_movie_alternative_titles": func(r golden.Row) error {
				return golden.Same("titles", postTitle(r).Name.Titles, r.Strings("parsedTitles"))
			},
			"should_parse_movie_year":                  func(r golden.Row) error { return year(postTitle(r), r.Int("year")) },
			"should_parse_tmdb_id":                     idTag("tmdbid", func(r golden.Row) string { return strconv.Itoa(r.Int("tmdbId")) }),
			"should_parse_imdb_in_title":               idTag("imdbid", func(r golden.Row) string { return r.String("imdb") }),
			"should_not_parse_wrong_language_in_title": exactly("Unknown"),
		},
		Unported: map[string]string{
			"should_parse_hardcoded_subs": "SlipStream reads no hardcoded subtitles from a name.",
			"should_normalize_imdbid":     "SlipStream reads IMDb ids only from a name.",
		},
	})
}

func TestGoldenEdition(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/radarr/EditionParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_edition":     func(r golden.Row) error { return edition(postTitle(r), r.String("edition")) },
			"should_not_parse_edition": func(r golden.Row) error { return edition(postTitle(r), "") },
		},
	})
}

func TestGoldenCrap(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/radarr/CrapParserFixture.json",
		Checks: map[string]golden.Check{
			"should_not_parse_crap":                             unreadTitle("title"),
			"should_not_parse_file_name_without_proper_spacing": unreadTitle("fileName"),
			"should_not_parse_invalid_release_name":             unreadTitle("fileName"),
			"should_not_parse_random": func(r golden.Row) error {
				var errs []error
				length := r.Int("length")
				rng := rand.New(rand.NewPCG(uint64(length), 0)) //nolint:gosec // a fixed draw the test replays
				for _, name := range golden.RandomNames(rng, length) {
					if err := golden.Unread(title(name).Name); err != nil {
						errs = append(errs, fmt.Errorf("%s: %w", name, err))
					}
				}
				return errors.Join(errs...)
			},
		},
	})
}

func TestGoldenNormalizeTitle(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path:   "testdata/upstream/radarr/NormalizeTitleFixture.json",
		Checks: golden.Cleaning(names.CleanTitle),
	})
}

const subtitles = "SlipStream reads no subtitle file names."

func TestGoldenLanguage(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/radarr/LanguageParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_language_english":                                                 has("English"),
			"should_parse_language_unknown":                                                 has("Unknown"),
			"should_parse_language_french":                                                  has("French"),
			"should_parse_language_french_english":                                          has("French", "English"),
			"should_parse_language_spanish":                                                 has("Spanish"),
			"should_parse_language_german":                                                  has("German"),
			"should_parse_language_italian":                                                 has("Italian"),
			"should_parse_language_danish":                                                  has("Danish"),
			"should_parse_language_dutch":                                                   has("Dutch"),
			"should_parse_language_japanese":                                                has("Japanese"),
			"should_parse_language_icelandic":                                               has("Icelandic"),
			"should_parse_language_chinese":                                                 has("Chinese"),
			"should_parse_language_russian":                                                 has("Russian"),
			"should_parse_language_romanian":                                                has("Romanian"),
			"should_parse_language_hindi":                                                   has("Hindi"),
			"should_parse_language_thai":                                                    has("Thai"),
			"should_parse_language_bulgarian":                                               has("Bulgarian"),
			"should_parse_language_brazilian_portuguese":                                    has("PortugueseBR"),
			"should_parse_language_polish":                                                  has("Polish"),
			"should_parse_language_polish_subbed":                                           has("Unknown"),
			"should_parse_language_vietnamese":                                              has("Vietnamese"),
			"should_parse_language_swedish":                                                 has("Swedish"),
			"should_parse_language_norwegian":                                               has("Norwegian"),
			"should_parse_language_finnish":                                                 has("Finnish"),
			"should_parse_language_turkish":                                                 has("Turkish"),
			"should_parse_language_portuguese":                                              has("Portuguese"),
			"should_parse_language_flemish":                                                 has("Flemish"),
			"should_parse_language_greek":                                                   has("Greek"),
			"should_parse_language_korean":                                                  has("Korean"),
			"should_parse_language_hungarian":                                               has("Hungarian"),
			"should_parse_language_hebrew":                                                  has("Hebrew"),
			"should_parse_language_lithuanian":                                              has("Lithuanian"),
			"should_parse_language_czech":                                                   has("Czech"),
			"should_parse_language_arabic":                                                  has("Arabic"),
			"should_parse_language_ukrainian":                                               has("Ukrainian"),
			"should_parse_language_ukrainian_multi":                                         has("Ukrainian"),
			"should_parse_language_persian":                                                 has("Persian"),
			"should_parse_language_bengali":                                                 has("Bengali"),
			"should_parse_language_telugu":                                                  has("Telugu"),
			"should_parse_language_slovak":                                                  has("Slovak"),
			"should_parse_language_latvian":                                                 has("Latvian"),
			"should_parse_language_spanish_latino":                                          first("SpanishLatino"),
			"should_parse_language_catalan":                                                 has("Catalan"),
			"should_parse_language_malayalam":                                               has("Malayalam"),
			"should_parse_language_kannada":                                                 has("Kannada"),
			"should_parse_language_albanian":                                                has("Albanian"),
			"should_parse_language_afrikaans":                                               has("Afrikaans"),
			"should_parse_language_marathi":                                                 has("Marathi"),
			"should_parse_language_tagalog":                                                 has("Tagalog"),
			"should_parse_language_urdu":                                                    has("Urdu"),
			"should_parse_language_romansh":                                                 has("Romansh"),
			"should_parse_language_mongolian":                                               has("Mongolian"),
			"should_parse_language_georgian":                                                has("Georgian"),
			"should_parse_language_irish":                                                   has("Irish"),
			"should_add_original_language_to_german_release_with_dl_tag":                    exactly("German", "Original"),
			"should_not_add_original_language_to_german_release_when_title_contains_web_dl": exactly("German"),
			"should_parse_original_title_from_release_name":                                 exactly("Original"),
			"should_add_original_language_and_english_to_german_release_with_ml_tag":        exactly("German", "Original", "English"),
		},
		Unported: map[string]string{
			"should_parse_subtitle_language_english": subtitles,
			"should_parse_subtitle_language":         subtitles,
			"should_parse_subtitle_language_unknown": subtitles,
			"should_parse_title_and_tags":            subtitles,
			"should_not_parse_false_title":           subtitles,
		},
	})
}
